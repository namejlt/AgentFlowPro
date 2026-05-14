package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/namejlt/AgentFlowPro/internal/export"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

// ListReports returns a paginated list of reports for the current user.
func (a *App) ListReports(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.Report
	q := a.DB.Model(&model.Report{}).Where("owner_id = ?", uid(c))
	if wf := c.Query("workflow_id"); wf != "" {
		if id, err := parseUUID(wf); err == nil {
			q = q.Where("workflow_id = ?", id)
		}
	}
	if kw := p.Keyword; kw != "" {
		q = q.Where("title ILIKE ?", "%"+kw+"%")
	}
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if ar := c.Query("archived"); ar != "" {
		q = q.Where("archived = ?", ar == "true")
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, r := range list {
		var wfName string
		var wf model.Workflow
		if err := a.DB.First(&wf, "id = ?", r.WorkflowID).Error; err == nil {
			wfName = wf.Name
		}
		out = append(out, gin.H{
			"id": r.ID.String(), "task_id": r.TaskID.String(), "title": r.Title,
			"workflow_id": r.WorkflowID.String(), "workflow_name": wfName,
			"owner_id": r.OwnerID.String(), "status": r.Status, "archived": r.Archived,
			"total_tokens": r.TotalTokens, "duration_ms": r.DurationMS,
			"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
		})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

// GetReport returns a single report by ID for the current user.
func (a *App) GetReport(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var r model.Report
	if err := a.DB.First(&r, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var wfName string
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ?", r.WorkflowID).Error; err == nil {
		wfName = wf.Name
	}
	response.OK(c, gin.H{
		"id": r.ID.String(), "task_id": r.TaskID.String(), "title": r.Title,
		"workflow_id": r.WorkflowID.String(), "workflow_name": wfName,
		"owner_id": r.OwnerID.String(), "content_md": r.ContentMD,
		"agent_outputs": r.AgentOutputs, "debate_logs": r.DebateLogs,
		"risk_reviews": r.RiskReviews, "exec_logs": r.ExecLogs,
		"input_snapshot": r.InputSnapshot, "status": r.Status,
		"archived": r.Archived, "total_tokens": r.TotalTokens,
		"duration_ms": r.DurationMS, "created_at": r.CreatedAt,
		"updated_at": r.UpdatedAt,
	})
}

func (a *App) ExportReportMD(c *gin.Context) {
	r := a.mustReport(c)
	if r == nil {
		return
	}
	b, name := export.MarkdownFile(r.Title, r.ContentMD)
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+name)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", b)
}

func (a *App) ExportReportPDF(c *gin.Context) {
	r := a.mustReport(c)
	if r == nil {
		return
	}
	pdf, err := export.BuildPDF(c.Request.Context(), r.ContentMD, a.Cfg.ChromePath)
	if err != nil {
		response.Fail(c, apperr.ErrExportNotReady)
		return
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+export.PDFFilename(r.Title))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func (a *App) ExportReportDOCX(c *gin.Context) {
	r := a.mustReport(c)
	if r == nil {
		return
	}
	b, err := export.BuildDOCX(r.Title, r.ContentMD)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", "attachment; filename="+export.DOCXFilename(r.Title))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", b)
}

func (a *App) mustReport(c *gin.Context) *model.Report {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return nil
	}
	var r model.Report
	if err := a.DB.First(&r, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return nil
	}
	return &r
}

func (a *App) DeleteReport(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.Report{}, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) ArchiveReport(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Archived bool `json:"archived"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := a.DB.Model(&model.Report{}).Where("id = ? AND owner_id = ?", id, uid(c)).Update("archived", body.Archived).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) BatchDeleteReports(c *gin.Context) {
	var body struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.Report{}, "owner_id = ? AND id IN ?", uid(c), body.IDs).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
