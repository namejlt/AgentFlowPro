package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/namejlt/AgentFlowPro/internal/datasource"
	"github.com/namejlt/AgentFlowPro/internal/llm"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/jsonutil"
	"github.com/namejlt/AgentFlowPro/internal/repository"
)

// Runner executes workflow tasks with DAG-based node scheduling.
type Runner struct {
	DB     *gorm.DB
	Store  *repository.Store
	Key    []byte
	Hub    *Hub
	LLM    *llm.Client
	DS     *datasource.Executor
	Log    *zap.Logger
	MaxPar int

	mu      sync.Mutex
	cancels map[uuid.UUID]context.CancelFunc
}

// NewRunner creates a new workflow task runner.
func NewRunner(db *gorm.DB, store *repository.Store, key []byte, hub *Hub, maxPar int) *Runner {
	if maxPar <= 0 {
		maxPar = 32
	}
	return &Runner{
		DB:      db,
		Store:   store,
		Key:     key,
		Hub:     hub,
		LLM:     llm.NewClient(),
		DS:      datasource.NewExecutor(),
		Log:     zap.L(),
		MaxPar:  maxPar,
		cancels: map[uuid.UUID]context.CancelFunc{},
	}
}

// Stop cancels a running task by ID.
func (r *Runner) Stop(taskID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.cancels[taskID]; ok {
		c()
		delete(r.cancels, taskID)
	}
}

// Run starts executing a task with the given timeout.
func (r *Runner) Run(parent context.Context, taskID uuid.UUID, taskTimeout time.Duration) {
	ctx, cancel := context.WithTimeout(parent, taskTimeout)
	r.mu.Lock()
	r.cancels[taskID] = cancel
	r.mu.Unlock()
	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.cancels, taskID)
		r.mu.Unlock()
	}()

	if err := r.run(ctx, taskID); err != nil {
		r.Log.Error("task failed", zap.String("task_id", taskID.String()), zap.Error(err))
		_ = r.Store.DB.Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]any{
			"status":        "failed",
			"error_message": err.Error(),
			"finished_at":   time.Now(),
		}).Error
		r.Hub.PublishJSON(taskID, "task_failed", map[string]any{
			"task_id":     taskID.String(),
			"error":       err.Error(),
			"failed_step": nil,
		})
	}
}

// run is the core DAG execution engine.
func (r *Runner) run(ctx context.Context, taskID uuid.UUID) error {
	var t model.Task
	if err := r.Store.DB.WithContext(ctx).First(&t, "id = ?", taskID).Error; err != nil {
		return err
	}
	var wf model.Workflow
	if err := r.Store.DB.WithContext(ctx).First(&wf, "id = ?", t.WorkflowID).Error; err != nil {
		return err
	}
	g, err := ParseGraph(wf.Nodes, wf.Edges)
	if err != nil {
		return err
	}

	global, _ := jsonutil.UnmarshalMap(t.InputParams)
	store := newRunStore(global)

	// Inject current date/time as template variables so agents can use {{current_date}} etc.
	now := time.Now()
	store.setKV("current_date", now.Format("2006-01-02"))
	store.setKV("current_time", now.Format("15:04:05"))
	store.setKV("report_date", now.Format("2006年01月02日"))

	_ = r.Store.DB.Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status":           "running",
		"started_at":       time.Now(),
		"workflow_version": t.WorkflowVersion,
	}).Error

	// Initialize state tracking for all nodes
	state := map[string]string{} // pending|running|completed|skipped|failed
	for id := range g.NodeMap {
		state[id] = "pending"
	}

	// Build predecessor count map (mutable copy for tracking)
	pred := g.PredCount()

	// Execution config
	execCfg, _ := jsonutil.UnmarshalMap(wf.ExecConfig)
	maxConc := r.MaxPar
	if v, ok := execCfg["max_concurrency"].(float64); ok && int(v) > 0 {
		maxConc = int(v)
	}
	sem := make(chan struct{}, maxConc)

	started := time.Now()

	// Kahn's algorithm with condition branch support
	ready := []string{}
	for id := range g.NodeMap {
		if pred[id] == 0 {
			ready = append(ready, id)
		}
	}

	for len(ready) > 0 {
		select {
		case <-ctx.Done():
			_ = r.failTask(ctx, taskID, ctx.Err())
			return ctx.Err()
		default:
		}

		nodeID := ready[0]
		ready = ready[1:]

		if state[nodeID] != "pending" {
			continue
		}

		node := g.NodeMap[nodeID]
		state[nodeID] = "running"
		r.Hub.PublishJSON(taskID, "node_status", map[string]any{
			"node_id":   nodeID,
			"status":    "running",
			"timestamp": time.Now().UnixMilli(),
		})

		sem <- struct{}{}
		out, nextLabel, err := r.executeNode(ctx, taskID, &wf, g, node, store, func(ev string, payload any) {
			r.Hub.PublishJSON(taskID, ev, payload)
		})
		<-sem

		if err != nil {
			state[nodeID] = "failed"
			r.Hub.PublishJSON(taskID, "node_status", map[string]any{
				"node_id":   nodeID,
				"status":    "failed",
				"timestamp": time.Now().UnixMilli(),
			})
			_ = r.failTask(ctx, taskID, err)
			return err
		}

		state[nodeID] = "completed"
		store.setNodeOutput(nodeID, out)
		r.Hub.PublishJSON(taskID, "node_status", map[string]any{
			"node_id":   nodeID,
			"status":    "completed",
			"timestamp": time.Now().UnixMilli(),
		})

		// Schedule successors, respecting condition branch labels
		for _, succ := range g.Succs[nodeID] {
			if state[succ] == "skipped" {
				continue
			}

			// Check edge label matching for condition branches
			if edgeLabel, hasLabel := g.EdgeLabels[nodeID][succ]; hasLabel && nextLabel != "" {
				if edgeLabel != nextLabel {
					// This branch is not taken - skip the successor and its downstream
					r.skipBranch(g, state, succ)
					continue
				}
			}

			pred[succ]--
			if pred[succ] == 0 {
				ready = append(ready, succ)
			}
		}
	}

	// Check if all non-skipped nodes completed
	for id, st := range state {
		if st == "pending" {
			return fmt.Errorf("node %s is still pending after execution (possible cycle or disconnected node)", id)
		}
	}

	title := store.getString("report:title")
	if title == "" {
		title = wf.Name + " 报告"
	}
	dur := time.Since(started).Milliseconds()
	return r.finalizeSuccess(ctx, taskID, &wf, &t, store, title, dur)
}

// skipBranch marks a node and all its downstream as skipped.
func (r *Runner) skipBranch(g *Graph, state map[string]string, nodeID string) {
	if state[nodeID] == "skipped" || state[nodeID] == "completed" {
		return
	}
	state[nodeID] = "skipped"
	for _, succ := range g.Succs[nodeID] {
		r.skipBranch(g, state, succ)
	}
}

func (r *Runner) failTask(ctx context.Context, taskID uuid.UUID, err error) error {
	_ = r.Store.DB.WithContext(ctx).Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]any{
		"status": "failed", "error_message": err.Error(), "finished_at": time.Now(),
	}).Error
	return err
}

// runStore holds execution state for a single workflow run.
type runStore struct {
	mu     sync.Mutex
	global map[string]any
	kv     map[string]any
	logs   []any
	debate []any
	risk   []any
	agents map[string]any
}

func newRunStore(global map[string]any) *runStore {
	return &runStore{
		global: global,
		kv:     map[string]any{},
		logs:   []any{},
		debate: []any{},
		risk:   []any{},
		agents: map[string]any{},
	}
}

func (s *runStore) log(level, node, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, map[string]any{
		"ts": time.Now().UnixMilli(), "node": node, "level": level, "msg": msg,
	})
}

func (s *runStore) setNodeOutput(nodeID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kv["node:"+nodeID+":output"] = text
}

func (s *runStore) setAgentOutput(agentID uuid.UUID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kv["agent:"+agentID.String()+":output"] = text
	s.agents[agentID.String()] = text
}

func (s *runStore) setKV(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kv[key] = val
}

func (s *runStore) getString(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.kv[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (s *runStore) snapshot() (map[string]any, []any, []any, []any, map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := map[string]any{}
	for k, v := range s.global {
		g[k] = v
	}
	return g, append([]any{}, s.logs...), append([]any{}, s.debate...), append([]any{}, s.risk...), mapsClone(s.agents)
}

func mapsClone(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *runStore) flatVars() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for k, v := range s.global {
		out[k] = fmt.Sprint(v)
	}
	for k, v := range s.kv {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func (s *runStore) mergedVars() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{}
	for k, v := range s.global {
		out[k] = v
	}
	for k, v := range s.kv {
		out[k] = v
	}
	return out
}

func (s *runStore) appendDebate(rec any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debate = append(s.debate, rec)
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *runStore) appendRisk(rec any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.risk = append(s.risk, rec)
}
