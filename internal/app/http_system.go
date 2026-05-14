package app

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

func (a *App) GetSystemConfig(c *gin.Context) {
	var list []model.SystemConfig
	_ = a.DB.Find(&list).Error
	out := make([]gin.H, 0, len(list))
	for _, s := range list {
		out = append(out, gin.H{"key": s.Key, "value": string(s.Value), "description": s.Description})
	}
	response.OK(c, out)
}

func (a *App) PatchSystemConfig(c *gin.Context) {
	var body []struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	for _, it := range body {
		_ = a.DB.Model(&model.SystemConfig{}).Where("cfg_key = ?", it.Key).Updates(map[string]any{"value": jbytes(it.Value)}).Error
	}
	response.OK(c, gin.H{"ok": true})
}

// Dashboard returns dashboard statistics for the current user.
func (a *App) Dashboard(c *gin.Context) {
	var tasks, wfs, agents, reports int64
	since := time.Now().Add(-24 * time.Hour)
	ownerID := uid(c)
	_ = a.DB.Model(&model.Task{}).Where("owner_id = ?", ownerID).Count(&tasks).Error
	_ = a.DB.Model(&model.Workflow{}).Where("owner_id = ?", ownerID).Count(&wfs).Error
	_ = a.DB.Model(&model.Agent{}).Where("owner_id = ?", ownerID).Count(&agents).Error
	_ = a.DB.Model(&model.Report{}).Where("owner_id = ?", ownerID).Count(&reports).Error
	var runningCnt int64
	_ = a.DB.Model(&model.Task{}).Where("owner_id = ? AND status = ?", ownerID, "running").Count(&runningCnt).Error
	var okCnt, allCnt int64
	_ = a.DB.Model(&model.Task{}).Where("owner_id = ? AND created_at >= ?", ownerID, since).Count(&allCnt).Error
	_ = a.DB.Model(&model.Task{}).Where("owner_id = ? AND created_at >= ? AND status = ?", ownerID, since, "completed").Count(&okCnt).Error
	var rate float64
	if allCnt > 0 {
		rate = float64(okCnt) / float64(allCnt) * 100
		// Round to 1 decimal place
		rate = float64(int(rate*10+0.5)) / 10
	}
	var recentTasks []model.Task
	_ = a.DB.Where("owner_id = ?", ownerID).Order("created_at desc").Limit(10).Find(&recentTasks).Error
	recentOut := make([]gin.H, 0, len(recentTasks))
	for _, t := range recentTasks {
		var wfName string
		var wf model.Workflow
		if err := a.DB.First(&wf, "id = ?", t.WorkflowID).Error; err == nil {
			wfName = wf.Name
		}
		recentOut = append(recentOut, gin.H{
			"id": t.ID.String(), "workflow_name": wfName, "status": t.Status,
			"duration_ms": t.DurationMS, "created_at": t.CreatedAt,
		})
	}
	response.OK(c, gin.H{
		"workflow_count": wfs, "agent_count": agents, "task_running_count": runningCnt,
		"report_count": reports, "success_rate_24h": rate, "recent_tasks": recentOut,
	})
}

// ListAuditLogs returns paginated audit logs with search and filter support.
func (a *App) ListAuditLogs(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.OperationLog
	q := a.DB.Model(&model.OperationLog{})
	if kw := p.Keyword; kw != "" {
		q = q.Where("action ILIKE ? OR resource_type ILIKE ? OR resource_id ILIKE ?", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if resType := c.Query("resource_type"); resType != "" {
		q = q.Where("resource_type = ?", resType)
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, o := range list {
		var uid, rid, ip string
		if o.UserID != nil {
			uid = o.UserID.String()
		}
		if o.ResourceID != nil {
			rid = o.ResourceID.String()
		}
		if o.IP != nil {
			ip = *o.IP
		}
		out = append(out, gin.H{
			"id": o.ID.String(), "user_id": uid,
			"action": o.Action, "resource_type": o.ResourceType,
			"resource_id": rid, "detail": json.RawMessage(o.Detail),
			"ip": ip, "created_at": o.CreatedAt,
		})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}
