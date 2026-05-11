package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/namejlt/AgentFlowPro/internal/crypto"
	"github.com/namejlt/AgentFlowPro/internal/datasource"
	"github.com/namejlt/AgentFlowPro/internal/llm"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/jsonutil"
	"github.com/namejlt/AgentFlowPro/internal/templatex"
)

func (r *Runner) executeNode(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, g *Graph, node FlowNode, store *runStore, emit func(string, any)) (string, string, error) {
	stepID := uuid.New()
	ts := model.TaskStep{
		ID:       stepID,
		TaskID:   taskID,
		NodeID:   node.ID,
		NodeType: node.Type,
		Status:   "running",
		StartedAt: ptrTime(time.Now()),
	}
	_ = r.Store.DB.Create(&ts).Error
	store.log("INFO", node.ID, "node start "+node.Type)

	var out string
	var nextLabel string
	var err error

	switch node.Type {
	case "start":
		out = "started"
	case "end":
		titleTpl, _ := node.Data["title_template"].(string)
		title := templatex.Render(titleTpl, store.flatVars())
		store.setKV("report:title", title)
		out = title
	case "agent_run":
		aid := uuidFromAny(node.Data["agent_id"])
		if aid == uuid.Nil {
			err = fmt.Errorf("agent_run missing agent_id")
			break
		}
		ts.AgentID = &aid
		_ = r.Store.DB.Model(&ts).Update("agent_id", aid).Error
		out, err = r.runAgentNode(ctx, taskID, wf, node, store, aid, emit)
	case "parallel":
		out, err = r.runParallel(ctx, taskID, wf, node, store, emit)
	case "debate":
		out, err = r.runDebate(ctx, taskID, wf, node, store, emit)
	case "cross_validate":
		out, err = r.runCrossValidate(ctx, wf, node, store, emit)
	case "risk_review":
		out, err = r.runRiskReview(ctx, wf, node, store, emit)
	case "condition":
		expr, _ := node.Data["expression"].(string)
		label := templatex.Render(expr, store.flatVars())
		store.setKV("__branch__", label)
		out = label
		nextLabel = label
	case "summarize":
		out, err = r.runSummarize(ctx, wf, node, store, emit)
	case "transform":
		out, err = r.runTransform(node, store)
	default:
		err = fmt.Errorf("unknown node type: %s", node.Type)
	}

	fin := time.Now()
	st := "completed"
	if err != nil {
		st = "failed"
	}
	outB := jsonutil.MustMarshal(map[string]any{"text": out})
	_ = r.Store.DB.Model(&model.TaskStep{}).Where("id = ?", stepID).Updates(map[string]any{
		"status":        st,
		"finished_at":   fin,
		"output":        outB,
		"error_message": errString(err),
	}).Error
	store.log("INFO", node.ID, "node finish "+st)
	return out, nextLabel, err
}

func errString(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func ptrTime(t time.Time) *time.Time { return &t }

func uuidFromAny(v any) uuid.UUID {
	s := strings.TrimSpace(fmt.Sprint(v))
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func (r *Runner) decryptHeaders(sealed string) (map[string]string, error) {
	if sealed == "" {
		return map[string]string{}, nil
	}
	b, err := crypto.Open(r.Key, sealed)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Runner) decryptAuth(sealed string) (map[string]any, error) {
	if sealed == "" {
		return map[string]any{}, nil
	}
	b, err := crypto.Open(r.Key, sealed)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Runner) decryptAPIKey(sealed string) (string, error) {
	b, err := crypto.Open(r.Key, sealed)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *Runner) resolveModelID(wf *model.Workflow, agent *model.Agent) (*model.LLMModel, error) {
	var m model.LLMModel
	switch {
	case agent.LLMModelID != nil:
		if err := r.Store.DB.First(&m, "id = ?", *agent.LLMModelID).Error; err != nil {
			return nil, err
		}
		return &m, nil
	case wf.DefaultModelID != nil:
		if err := r.Store.DB.First(&m, "id = ?", *wf.DefaultModelID).Error; err != nil {
			return nil, err
		}
		return &m, nil
	default:
		if err := r.Store.DB.Where("is_default = ? AND deleted_at IS NULL", true).First(&m).Error; err != nil {
			return nil, err
		}
		return &m, nil
	}
}

func (r *Runner) runAgentNode(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, node FlowNode, store *runStore, agentID uuid.UUID, emit func(string, any)) (string, error) {
	var ag model.Agent
	if err := r.Store.DB.First(&ag, "id = ?", agentID).Error; err != nil {
		return "", err
	}
	if !ag.Enabled {
		return "", fmt.Errorf("agent disabled")
	}
	mo, err := r.resolveModelID(wf, &ag)
	if err != nil {
		return "", err
	}
	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", err
	}

	// datasource
	dsText := ""
	if ag.DataSourceID != nil {
		var ds model.DataSource
		if err := r.Store.DB.First(&ds, "id = ?", *ag.DataSourceID).Error; err == nil {
			params, err := datasource.ResolveParams(ds.ParamsSchema, store.global, map[string]any{})
			if err != nil {
				return "", err
			}
			res, err := r.DS.Execute(ctx, &ds, datasource.ExecuteInput{GlobalVars: store.global, Params: params}, r.decryptHeaders, r.decryptAuth)
			if err != nil {
				return "", err
			}
			dsText = res.Extracted
		}
	}

	extras := map[string]string{
		"datasource.result": dsText,
	}
	for k, v := range store.flatVars() {
		extras[k] = v
	}
	prompt := templatex.Render(ag.SystemPrompt, templatex.MergeStringMap(templatex.BuildPromptVarMap(store.global, nil), extras))

	messages := []llm.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "请根据系统指令输出分析结果。"},
	}

	stepID := uuid.NewString()
	emit("agent_stream_start", map[string]any{"step_id": stepID, "agent_id": agentID.String(), "agent_name": ag.Name})

	var acc strings.Builder
	stream := mo.StreamEnabled
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint:  mo.Endpoint,
		APIKey:    apiKey,
		Model:     mo.ModelID,
		Messages:  messages,
		Temp:      mo.Temperature,
		MaxTokens: mo.MaxTokens,
		Timeout:   time.Duration(mo.TimeoutMS) * time.Millisecond,
		Retries:   mo.RetryCount,
		Stream:    stream,
		OnChunk: func(c llm.StreamChunk) error {
			if c.Delta != "" {
				acc.WriteString(c.Delta)
				emit("agent_stream_chunk", map[string]any{"step_id": stepID, "chunk": c.Delta, "accumulated": acc.String()})
			}
			return nil
		},
	})
	if err != nil {
		emit("agent_stream_end", map[string]any{"step_id": stepID, "full_output": acc.String(), "tokens_used": 0})
		return "", err
	}
	out := resp.Content
	if stream && out == "" {
		out = acc.String()
	}
	emit("agent_stream_end", map[string]any{"step_id": stepID, "full_output": out, "tokens_used": resp.Tokens})
	store.setAgentOutput(agentID, out)
	return out, nil
}

func (r *Runner) runParallel(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	mode, _ := node.Data["mode"].(string)
	wait := "all"
	if v, ok := node.Data["wait_strategy"].(string); ok {
		wait = v
	}
	var parts []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	var eMu sync.Mutex
	var errList []error

	if mode == "agents" {
		ids := toUUIDSlice(node.Data["agent_ids"])
		if len(ids) == 0 {
			return "", fmt.Errorf("parallel agents missing agent_ids")
		}
		for _, aid := range ids {
			aid := aid
			wg.Add(1)
			go func() {
				defer wg.Done()
				txt, err := r.runAgentNode(ctx, taskID, wf, FlowNode{ID: node.ID + ":" + aid.String(), Type: "agent_run", Data: map[string]any{"agent_id": aid.String()}}, store, aid, emit)
				if err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				mu.Lock()
				parts = append(parts, fmt.Sprintf("【%s】\n%s", aid.String(), txt))
				mu.Unlock()
			}()
		}
		wg.Wait()
	} else {
		// graph downstream: run all successor agent_run nodes in parallel by graph
		// handled by workflow graph itself usually; here treat as agents list from child_node_ids
		ids := toUUIDSlice(node.Data["child_agent_ids"])
		for _, aid := range ids {
			aid := aid
			wg.Add(1)
			go func() {
				defer wg.Done()
				txt, err := r.runAgentNode(ctx, taskID, wf, FlowNode{ID: node.ID + ":" + aid.String(), Type: "agent_run", Data: map[string]any{"agent_id": aid.String()}}, store, aid, emit)
				if err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				mu.Lock()
				parts = append(parts, txt)
				mu.Unlock()
			}()
		}
		wg.Wait()
	}
	if wait == "any" {
		// simplified: still wait all goroutines
	}
	if len(errList) > 0 {
		return "", errList[0]
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func toUUIDSlice(v any) []uuid.UUID {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []uuid.UUID
	for _, it := range arr {
		id := uuidFromAny(it)
		if id != uuid.Nil {
			out = append(out, id)
		}
	}
	return out
}

func (r *Runner) runDebate(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	ids := toUUIDSlice(node.Data["agent_ids"])
	if len(ids) == 0 {
		return "", fmt.Errorf("debate missing agent_ids")
	}
	maxR := 3
	if v, ok := node.Data["max_rounds"].(float64); ok {
		maxR = int(v)
	}
	if maxR < 1 {
		maxR = 1
	}
	if maxR > 5 {
		maxR = 5
	}
	stopConsensus, _ := node.Data["stop_on_consensus"].(bool)

	lastRound := map[string]string{}
	for round := 1; round <= maxR; round++ {
		roundOutputs := map[string]string{}
		var wg sync.WaitGroup
		var mu sync.Mutex
		var eMu sync.Mutex
		var errList []error

		for _, aid := range ids {
			aid := aid
			wg.Add(1)
			go func() {
				defer wg.Done()
				// build context from other agents last round
				others := ""
				if round > 1 {
					for oid, txt := range lastRound {
						if oid == aid.String() {
							continue
						}
						others += fmt.Sprintf("【对手 %s 上轮结论】\n%s\n\n", oid, txt)
					}
				}
				var ag model.Agent
				if err := r.Store.DB.First(&ag, "id = ?", aid).Error; err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				mo, err := r.resolveModelID(wf, &ag)
				if err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
				if err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				prompt := templatex.Render(ag.SystemPrompt, templatex.MergeStringMap(templatex.BuildPromptVarMap(store.global, nil), map[string]string{
					"debate.others_last": others,
					"debate.round":       fmt.Sprint(round),
				}))
				msgs := []llm.Message{{Role: "system", Content: prompt}, {Role: "user", Content: "请输出本轮辩论结论。"}}
				resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
					Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
					Temp: mo.Temperature, MaxTokens: mo.MaxTokens, Timeout: time.Duration(mo.TimeoutMS) * time.Millisecond,
					Retries: mo.RetryCount, Stream: false,
				})
				if err != nil {
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				mu.Lock()
				roundOutputs[aid.String()] = resp.Content
				mu.Unlock()
			}()
		}
		wg.Wait()
		if len(errList) > 0 {
			return "", errList[0]
		}
		lastRound = roundOutputs
		store.appendDebate(map[string]any{"node_id": node.ID, "round": round, "outputs": roundOutputs})
		emit("debate_round", map[string]any{"node_id": node.ID, "round": round, "agent_outputs": roundOutputs})
		if stopConsensus && consensus(roundOutputs) {
			break
		}
	}
	var sb strings.Builder
	for aid, txt := range lastRound {
		sb.WriteString(fmt.Sprintf("【%s】\n%s\n\n", aid, txt))
	}
	return sb.String(), nil
}

func consensus(m map[string]string) bool {
	if len(m) == 0 {
		return false
	}
	var first string
	for _, v := range m {
		first = strings.TrimSpace(v)
		break
	}
	for _, v := range m {
		if strings.TrimSpace(v) != first {
			return false
		}
	}
	return true
}

func (r *Runner) runCrossValidate(ctx context.Context, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	ids := toUUIDSlice(node.Data["agent_ids"])
	var sb strings.Builder
	for _, aid := range ids {
		txt := store.getString("agent:" + aid.String() + ":output")
		sb.WriteString(fmt.Sprintf("【%s】\n%s\n", aid.String(), txt))
	}
	// optional LLM summarization
	sys := "你是交叉验证专家，请检测上述结论的一致性、矛盾点并输出结构化结论。"
	if v, ok := node.Data["prompt"].(string); ok && v != "" {
		sys = templatex.Render(v, store.flatVars())
	}
	mo, err := r.pickAnyModel(wf)
	if err != nil {
		return sb.String(), nil
	}
	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", err
	}
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: sb.String()}}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.2, MaxTokens: mo.MaxTokens, Timeout: time.Duration(mo.TimeoutMS) * time.Millisecond,
		Retries: mo.RetryCount, Stream: false,
	})
	if err != nil {
		return "", err
	}
	_ = emit
	return resp.Content, nil
}

func (r *Runner) runRiskReview(ctx context.Context, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	sys := "你是风险管理专家，请输出风险矩阵与等级（高/中/低）及建议。"
	if v, ok := node.Data["prompt"].(string); ok && v != "" {
		sys = templatex.Render(v, store.flatVars())
	}
	body := store.getString("node:" + fmt.Sprint(node.Data["upstream"]) + ":output")
	if body == "" {
		body = strings.Join(collectAgentOutputs(store), "\n")
	}
	mo, err := r.pickAnyModel(wf)
	if err != nil {
		return "", err
	}
	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", err
	}
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: body}}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.2, MaxTokens: mo.MaxTokens, Timeout: time.Duration(mo.TimeoutMS) * time.Millisecond,
		Retries: mo.RetryCount, Stream: false,
	})
	if err != nil {
		return "", err
	}
	store.appendRisk(map[string]any{"node_id": node.ID, "text": resp.Content})
	_ = emit
	return resp.Content, nil
}

func collectAgentOutputs(store *runStore) []string {
	_, _, _, _, ag := store.snapshot()
	var lines []string
	for k, v := range ag {
		lines = append(lines, fmt.Sprintf("【%s】\n%v", k, v))
	}
	return lines
}

func (r *Runner) pickAnyModel(wf *model.Workflow) (*model.LLMModel, error) {
	if wf.DefaultModelID != nil {
		var m model.LLMModel
		if err := r.Store.DB.First(&m, "id = ?", *wf.DefaultModelID).Error; err == nil {
			return &m, nil
		}
	}
	var m model.LLMModel
	if err := r.Store.DB.Where("deleted_at IS NULL").Order("is_default desc").First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Runner) runSummarize(ctx context.Context, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	sys := "你是报告汇总专家，请输出最终 Markdown 报告。"
	if v, ok := node.Data["prompt"].(string); ok && v != "" {
		sys = templatex.Render(v, store.flatVars())
	}
	body := strings.Join(collectAgentOutputs(store), "\n\n")
	mo, err := r.pickAnyModel(wf)
	if err != nil {
		return "", err
	}
	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", err
	}
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: body}}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.4, MaxTokens: mo.MaxTokens, Timeout: time.Duration(mo.TimeoutMS) * time.Millisecond,
		Retries: mo.RetryCount, Stream: false,
	})
	_ = emit
	store.setKV("report:md", resp.Content)
	return resp.Content, nil
}

func (r *Runner) runTransform(node FlowNode, store *runStore) (string, error) {
	rules, ok := node.Data["map"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("transform missing data.map")
	}
	flat := store.flatVars()
	out := map[string]string{}
	for k, v := range rules {
		out[k] = templatex.Render(fmt.Sprint(v), flat)
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func (r *Runner) finalizeSuccess(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, t *model.Task, store *runStore, title string, dur int64) error {
	glob, logs, debate, risk, agents := store.snapshot()
	md := store.getString("report:md")
	if strings.TrimSpace(md) == "" {
		md = store.getString("report:title")
	}
	rep := model.Report{
		ID:            uuid.New(),
		TaskID:        taskID,
		WorkflowID:    wf.ID,
		OwnerID:       t.OwnerID,
		Title:         title,
		ContentMD:     md,
		AgentOutputs:  jsonutil.MustMarshal(agents),
		DebateLogs:    jsonutil.MustMarshal(debate),
		RiskReviews:   jsonutil.MustMarshal(risk),
		ExecLogs:      jsonutil.MustMarshal(logs),
		InputSnapshot: jsonutil.MustMarshal(glob),
		Status:        "completed",
		Archived:      false,
		DurationMS:    &dur,
	}
	if err := r.Store.DB.Create(&rep).Error; err != nil {
		return err
	}
	_ = r.Store.DB.Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":      "completed",
		"finished_at": time.Now(),
		"duration_ms": dur,
		"report_id":   rep.ID,
	}).Error
	_ = r.Store.DB.Model(&model.Workflow{}).Where("id = ?", wf.ID).Updates(map[string]any{
		"last_run_at": time.Now(),
		"run_count":   gorm.Expr("run_count + ?", 1),
	}).Error
	r.Hub.PublishJSON(taskID, "task_complete", map[string]any{
		"task_id": taskID.String(), "report_id": rep.ID.String(), "duration": dur,
	})
	return nil
}
