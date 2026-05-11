package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/namejlt/AgentFlowPro/internal/crypto"
	"github.com/namejlt/AgentFlowPro/internal/datasource"
	"github.com/namejlt/AgentFlowPro/internal/llm"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
	"github.com/namejlt/AgentFlowPro/internal/templatex"
)

type agentDTO struct {
	Name           string         `json:"name"`
	RoleDesc       *string        `json:"role_desc"`
	Tags           []string       `json:"tags"`
	Icon           *string        `json:"icon"`
	SystemPrompt   string         `json:"system_prompt"`
	LLMModelID     *uuid.UUID     `json:"llm_model_id"`
	DataSourceID   *uuid.UUID     `json:"datasource_id"`
	ParamMappings  []any          `json:"param_mappings"`
	OutputFormat   string         `json:"output_format"`
	OutputLang     string         `json:"output_lang"`
	MaxOutputChars int            `json:"max_output_chars"`
	Enabled        *bool          `json:"enabled"`
}

func (a *App) CreateAgent(c *gin.Context) {
	var dto agentDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	ag := model.Agent{
		OwnerID: uid(c), Name: dto.Name, RoleDesc: dto.RoleDesc, Tags: jbytes(dto.Tags), Icon: dto.Icon,
		SystemPrompt: dto.SystemPrompt, LLMModelID: dto.LLMModelID, DataSourceID: dto.DataSourceID,
		ParamMappings: jbytes(dto.ParamMappings), OutputFormat: firstStr(dto.OutputFormat, "markdown"),
		OutputLang: firstStr(dto.OutputLang, "zh-CN"), MaxOutputChars: dto.MaxOutputChars, Enabled: true,
	}
	if dto.Enabled != nil {
		ag.Enabled = *dto.Enabled
	}
	if err := a.DB.Create(&ag).Error; err != nil {
		response.Fail(c, apperr.ErrConflict)
		return
	}
	response.OK(c, a.toAgentView(&ag))
}

func (a *App) UpdateAgent(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto agentDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ag model.Agent
	if err := a.DB.First(&ag, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	up := map[string]any{
		"name": dto.Name, "role_desc": dto.RoleDesc, "tags": jbytes(dto.Tags), "icon": dto.Icon,
		"system_prompt": dto.SystemPrompt, "llm_model_id": dto.LLMModelID, "datasource_id": dto.DataSourceID,
		"param_mappings": jbytes(dto.ParamMappings), "output_format": firstStr(dto.OutputFormat, ag.OutputFormat),
		"output_lang": firstStr(dto.OutputLang, ag.OutputLang), "max_output_chars": dto.MaxOutputChars,
	}
	if dto.Enabled != nil {
		up["enabled"] = *dto.Enabled
	}
	if err := a.DB.Model(&ag).Updates(up).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	_ = a.DB.First(&ag, "id = ?", id).Error
	response.OK(c, a.toAgentView(&ag))
}

func (a *App) CloneAgent(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ag model.Agent
	if err := a.DB.First(&ag, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	ag.ID = uuid.Nil
	ag.Name = ag.Name + " (copy)"
	ag.OwnerID = uid(c)
	if err := a.DB.Create(&ag).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, a.toAgentView(&ag))
}

func (a *App) DeleteAgent(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.Agent{}, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) ListAgents(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.Agent
	q := a.DB.Model(&model.Agent{}).Where("owner_id = ?", uid(c))
	if kw := p.Keyword; kw != "" {
		q = q.Where("name ILIKE ?", "%"+kw+"%")
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, a.toAgentView(&list[i]))
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) GetAgent(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ag model.Agent
	if err := a.DB.First(&ag, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, a.toAgentView(&ag))
}

func (a *App) toAgentView(ag *model.Agent) gin.H {
	var tags []string
	_ = json.Unmarshal(ag.Tags, &tags)
	var wfCount int64
	_ = a.DB.Raw(`SELECT COUNT(1) FROM workflows WHERE owner_id = ? AND deleted_at IS NULL`, ag.OwnerID).Scan(&wfCount).Error
	h := gin.H{
		"id": ag.ID.String(), "name": ag.Name, "role_desc": ag.RoleDesc, "tags": tags, "icon": ag.Icon,
		"system_prompt": ag.SystemPrompt, "llm_model_id": ag.LLMModelID, "datasource_id": ag.DataSourceID,
		"param_mappings": json.RawMessage(ag.ParamMappings), "output_format": ag.OutputFormat, "output_lang": ag.OutputLang,
		"max_output_chars": ag.MaxOutputChars, "enabled": ag.Enabled, "referenced_workflows_estimate": wfCount,
	}
	return h
}

func (a *App) PreviewAgent(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var body struct {
		Globals map[string]any `json:"globals"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Globals == nil {
		body.Globals = map[string]any{}
	}
	var ag model.Agent
	if err := a.DB.First(&ag, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var mo model.LLMModel
	if ag.LLMModelID != nil {
		_ = a.DB.First(&mo, "id = ?", *ag.LLMModelID).Error
	} else {
		_ = a.DB.Where("deleted_at IS NULL").Order("is_default desc").First(&mo).Error
	}
	key, err := crypto.Open(a.Cfg.EncryptionKey, mo.APIKeyEncrypted)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	dsText := ""
	if ag.DataSourceID != nil {
		var ds model.DataSource
		if err := a.DB.First(&ds, "id = ?", *ag.DataSourceID).Error; err == nil {
			params, err := datasource.ResolveParams(ds.ParamsSchema, body.Globals, map[string]any{})
			if err == nil {
				ex := datasource.NewExecutor()
				res, err := ex.Execute(c.Request.Context(), &ds, datasource.ExecuteInput{GlobalVars: body.Globals, Params: params}, func(sealed string) (map[string]string, error) {
					if sealed == "" {
						return map[string]string{}, nil
					}
					b, err := crypto.Open(a.Cfg.EncryptionKey, sealed)
					if err != nil {
						return nil, err
					}
					var m map[string]string
					_ = json.Unmarshal(b, &m)
					return m, nil
				}, func(sealed string) (map[string]any, error) {
					if ds.AuthConfigEncrypted == nil || *ds.AuthConfigEncrypted == "" {
						return map[string]any{}, nil
					}
					b, err := crypto.Open(a.Cfg.EncryptionKey, *ds.AuthConfigEncrypted)
					if err != nil {
						return nil, err
					}
					var m map[string]any
					_ = json.Unmarshal(b, &m)
					return m, nil
				})
				if err == nil {
					dsText = res.Extracted
				}
			}
		}
	}
	extras := map[string]string{"datasource.result": dsText}
	for k, v := range body.Globals {
		extras[k] = fmt.Sprint(v)
	}
	prompt := templatex.Render(ag.SystemPrompt, templatex.MergeStringMap(templatex.BuildPromptVarMap(body.Globals, nil), extras))
	client := llm.NewClient()
	resp, err := client.Chat(c.Request.Context(), llm.ChatOpts{
		Endpoint: mo.Endpoint, APIKey: string(key), Model: mo.ModelID,
		Messages: []llm.Message{{Role: "system", Content: prompt}, {Role: "user", Content: "请输出预览结果。"}},
		Temp: mo.Temperature, MaxTokens: minInt(mo.MaxTokens, ag.MaxOutputChars), Timeout: durationFromMS(mo.TimeoutMS), Retries: 0, Stream: false,
	})
	if err != nil {
		response.Fail(c, apperr.ErrUpstream)
		return
	}
	response.OK(c, gin.H{"output": resp.Content, "tokens_used": resp.Tokens})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func durationFromMS(ms int) time.Duration {
	if ms <= 0 {
		return 60 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}
