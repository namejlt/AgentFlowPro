package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

func (a *App) CreateTask(c *gin.Context) {
	var body struct {
		WorkflowID   uuid.UUID      `json:"workflow_id"`
		InputParams  map[string]any `json:"input_params"`
		Mode         string         `json:"mode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ?", body.WorkflowID).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	if wf.OwnerID != uid(c) && wf.Visibility == "private" {
		response.Fail(c, apperr.ErrForbidden)
		return
	}
	t := model.Task{
		WorkflowID:      wf.ID,
		WorkflowVersion: wf.Version,
		OwnerID:         uid(c),
		InputParams:     jbytes(body.InputParams),
		Mode:            firstStr(body.Mode, "normal"),
		Status:          "pending",
	}
	if err := a.DB.Create(&t).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	tid := t.ID
	go func(id uuid.UUID) {
		a.runner.Run(context.Background(), id, a.Cfg.EngineTaskTimeout)
	}(tid)
	response.OK(c, gin.H{"task_id": t.ID.String(), "status": t.Status})
}

func (a *App) GetTask(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var t model.Task
	if err := a.DB.First(&t, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, gin.H{
		"id": t.ID.String(), "workflow_id": t.WorkflowID.String(), "workflow_version": t.WorkflowVersion,
		"input_params": json.RawMessage(t.InputParams), "mode": t.Mode, "status": t.Status,
		"report_id": t.ReportID, "error_message": t.ErrorMessage, "started_at": t.StartedAt, "finished_at": t.FinishedAt, "duration_ms": t.DurationMS,
	})
}

// ListTasks returns a paginated list of tasks for the current user.
func (a *App) ListTasks(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.Task
	q := a.DB.Model(&model.Task{}).Where("owner_id = ?", uid(c))
	if kw := p.Keyword; kw != "" {
		q = q.Where("mode ILIKE ? OR status ILIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		var wfName string
		var wf model.Workflow
		if err := a.DB.First(&wf, "id = ?", t.WorkflowID).Error; err == nil {
			wfName = wf.Name
		}
		out = append(out, gin.H{
			"id": t.ID.String(), "workflow_id": t.WorkflowID.String(), "workflow_name": wfName,
			"workflow_version": t.WorkflowVersion, "owner_id": t.OwnerID.String(),
			"input_params": json.RawMessage(t.InputParams), "mode": t.Mode, "status": t.Status,
			"report_id": t.ReportID, "error_message": t.ErrorMessage, "error_step_id": t.ErrorStepID,
			"started_at": t.StartedAt, "finished_at": t.FinishedAt, "duration_ms": t.DurationMS,
			"created_at": t.CreatedAt, "updated_at": t.UpdatedAt,
		})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) ListTaskSteps(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var t model.Task
	if err := a.DB.First(&t, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var steps []model.TaskStep
	if err := a.DB.Where("task_id = ?", t.ID).Order("created_at asc").Find(&steps).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(steps))
	for _, s := range steps {
		out = append(out, gin.H{
			"id": s.ID.String(), "node_id": s.NodeID, "node_type": s.NodeType, "agent_id": s.AgentID,
			"status": s.Status, "debate_round": s.DebateRound, "input": json.RawMessage(s.Input), "output": json.RawMessage(s.Output),
			"tokens_used": s.TokensUsed, "error_message": s.ErrorMessage, "started_at": s.StartedAt, "finished_at": s.FinishedAt,
		})
	}
	response.OK(c, out)
}

func (a *App) StopTask(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	a.runner.Stop(id)
	_ = a.DB.Model(&model.Task{}).Where("id = ? AND owner_id = ?", id, uid(c)).Updates(map[string]any{"status": "stopped", "finished_at": time.Now()}).Error
	response.OK(c, gin.H{"ok": true})
}

func (a *App) RerunTask(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var old model.Task
	if err := a.DB.First(&old, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	t := model.Task{
		WorkflowID:      old.WorkflowID,
		WorkflowVersion: old.WorkflowVersion,
		OwnerID:         uid(c),
		InputParams:     old.InputParams,
		Mode:            old.Mode,
		Status:          "pending",
	}
	if err := a.DB.Create(&t).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	go func(id uuid.UUID) { a.runner.Run(context.Background(), id, a.Cfg.EngineTaskTimeout) }(t.ID)
	response.OK(c, gin.H{"task_id": t.ID.String()})
}

func (a *App) TaskStream(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var t model.Task
	if err := a.DB.First(&t, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	ch := a.Hub.Subscribe(id)
	defer a.Hub.Unsubscribe(id, ch)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	fl, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(ev.Data)
			_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Event, string(b))
			fl.Flush()
		case <-tick.C:
			_, _ = c.Writer.Write([]byte(": ping\n\n"))
			fl.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
