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

// executeNode executes a single workflow node and returns its output.
func (r *Runner) executeNode(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, g *Graph, node FlowNode, store *runStore, emit func(string, any)) (string, string, error) {
	stepID := uuid.New()
	ts := model.TaskStep{
		ID:        stepID,
		TaskID:    taskID,
		NodeID:    node.ID,
		NodeType:  node.Type,
		Status:    "running",
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
		store.log("INFO", node.ID, "workflow started")
	case "end":
		titleTpl, _ := node.Data["title_template"].(string)
		title := templatex.Render(titleTpl, store.flatVars())
		if title == "" {
			title = wf.Name + " 报告"
		}
		store.setKV("report:title", title)
		out = title
		store.log("INFO", node.ID, "workflow finished")
	case "agent_run":
		aid := uuidFromAny(node.Data["agent_id"])
		if aid == uuid.Nil {
			err = fmt.Errorf("agent_run node missing agent_id")
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
		if expr == "" {
			expr, _ = node.Data["condition"].(string)
		}
		label := templatex.Render(expr, store.flatVars())
		store.setKV("__branch__", label)
		out = label
		nextLabel = label
		store.log("INFO", node.ID, fmt.Sprintf("condition evaluated to: %s", label))
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

// resolveModelID resolves the LLM model for an agent, falling back to workflow default.
func (r *Runner) resolveModelID(wf *model.Workflow, agent *model.Agent) (*model.LLMModel, error) {
	var m model.LLMModel
	switch {
	case agent.LLMModelID != nil:
		if err := r.Store.DB.First(&m, "id = ?", *agent.LLMModelID).Error; err != nil {
			return nil, fmt.Errorf("agent model not found: %w", err)
		}
		return &m, nil
	case wf.DefaultModelID != nil:
		if err := r.Store.DB.First(&m, "id = ?", *wf.DefaultModelID).Error; err != nil {
			return nil, fmt.Errorf("workflow default model not found: %w", err)
		}
		return &m, nil
	default:
		if err := r.Store.DB.Where("is_default = ? AND deleted_at IS NULL", true).First(&m).Error; err != nil {
			return nil, fmt.Errorf("no default model found: %w", err)
		}
		return &m, nil
	}
}

// runAgentNode executes an agent_run node: fetches data, renders prompt, calls LLM.
func (r *Runner) runAgentNode(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, node FlowNode, store *runStore, agentID uuid.UUID, emit func(string, any)) (string, error) {
	var ag model.Agent
	if err := r.Store.DB.First(&ag, "id = ?", agentID).Error; err != nil {
		return "", fmt.Errorf("agent not found: %w", err)
	}
	if !ag.Enabled {
		return "", fmt.Errorf("agent %s is disabled", ag.Name)
	}

	mo, err := r.resolveModelID(wf, &ag)
	if err != nil {
		return "", err
	}

	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt api key failed: %w", err)
	}

	// Execute datasources if bound
	dsText := ""
	var dsIDs []uuid.UUID
	_ = json.Unmarshal(ag.DataSourceIDs, &dsIDs)
	if len(dsIDs) == 0 {
		store.log("INFO", node.ID, "no datasource bound to this agent")
	}
	for _, dsID := range dsIDs {
		var ds model.DataSource
		if err := r.Store.DB.First(&ds, "id = ?", dsID).Error; err != nil {
			store.log("WARN", node.ID, fmt.Sprintf("datasource id=%s not found: %v", dsID, err))
			continue
		}
		allVars := store.mergedVars()
		store.log("INFO", node.ID, fmt.Sprintf("executing datasource %s (%s) with allVars keys: %v", ds.Name, dsID, mapKeys(allVars)))
		params, err := datasource.ResolveParams(ds.ParamsSchema, allVars, map[string]any{})
		if err != nil {
			store.log("WARN", node.ID, fmt.Sprintf("resolve datasource params failed: %v", err))
			continue
		}
		res, err := r.DS.Execute(ctx, &ds, datasource.ExecuteInput{GlobalVars: allVars, Params: params}, r.decryptHeaders, r.decryptAuth)
		if err != nil {
			store.log("WARN", node.ID, fmt.Sprintf("datasource execution failed: %v", err))
			continue
		}
		dsData := res.Extracted
		store.log("INFO", node.ID, fmt.Sprintf("datasource %s extracted %d chars (cache=%v)", ds.Name, len(dsData), res.FromCache))
		if len(dsData) == 0 {
			store.log("WARN", node.ID, fmt.Sprintf("datasource %s returned empty content", ds.Name))
			continue
		}
		if len(dsText) > 0 {
			dsText += "\n\n---\n\n"
		}
		dsText += fmt.Sprintf("【%s】\n%s", ds.Name, dsData)
	}

	// If no datasource data, gather accumulated agent outputs as fallback context
	if dsText == "" {
		prevOutputs := collectAgentOutputs(store)
		if len(prevOutputs) > 0 {
			dsText = strings.Join(prevOutputs, "\n\n---\n\n")
			store.log("INFO", node.ID, fmt.Sprintf("using %d previous agent outputs as context (%d chars)", len(prevOutputs), len(dsText)))
		}
	}

	// Build prompt variables from globals + stored node outputs
	extras := map[string]string{
		"datasource.result": dsText,
	}
	for k, v := range store.flatVars() {
		extras[k] = v
	}
	prompt := templatex.Render(ag.SystemPrompt, templatex.MergeStringMap(templatex.BuildPromptVarMap(store.global, nil), extras))

	// Build messages: system prompt + datasource data injected as user content
	messages := []llm.Message{
		{Role: "system", Content: prompt},
	}
	currentDateStr := store.getString("report_date")
	if currentDateStr == "" {
		currentDateStr = time.Now().Format("2006年01月02日")
	}
	userContent := fmt.Sprintf("[系统时间：%s]\n请根据系统指令输出分析结果。", currentDateStr)
	if dsText != "" {
		userContent = fmt.Sprintf("[系统时间：%s]\n以下是本次分析所需的数据:\n\n%s\n\n请根据系统指令输出分析结果。", currentDateStr, dsText)
	}
	messages = append(messages, llm.Message{Role: "user", Content: userContent})

	stepID := uuid.NewString()
	emit("agent_stream_start", map[string]any{
		"step_id":    stepID,
		"agent_id":   agentID.String(),
		"agent_name": ag.Name,
	})

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
				emit("agent_stream_chunk", map[string]any{
					"step_id":     stepID,
					"chunk":       c.Delta,
					"accumulated": acc.String(),
				})
			}
			return nil
		},
	})
	if err != nil {
		emit("agent_stream_end", map[string]any{
			"step_id":     stepID,
			"full_output": acc.String(),
			"tokens_used": 0,
			"error":       err.Error(),
		})
		return "", err
	}
	out := resp.Content
	if stream && out == "" {
		out = acc.String()
	}
	emit("agent_stream_end", map[string]any{
		"step_id":     stepID,
		"full_output": out,
		"tokens_used": resp.Tokens,
	})
	store.setAgentOutput(agentID, out)
	return out, nil
}

// runParallel executes multiple agents in parallel.
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
				txt, err := r.runAgentNode(ctx, taskID, wf, FlowNode{
					ID:   node.ID + ":" + aid.String(),
					Type: "agent_run",
					Data: map[string]any{"agent_id": aid.String()},
				}, store, aid, emit)
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
		ids := toUUIDSlice(node.Data["child_agent_ids"])
		for _, aid := range ids {
			aid := aid
			wg.Add(1)
			go func() {
				defer wg.Done()
				txt, err := r.runAgentNode(ctx, taskID, wf, FlowNode{
					ID:   node.ID + ":" + aid.String(),
					Type: "agent_run",
					Data: map[string]any{"agent_id": aid.String()},
				}, store, aid, emit)
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
	if wait == "any" && len(parts) > 0 {
		// Return first successful result
		return parts[0], nil
	}
	if len(errList) > 0 {
		return "", errList[0]
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func toUUIDSlice(v any) []uuid.UUID {
	arr, ok := v.([]any)
	if !ok {
		// Try string slice
		if sarr, ok := v.([]string); ok {
			var out []uuid.UUID
			for _, s := range sarr {
				id := uuidFromAny(s)
				if id != uuid.Nil {
					out = append(out, id)
				}
			}
			return out
		}
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

// runDebate executes a multi-round debate between agents.
func (r *Runner) runDebate(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	ids := toUUIDSlice(node.Data["agent_ids"])
	if len(ids) == 0 {
		return "", fmt.Errorf("debate missing agent_ids")
	}
	maxR := 3
	if v, ok := node.Data["max_rounds"].(float64); ok {
		maxR = int(v)
	}
	if v, ok := node.Data["max_rounds"].(int); ok {
		maxR = v
	}
	if maxR < 1 {
		maxR = 1
	}
	if maxR > 5 {
		maxR = 5
	}
	stopConsensus, _ := node.Data["stop_on_consensus"].(bool)

	// Gather accumulated agent outputs as debate context
	debateContext := strings.Join(collectAgentOutputs(store), "\n\n---\n\n")
	if debateContext == "" {
		debateContext = "（暂无前置分析数据）"
	}
	store.log("INFO", node.ID, fmt.Sprintf("debate starting with %d chars of context from %d agents", len(debateContext), len(collectAgentOutputs(store))))

	lastRound := map[string]string{}
	lastRoundNames := map[string]string{}
	agentNames := map[string]string{} // agent_id -> name, loaded once
	for _, aid := range ids {
		var ag model.Agent
		if err := r.Store.DB.First(&ag, "id = ?", aid).Error; err == nil {
			agentNames[aid.String()] = ag.Name
		}
	}
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
				aidStr := aid.String()
				// Build context from previous rounds + own previous output
				contextParts := []string{}
				if round == 1 {
					// Round 1: pass all upstream analysis data as debate topic
					if debateContext != "" {
						contextParts = append(contextParts, fmt.Sprintf("【前置分析数据】\n%s", debateContext))
					}
				} else {
					// Rounds 2+: pass all other agents' last round + own last round + original analysis
					for oid, txt := range lastRound {
						name := agentNames[oid]
						if name == "" {
							name = oid
						}
						tag := "对手"
						if oid == aidStr {
							tag = "我方"
						}
						contextParts = append(contextParts, fmt.Sprintf("【%s %s 上轮结论】\n%s", tag, name, txt))
					}
					// Include original analysis context so agents don't lose the topic
					if debateContext != "" {
						contextParts = append(contextParts, fmt.Sprintf("【原始分析数据参考】\n%s", debateContext))
					}
				}
				others := strings.Join(contextParts, "\n\n")
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
				prompt := templatex.Render(ag.SystemPrompt, templatex.MergeStringMap(
					store.flatVars(),
					map[string]string{
						"debate.others_last": others,
						"debate.round":       fmt.Sprint(round),
					},
				))
				reportDate := store.getString("report_date")
				if reportDate == "" {
					reportDate = time.Now().Format("2006年01月02日")
				}
				userMsg := fmt.Sprintf("[系统时间：%s]\n以下是辩论分析数据:\n\n%s\n\n请基于以上数据，结合你的专业判断，输出本轮辩论结论。", reportDate, others)
				msgs := []llm.Message{
					{Role: "system", Content: prompt},
					{Role: "user", Content: userMsg},
				}
				debateStepID := uuid.NewString()
				emit("debate_stream_start", map[string]any{
					"step_id":    debateStepID,
					"agent_id":   aid.String(),
					"agent_name": ag.Name,
					"round":      round,
				})
				var debateAcc strings.Builder
				resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
					Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
					Temp: mo.Temperature, MaxTokens: mo.MaxTokens,
					Timeout: time.Duration(mo.TimeoutMS) * time.Millisecond,
					Retries: mo.RetryCount, Stream: true,
					OnChunk: func(c llm.StreamChunk) error {
						if c.Delta != "" {
							debateAcc.WriteString(c.Delta)
							emit("debate_stream_chunk", map[string]any{
								"step_id": debateStepID,
								"chunk":   c.Delta,
								"accumulated": debateAcc.String(),
							})
						}
						return nil
					},
				})
				if err != nil {
					emit("debate_stream_end", map[string]any{
						"step_id":     debateStepID,
						"full_output": debateAcc.String(),
						"error":       err.Error(),
					})
					eMu.Lock()
					errList = append(errList, err)
					eMu.Unlock()
					return
				}
				debateOut := resp.Content
				if debateOut == "" {
					debateOut = debateAcc.String()
				}
				emit("debate_stream_end", map[string]any{
					"step_id":     debateStepID,
					"full_output": debateOut,
					"tokens_used": resp.Tokens,
				})
				mu.Lock()
				roundOutputs[aid.String()] = debateOut
				if name, ok := agentNames[aid.String()]; ok && name != "" {
					lastRoundNames[aid.String()] = name
				}
				mu.Unlock()
			}()
		}
		wg.Wait()
		if len(errList) > 0 {
			return "", errList[0]
		}
		lastRound = roundOutputs
		for aid, output := range roundOutputs {
			name := agentNames[aid]
			if name == "" {
				name = aid
			}
			store.appendDebate(map[string]any{
				"agent_id":   aid,
				"agent_name": name,
				"output":     output,
				"round":      round,
			})
		}
		// Build array-format debate round for frontend display
		debateEntries := make([]map[string]any, 0, len(roundOutputs))
		for aid, output := range roundOutputs {
			name := agentNames[aid]
			if name == "" {
				name = aid
			}
			debateEntries = append(debateEntries, map[string]any{
				"agent_id":   aid,
				"agent_name": name,
				"output":     output,
				"round":      round,
			})
		}
		emit("debate_round", map[string]any{"node_id": node.ID, "round": round, "agent_outputs": debateEntries})
		if stopConsensus && consensus(roundOutputs) {
			store.log("INFO", node.ID, fmt.Sprintf("debate consensus reached at round %d", round))
			break
		}
	}
	// Store each debate agent's final output so downstream nodes (summarize, risk_review) can access them
	for aidStr, txt := range lastRound {
		if aid, err := uuid.Parse(aidStr); err == nil {
			store.setAgentOutput(aid, txt)
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

// runCrossValidate validates consistency across multiple agent outputs.
func (r *Runner) runCrossValidate(ctx context.Context, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	ids := toUUIDSlice(node.Data["agent_ids"])
	var sb strings.Builder
	for _, aid := range ids {
		txt := store.getString("agent:" + aid.String() + ":output")
		sb.WriteString(fmt.Sprintf("【%s】\n%s\n", aid.String(), txt))
	}
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
	reportDate := store.getString("report_date")
	if reportDate == "" {
		reportDate = time.Now().Format("2006年01月02日")
	}
	userContent := fmt.Sprintf("[系统时间：%s]\n%s", reportDate, sb.String())
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: userContent}}

	stepID := uuid.NewString()
	emit("cross_validate_stream_start", map[string]any{
		"step_id": stepID,
		"node_id": node.ID,
	})

	var acc strings.Builder
	llmTimeout := time.Duration(mo.TimeoutMS) * time.Millisecond
	if llmTimeout < 300*time.Second {
		llmTimeout = 300 * time.Second
	}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.2, MaxTokens: mo.MaxTokens, Timeout: llmTimeout,
		Retries: mo.RetryCount, Stream: true,
		OnChunk: func(c llm.StreamChunk) error {
			if c.Delta != "" {
				acc.WriteString(c.Delta)
				emit("cross_validate_stream_chunk", map[string]any{
					"step_id": stepID,
					"chunk":   c.Delta,
					"accumulated": acc.String(),
				})
			}
			return nil
		},
	})
	if err != nil {
		emit("cross_validate_stream_end", map[string]any{
			"step_id":     stepID,
			"full_output": acc.String(),
			"error":       err.Error(),
		})
		return "", err
	}
	out := resp.Content
	if out == "" {
		out = acc.String()
	}
	emit("cross_validate_stream_end", map[string]any{
		"step_id":     stepID,
		"full_output": out,
		"tokens_used": resp.Tokens,
	})
	emit("cross_validate", map[string]any{"node_id": node.ID, "result": out})
	return out, nil
}

// runRiskReview performs risk assessment on previous outputs.
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
	reportDate := store.getString("report_date")
	if reportDate == "" {
		reportDate = time.Now().Format("2006年01月02日")
	}
	userContent := fmt.Sprintf("[系统时间：%s]\n%s", reportDate, body)
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: userContent}}

	stepID := uuid.NewString()
	emit("risk_review_stream_start", map[string]any{
		"step_id": stepID,
		"node_id": node.ID,
	})

	var acc strings.Builder
	llmTimeout := time.Duration(mo.TimeoutMS) * time.Millisecond
	if llmTimeout < 300*time.Second {
		llmTimeout = 300 * time.Second
	}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.2, MaxTokens: mo.MaxTokens, Timeout: llmTimeout,
		Retries: mo.RetryCount, Stream: true,
		OnChunk: func(c llm.StreamChunk) error {
			if c.Delta != "" {
				acc.WriteString(c.Delta)
				emit("risk_review_stream_chunk", map[string]any{
					"step_id": stepID,
					"chunk":   c.Delta,
					"accumulated": acc.String(),
				})
			}
			return nil
		},
	})
	if err != nil {
		emit("risk_review_stream_end", map[string]any{
			"step_id":     stepID,
			"full_output": acc.String(),
			"error":       err.Error(),
		})
		return "", err
	}
	out := resp.Content
	if out == "" {
		out = acc.String()
	}
	emit("risk_review_stream_end", map[string]any{
		"step_id":     stepID,
		"full_output": out,
		"tokens_used": resp.Tokens,
	})
	store.appendRisk(map[string]any{"node_id": node.ID, "text": out})
	emit("risk_review", map[string]any{"node_id": node.ID, "result": out})
	return out, nil
}

func collectAgentOutputs(store *runStore) []string {
	_, _, _, _, ag := store.snapshot()
	var lines []string
	for k, v := range ag {
		lines = append(lines, fmt.Sprintf("【%s】\n%v", k, v))
	}
	return lines
}

// collectAllOutputs collects ALL accumulated outputs including agent outputs,
// cross_validate, risk_review, debate, and other node outputs from the store.
func collectAllOutputs(store *runStore) []string {
	// Start with all agent outputs
	_, _, _, _, ag := store.snapshot()
	seen := map[string]bool{}
	var lines []string
	for k, v := range ag {
		lines = append(lines, fmt.Sprintf("【%s】\n%v", k, v))
		seen[k] = true
	}

	// Also include node outputs from kv that are NOT agent outputs
	// (e.g. cross_validate, risk_review, debate node results)
	flat := store.flatVars()
	for k, v := range flat {
		// Skip internal keys and already-included agent outputs
		if strings.HasPrefix(k, "agent:") && strings.HasSuffix(k, ":output") {
			agentID := strings.TrimPrefix(k, "agent:")
			agentID = strings.TrimSuffix(agentID, ":output")
			if seen[agentID] {
				continue
			}
		}
		if strings.HasPrefix(k, "node:") {
			// Show node output as a section
			nodeID := strings.TrimPrefix(k, "node:")
			nodeID = strings.TrimSuffix(nodeID, ":output")
			if !seen[nodeID] {
				lines = append(lines, fmt.Sprintf("【%s】\n%v", nodeID, v))
				seen[nodeID] = true
			}
		}
	}
	return lines
}

// pickAnyModel picks any available model, preferring workflow default.
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

// runSummarize generates final summary report.
func (r *Runner) runSummarize(ctx context.Context, wf *model.Workflow, node FlowNode, store *runStore, emit func(string, any)) (string, error) {
	sys := "你是报告汇总专家，请输出最终 Markdown 报告。"
	if v, ok := node.Data["prompt"].(string); ok && v != "" {
		sys = templatex.Render(v, store.flatVars())
	}
	// Collect ALL accumulated outputs — agent outputs + cross_validate + risk_review + debate
	body := strings.Join(collectAllOutputs(store), "\n\n")
	mo, err := r.pickAnyModel(wf)
	if err != nil {
		return "", err
	}
	apiKey, err := r.decryptAPIKey(mo.APIKeyEncrypted)
	if err != nil {
		return "", err
	}
	reportDate := store.getString("report_date")
	if reportDate == "" {
		reportDate = time.Now().Format("2006年01月02日")
	}
	userContent := fmt.Sprintf("[系统时间：%s]\n请基于以下所有分析结果，生成一份完整的最终 Markdown 报告。\n\n%s", reportDate, body)
	msgs := []llm.Message{{Role: "system", Content: sys}, {Role: "user", Content: userContent}}

	stepID := uuid.NewString()
	emit("summarize_stream_start", map[string]any{
		"step_id": stepID,
		"node_id": node.ID,
	})

	var acc strings.Builder
	// Summarize processes all agent outputs — use generous timeout (5min)
	llmTimeout := time.Duration(mo.TimeoutMS) * time.Millisecond
	if llmTimeout < 300*time.Second {
		llmTimeout = 300 * time.Second
	}
	resp, err := r.LLM.Chat(ctx, llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: apiKey, Model: mo.ModelID, Messages: msgs,
		Temp: 0.4, MaxTokens: mo.MaxTokens, Timeout: llmTimeout,
		Retries: mo.RetryCount, Stream: true,
		OnChunk: func(c llm.StreamChunk) error {
			if c.Delta != "" {
				acc.WriteString(c.Delta)
				emit("summarize_stream_chunk", map[string]any{
					"step_id": stepID,
					"chunk":   c.Delta,
					"accumulated": acc.String(),
				})
			}
			return nil
		},
	})
	if err != nil {
		emit("summarize_stream_end", map[string]any{
			"step_id":     stepID,
			"full_output": acc.String(),
			"error":       err.Error(),
		})
		return "", err
	}
	out := resp.Content
	if out == "" {
		out = acc.String()
	}
	emit("summarize_stream_end", map[string]any{
		"step_id":     stepID,
		"full_output": out,
		"tokens_used": resp.Tokens,
	})
	emit("summarize", map[string]any{"node_id": node.ID, "result": out})
	store.setKV("report:md", out)
	return out, nil
}

// runTransform applies variable transformation rules.
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

// finalizeSuccess creates report and updates task status on successful completion.
func (r *Runner) finalizeSuccess(ctx context.Context, taskID uuid.UUID, wf *model.Workflow, t *model.Task, store *runStore, title string, dur int64) error {
	glob, logs, debate, risk, agents := store.snapshot()
	md := store.getString("report:md")
	if strings.TrimSpace(md) == "" {
		md = store.getString("report:title")
	}

	// Debate logs are already stored in frontend-friendly format {agent_id, agent_name, output, round}
	debateOut := make([]map[string]any, 0)
	for _, raw := range debate {
		if m, ok := raw.(map[string]any); ok {
			debateOut = append(debateOut, m)
		}
	}

	// Restructure risk reviews to frontend-friendly format
	riskOut := make([]map[string]any, 0)
	for _, raw := range risk {
		if m, ok := raw.(map[string]any); ok {
			text, _ := m["text"].(string)
			nodeID, _ := m["node_id"].(string)
			level := "medium"
			if strings.Contains(text, "严重") || strings.Contains(text, "critical") {
				level = "critical"
			} else if strings.Contains(text, "高") || strings.Contains(text, "high") {
				level = "high"
			} else if strings.Contains(text, "低") || strings.Contains(text, "low") {
				level = "low"
			}
			riskOut = append(riskOut, map[string]any{
				"dimension": nodeID,
				"level":     level,
				"summary":   text,
			})
		}
	}

	// Restructure exec logs to frontend-friendly format
	logOut := make([]map[string]any, 0)
	for _, raw := range logs {
		if m, ok := raw.(map[string]any); ok {
			node, _ := m["node"].(string)
			level, _ := m["level"].(string)
			msg, _ := m["msg"].(string)
			ts, _ := m["ts"].(float64)
			logOut = append(logOut, map[string]any{
				"node_id":   node,
				"node_type": "",
				"action":    level,
				"detail":    msg,
				"timestamp": ts,
			})
		}
	}

	rep := model.Report{
		ID:            uuid.New(),
		TaskID:        taskID,
		WorkflowID:    wf.ID,
		OwnerID:       t.OwnerID,
		Title:         title,
		ContentMD:     md,
		AgentOutputs:  jsonutil.MustMarshal(agents),
		DebateLogs:    jsonutil.MustMarshal(debateOut),
		RiskReviews:   jsonutil.MustMarshal(riskOut),
		ExecLogs:      jsonutil.MustMarshal(logOut),
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
