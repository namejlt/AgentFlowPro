package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

type wfDTO struct {
	Name           string         `json:"name"`
	Description    *string        `json:"description"`
	Tags           []string       `json:"tags"`
	GlobalParams   []any          `json:"global_params"`
	Nodes          []any          `json:"nodes"`
	Edges          []any          `json:"edges"`
	ExecConfig     map[string]any `json:"exec_config"`
	DefaultModelID *uuid.UUID     `json:"default_model_id"`
	Visibility     string         `json:"visibility"`
}

func (a *App) CreateWorkflow(c *gin.Context) {
	var dto wfDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	wf := model.Workflow{
		OwnerID: uid(c), Name: dto.Name, Description: dto.Description, Tags: jbytes(dto.Tags),
		GlobalParams: jbytes(dto.GlobalParams), Nodes: jbytes(dto.Nodes), Edges: jbytes(dto.Edges),
		ExecConfig: jbytes(dto.ExecConfig), DefaultModelID: dto.DefaultModelID,
		Version: 1, Visibility: firstStr(dto.Visibility, "private"),
	}
	if err := a.DB.Create(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrConflict)
		return
	}
	a.saveVersion(&wf, uid(c))
	a.LogCreate(c, "workflow", wf.ID, gin.H{"name": wf.Name})
	response.OK(c, a.toWFView(&wf))
}

func (a *App) UpdateWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto wfDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	wf.Name = dto.Name
	wf.Description = dto.Description
	wf.Tags = jbytes(dto.Tags)
	wf.GlobalParams = jbytes(dto.GlobalParams)
	wf.Nodes = jbytes(dto.Nodes)
	wf.Edges = jbytes(dto.Edges)
	wf.ExecConfig = jbytes(dto.ExecConfig)
	wf.DefaultModelID = dto.DefaultModelID
	if dto.Visibility != "" {
		wf.Visibility = dto.Visibility
	}
	wf.Version++
	if err := a.DB.Save(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.saveVersion(&wf, uid(c))
	a.LogUpdate(c, "workflow", wf.ID, gin.H{"name": wf.Name})
	response.OK(c, a.toWFView(&wf))
}

func (a *App) saveVersion(wf *model.Workflow, creator uuid.UUID) {
	snap := map[string]any{
		"name": wf.Name, "description": wf.Description, "tags": json.RawMessage(wf.Tags),
		"global_params": json.RawMessage(wf.GlobalParams), "nodes": json.RawMessage(wf.Nodes), "edges": json.RawMessage(wf.Edges),
		"exec_config": json.RawMessage(wf.ExecConfig), "default_model_id": wf.DefaultModelID, "version": wf.Version,
	}
	cb := creator
	v := model.WorkflowVersion{WorkflowID: wf.ID, Version: wf.Version, Snapshot: jbytes(snap), CreatedBy: &cb}
	_ = a.DB.Create(&v).Error
}

func (a *App) CloneWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	if wf.OwnerID != uid(c) && wf.Visibility != "public" && wf.Visibility != "shared" {
		response.Fail(c, apperr.ErrForbidden)
		return
	}
	wf.ID = uuid.Nil
	wf.OwnerID = uid(c)
	wf.Name = wf.Name + " (copy)"
	wf.Version = 1
	wf.ShareCode = nil
	if err := a.DB.Create(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.saveVersion(&wf, uid(c))
	response.OK(c, a.toWFView(&wf))
}

func (a *App) DeleteWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.Workflow{}, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.LogDelete(c, "workflow", id, gin.H{})
	response.OK(c, gin.H{"ok": true})
}

func (a *App) ListWorkflows(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.Workflow
	q := a.DB.Model(&model.Workflow{})
	q = q.Where("owner_id = ? OR visibility IN ?", uid(c), []string{"public", "shared"})
	if vis := c.Query("visibility"); vis != "" {
		q = q.Where("visibility = ?", vis)
	}
	if kw := p.Keyword; kw != "" {
		q = q.Where("name ILIKE ?", "%"+kw+"%")
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("updated_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, a.toWFView(&list[i]))
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) GetWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	if wf.OwnerID != uid(c) && wf.Visibility == "private" {
		response.Fail(c, apperr.ErrForbidden)
		return
	}
	response.OK(c, a.toWFView(&wf))
}

func (a *App) toWFView(wf *model.Workflow) gin.H {
	var tags []string
	_ = json.Unmarshal(wf.Tags, &tags)
	return gin.H{
		"id": wf.ID.String(), "name": wf.Name, "description": wf.Description, "tags": tags,
		"global_params": json.RawMessage(wf.GlobalParams), "nodes": json.RawMessage(wf.Nodes), "edges": json.RawMessage(wf.Edges),
		"exec_config": json.RawMessage(wf.ExecConfig), "default_model_id": wf.DefaultModelID, "version": wf.Version,
		"visibility": wf.Visibility, "share_code": wf.ShareCode, "archived": wf.Archived, "last_run_at": wf.LastRunAt, "run_count": wf.RunCount,
		"owner_id": wf.OwnerID.String(), "created_at": wf.CreatedAt, "updated_at": wf.UpdatedAt,
	}
}

func (a *App) ListWorkflowVersions(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	p := pagination.FromQuery(c)
	var total int64
	_ = a.DB.Model(&model.WorkflowVersion{}).Where("workflow_id = ?", id).Count(&total).Error
	var list []model.WorkflowVersion
	if err := a.DB.Where("workflow_id = ?", id).Order("version desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, v := range list {
		var createdBy string
		if v.CreatedBy != nil {
			createdBy = v.CreatedBy.String()
		}
		out = append(out, gin.H{"version": v.Version, "created_at": v.CreatedAt, "created_by": createdBy, "snapshot": json.RawMessage(v.Snapshot)})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) RollbackWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	ver := c.Param("ver")
	verInt, err := strconv.Atoi(ver)
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var snap model.WorkflowVersion
	if err := a.DB.Where("workflow_id = ? AND version = ?", id, verInt).First(&snap).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var m map[string]any
	_ = json.Unmarshal(snap.Snapshot, &m)
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	wf.Name = str(m["name"])
	wf.Description = strPtr(m["description"])
	wf.Tags = jbytes(m["tags"])
	wf.GlobalParams = jbytes(m["global_params"])
	wf.Nodes = jbytes(m["nodes"])
	wf.Edges = jbytes(m["edges"])
	wf.ExecConfig = jbytes(m["exec_config"])
	wf.Version++
	if err := a.DB.Save(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.saveVersion(&wf, uid(c))
	response.OK(c, a.toWFView(&wf))
}

func str(v any) string { s, _ := v.(string); return s }
func strPtr(v any) *string {
	if v == nil {
		return nil
	}
	s, _ := v.(string)
	return &s
}

func (a *App) ExportWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	payload := map[string]any{
		"format": "agentflow", "version": 2,
		"name": wf.Name, "description": wf.Description, "tags": json.RawMessage(wf.Tags),
		"global_params": json.RawMessage(wf.GlobalParams), "nodes": json.RawMessage(wf.Nodes), "edges": json.RawMessage(wf.Edges),
		"exec_config": json.RawMessage(wf.ExecConfig),
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename="+wf.Name+".agentflow.json")
	c.Writer.WriteHeader(200)
	_, _ = c.Writer.Write(b)
}

func (a *App) ImportWorkflow(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	match := a.buildMatchReport(body)
	sess := model.WorkflowImportSession{OwnerID: uid(c), Payload: jbytes(body), MatchReport: jbytes(match), Status: "pending"}
	if err := a.DB.Create(&sess).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"session_id": sess.ID.String(), "match_report": match})
}

func (a *App) buildMatchReport(payload map[string]any) gin.H {
	matchedAgents := []gin.H{}
	matchedDS := []gin.H{}
	matchedModels := []gin.H{}
	missingAgents := []gin.H{}
	missingDS := []gin.H{}
	missingModels := []gin.H{}

	nodes, _ := payload["nodes"].([]any)
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		data, _ := node["data"].(map[string]any)
		if agentID, ok := data["agent_id"].(string); ok {
			var ag model.Agent
			if err := a.DB.First(&ag, "id = ?", agentID).Error; err == nil {
				matchedAgents = append(matchedAgents, gin.H{"import_id": agentID, "import_name": agentID, "local_id": ag.ID.String(), "local_name": ag.Name})
			} else {
				missingAgents = append(missingAgents, gin.H{"import_id": agentID, "import_name": agentID})
			}
		}
		if agentIDs, ok := data["agent_ids"].([]any); ok {
			for _, aid := range agentIDs {
				idStr, _ := aid.(string)
				var ag model.Agent
				if err := a.DB.First(&ag, "id = ?", idStr).Error; err == nil {
					matchedAgents = append(matchedAgents, gin.H{"import_id": idStr, "import_name": idStr, "local_id": ag.ID.String(), "local_name": ag.Name})
				} else {
					missingAgents = append(missingAgents, gin.H{"import_id": idStr, "import_name": idStr})
				}
			}
		}
	}

	if defaultModelID, ok := payload["default_model_id"].(string); ok {
		var mo model.LLMModel
		if err := a.DB.First(&mo, "id = ?", defaultModelID).Error; err == nil {
			matchedModels = append(matchedModels, gin.H{"import_id": defaultModelID, "import_name": defaultModelID, "local_id": mo.ID.String(), "local_name": mo.Name})
		} else {
			missingModels = append(missingModels, gin.H{"import_id": defaultModelID, "import_name": defaultModelID})
		}
	}

	return gin.H{
		"matched_agents": matchedAgents, "matched_datasources": matchedDS, "matched_models": matchedModels,
		"missing_agents": missingAgents, "missing_datasources": missingDS, "missing_models": missingModels,
	}
}

func (a *App) ConfirmImport(c *gin.Context) {
	var body struct {
		SessionID uuid.UUID      `json:"session_id"`
		Bindings  map[string]any `json:"bindings"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var sess model.WorkflowImportSession
	if err := a.DB.First(&sess, "id = ? AND owner_id = ?", body.SessionID, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var p map[string]any
	_ = json.Unmarshal(sess.Payload, &p)
	// Apply bindings: replace old resource IDs with new ones in nodes
	if len(body.Bindings) > 0 {
		nodes, _ := p["nodes"].([]any)
		for _, n := range nodes {
			node, _ := n.(map[string]any)
			data, _ := node["data"].(map[string]any)
			if agentID, ok := data["agent_id"].(string); ok {
				if newID, exists := body.Bindings[agentID]; exists {
					if s, ok2 := newID.(string); ok2 {
						data["agent_id"] = s
					}
				}
			}
			if agentIDs, ok := data["agent_ids"].([]any); ok {
				for i, aid := range agentIDs {
					if idStr, ok2 := aid.(string); ok2 {
						if newID, exists := body.Bindings[idStr]; exists {
							if s, ok3 := newID.(string); ok3 {
								agentIDs[i] = s
							}
						}
					}
				}
				data["agent_ids"] = agentIDs
			}
		}
		p["nodes"] = nodes
		if modelID, ok := p["default_model_id"].(string); ok {
			if newID, exists := body.Bindings[modelID]; exists {
				if s, ok2 := newID.(string); ok2 {
					p["default_model_id"] = s
				}
			}
		}
	}
	wf := model.Workflow{
		OwnerID: uid(c), Name: str(p["name"]) + " (imported)",
		Description: strPtr(p["description"]),
		Tags:        jbytes(p["tags"]), GlobalParams: jbytes(p["global_params"]), Nodes: jbytes(p["nodes"]), Edges: jbytes(p["edges"]),
		ExecConfig: jbytes(p["exec_config"]), Version: 1, Visibility: "private",
	}
	if err := a.DB.Create(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.saveVersion(&wf, uid(c))
	_ = a.DB.Model(&sess).Update("status", "completed").Error
	response.OK(c, a.toWFView(&wf))
}

func (a *App) ShareWorkflow(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto struct {
		Visibility string     `json:"visibility"`
		ExpiresAt  *time.Time `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	code := randomCode(16)
	st := model.ShareToken{WorkflowID: id, Code: code, Visibility: firstStr(dto.Visibility, "shared"), ExpiresAt: dto.ExpiresAt, CreatedBy: uid(c)}
	if err := a.DB.Create(&st).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	_ = a.DB.Model(&model.Workflow{}).Where("id = ?", id).Updates(map[string]any{"visibility": dto.Visibility, "share_code": code}).Error
	response.OK(c, gin.H{"share_code": code, "share_url": "/workflows/import?code=" + code})
}

func randomCode(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *App) CloneWorkflowByCode(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var st model.ShareToken
	if err := a.DB.Where("code = ?", body.Code).First(&st).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	if st.ExpiresAt != nil && time.Now().After(*st.ExpiresAt) {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var wf model.Workflow
	if err := a.DB.First(&wf, "id = ?", st.WorkflowID).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	wf.ID = uuid.Nil
	wf.OwnerID = uid(c)
	wf.Name = wf.Name + " (clone)"
	wf.Version = 1
	wf.ShareCode = nil
	if err := a.DB.Create(&wf).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	a.saveVersion(&wf, uid(c))
	response.OK(c, a.toWFView(&wf))
}

func (a *App) PatchWorkflowVisibility(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto struct {
		Visibility string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Model(&model.Workflow{}).Where("id = ? AND owner_id = ?", id, uid(c)).Update("visibility", dto.Visibility).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
