package app

import (
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

func (a *App) Dashboard(c *gin.Context) {
	var tasks, wfs, agents, reports int64
	since := time.Now().Add(-24 * time.Hour)
	_ = a.DB.Model(&model.Task{}).Count(&tasks).Error
	_ = a.DB.Model(&model.Workflow{}).Count(&wfs).Error
	_ = a.DB.Model(&model.Agent{}).Count(&agents).Error
	_ = a.DB.Model(&model.Report{}).Count(&reports).Error
	var okCnt, allCnt int64
	_ = a.DB.Model(&model.Task{}).Where("created_at >= ?", since).Count(&allCnt).Error
	_ = a.DB.Model(&model.Task{}).Where("created_at >= ? AND status = ?", since, "completed").Count(&okCnt).Error
	var rate float64
	if allCnt > 0 {
		rate = float64(okCnt) / float64(allCnt)
	}
	response.OK(c, gin.H{
		"tasks_total": tasks, "workflows_total": wfs, "agents_total": agents, "reports_total": reports,
		"last_24h_success_rate": rate,
	})
}

func (a *App) ListAuditLogs(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.OperationLog
	q := a.DB.Model(&model.OperationLog{})
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, o := range list {
		out = append(out, gin.H{"id": o.ID.String(), "user_id": o.UserID, "action": o.Action, "resource_type": o.ResourceType, "resource_id": o.ResourceID, "created_at": o.CreatedAt})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}
