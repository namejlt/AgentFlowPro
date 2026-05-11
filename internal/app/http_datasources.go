package app

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/namejlt/AgentFlowPro/internal/crypto"
	"github.com/namejlt/AgentFlowPro/internal/datasource"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/jsonutil"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

type dsDTO struct {
	Name             string         `json:"name"`
	Description      *string        `json:"description"`
	Category         *string        `json:"category"`
	Tags             []string       `json:"tags"`
	Icon             *string        `json:"icon"`
	DSType           string         `json:"ds_type"`
	URLTemplate      *string        `json:"url_template"`
	HTTPMethod       *string        `json:"http_method"`
	ContentType      *string        `json:"content_type"`
	TimeoutMS        int            `json:"timeout_ms"`
	RetryCount       int            `json:"retry_count"`
	Headers          map[string]string `json:"headers"`
	BodyTemplate     map[string]any `json:"body_template"`
	AuthType         string         `json:"auth_type"`
	AuthConfig       map[string]any `json:"auth_config"`
	ParamsSchema     []any          `json:"params_schema"`
	CachePolicy      string         `json:"cache_policy"`
	CacheTTLSeconds  *int           `json:"cache_ttl_seconds"`
	ResponseJSONPath *string       `json:"response_jsonpath"`
	ExtraConfig      map[string]any `json:"extra_config"`
	UploadedFileID   *uuid.UUID     `json:"uploaded_file_id"`
	Enabled          *bool          `json:"enabled"`
}

func (a *App) CreateDatasource(c *gin.Context) {
	var dto dsDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	ds := model.DataSource{
		OwnerID:     uid(c),
		Name:        dto.Name,
		Description: dto.Description,
		Category:    dto.Category,
		Tags:        jbytes(dto.Tags),
		Icon:        dto.Icon,
		DSType:      dto.DSType,
		URLTemplate: dto.URLTemplate,
		HTTPMethod:  dto.HTTPMethod,
		ContentType: dto.ContentType,
		TimeoutMS:   dto.TimeoutMS,
		RetryCount:  dto.RetryCount,
		BodyTemplate: func() []byte {
			if dto.BodyTemplate == nil {
				return []byte("{}")
			}
			return jbytes(dto.BodyTemplate)
		}(),
		AuthType:         firstStr(dto.AuthType, "none"),
		ParamsSchema:     jbytes(dto.ParamsSchema),
		CachePolicy:      firstStr(dto.CachePolicy, "none"),
		CacheTTLSeconds:  dto.CacheTTLSeconds,
		ResponseJSONPath: dto.ResponseJSONPath,
		ExtraConfig:      jbytes(dto.ExtraConfig),
		UploadedFileID:   dto.UploadedFileID,
		Enabled:          true,
	}
	if dto.Enabled != nil {
		ds.Enabled = *dto.Enabled
	}
	if len(dto.Headers) > 0 {
		b, _ := json.Marshal(dto.Headers)
		s, err := crypto.Seal(a.Cfg.EncryptionKey, b)
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		ds.HeadersEncrypted = &s
	}
	if len(dto.AuthConfig) > 0 {
		b, _ := json.Marshal(dto.AuthConfig)
		s, err := crypto.Seal(a.Cfg.EncryptionKey, b)
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		ds.AuthConfigEncrypted = &s
	}
	if err := a.DB.Create(&ds).Error; err != nil {
		response.Fail(c, apperr.ErrConflict)
		return
	}
	response.OK(c, a.toDSView(&ds))
}

func firstStr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func (a *App) UpdateDatasource(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto dsDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ds model.DataSource
	if err := a.DB.First(&ds, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	updates := map[string]any{
		"name": dto.Name, "description": dto.Description, "category": dto.Category,
		"tags": jbytes(dto.Tags), "icon": dto.Icon, "ds_type": dto.DSType,
		"url_template": dto.URLTemplate, "http_method": dto.HTTPMethod, "content_type": dto.ContentType,
		"timeout_ms": dto.TimeoutMS, "retry_count": dto.RetryCount,
		"body_template": jbytes(dto.BodyTemplate), "auth_type": firstStr(dto.AuthType, "none"),
		"params_schema": jbytes(dto.ParamsSchema), "cache_policy": firstStr(dto.CachePolicy, "none"),
		"cache_ttl_seconds": dto.CacheTTLSeconds, "response_jsonpath": dto.ResponseJSONPath,
		"extra_config": jbytes(dto.ExtraConfig), "uploaded_file_id": dto.UploadedFileID,
	}
	if dto.Enabled != nil {
		updates["enabled"] = *dto.Enabled
	}
	if len(dto.Headers) > 0 {
		b, _ := json.Marshal(dto.Headers)
		s, err := crypto.Seal(a.Cfg.EncryptionKey, b)
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		updates["headers_encrypted"] = s
	}
	if len(dto.AuthConfig) > 0 {
		b, _ := json.Marshal(dto.AuthConfig)
		s, err := crypto.Seal(a.Cfg.EncryptionKey, b)
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		updates["auth_config_encrypted"] = s
	}
	if err := a.DB.Model(&ds).Updates(updates).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	_ = a.DB.First(&ds, "id = ?", id).Error
	response.OK(c, a.toDSView(&ds))
}

func (a *App) CloneDatasource(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ds model.DataSource
	if err := a.DB.First(&ds, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	ds.ID = uuid.Nil
	ds.Name = ds.Name + " (copy)"
	ds.OwnerID = uid(c)
	if err := a.DB.Create(&ds).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, a.toDSView(&ds))
}

func (a *App) DeleteDatasource(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var cnt int64
	if err := a.DB.Model(&model.Agent{}).Where("datasource_id = ?", id).Count(&cnt).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	if cnt > 0 {
		response.Fail(c, apperr.WithFields(apperr.ErrConflict, gin.H{"reason": "referenced_by_agents"}))
		return
	}
	if err := a.DB.Delete(&model.DataSource{}, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) PatchDatasourceStatus(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Model(&model.DataSource{}).Where("id = ? AND owner_id = ?", id, uid(c)).Update("enabled", dto.Enabled).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) TestDatasource(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ds model.DataSource
	if err := a.DB.First(&ds, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	if body == nil {
		body = map[string]any{}
	}
	globals, _ := jsonutil.UnmarshalMap(jbytes(body["globals"]))
	params, _ := jsonutil.UnmarshalMap(jbytes(body["params"]))
	ex := datasource.NewExecutor()
	res, err := ex.Execute(c.Request.Context(), &ds, datasource.ExecuteInput{GlobalVars: globals, Params: params}, func(sealed string) (map[string]string, error) {
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
		if sealed == "" {
			return map[string]any{}, nil
		}
		b, err := crypto.Open(a.Cfg.EncryptionKey, sealed)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		return m, nil
	})
	st := "ok"
	if err != nil {
		st = "error"
	}
	_ = a.DB.Model(&ds).Updates(map[string]any{"last_tested_at": time.Now(), "last_test_status": st}).Error

	if err != nil {
		response.OK(c, gin.H{"ok": false, "error": err.Error()})
		return
	}
	response.OK(c, gin.H{"ok": true, "extracted": res.Extracted, "raw": string(res.Raw), "from_cache": res.FromCache})
}

func (a *App) ListDatasources(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.DataSource
	q := a.DB.Model(&model.DataSource{}).Where("owner_id = ?", uid(c))
	if t := c.Query("type"); t != "" {
		q = q.Where("ds_type = ?", t)
	}
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
		out = append(out, a.toDSView(&list[i]))
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) GetDatasource(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var ds model.DataSource
	if err := a.DB.First(&ds, "id = ? AND owner_id = ?", id, uid(c)).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, a.toDSView(&ds))
}

func (a *App) toDSView(ds *model.DataSource) gin.H {
	var tags []string
	_ = json.Unmarshal(ds.Tags, &tags)
	h := gin.H{
		"id": ds.ID.String(), "name": ds.Name, "description": ds.Description, "category": ds.Category,
		"tags": tags, "icon": ds.Icon, "ds_type": ds.DSType, "url_template": ds.URLTemplate,
		"http_method": ds.HTTPMethod, "content_type": ds.ContentType, "timeout_ms": ds.TimeoutMS,
		"retry_count": ds.RetryCount, "auth_type": ds.AuthType, "params_schema": json.RawMessage(ds.ParamsSchema),
		"cache_policy": ds.CachePolicy, "cache_ttl_seconds": ds.CacheTTLSeconds, "response_jsonpath": ds.ResponseJSONPath,
		"extra_config": json.RawMessage(ds.ExtraConfig), "uploaded_file_id": ds.UploadedFileID, "enabled": ds.Enabled,
		"last_tested_at": ds.LastTestedAt, "last_test_status": ds.LastTestStatus,
		"headers": gin.H{}, "auth_config": gin.H{},
	}
	if ds.HeadersEncrypted != nil && *ds.HeadersEncrypted != "" {
		h["headers"] = gin.H{"sealed": true}
	}
	if ds.AuthConfigEncrypted != nil && *ds.AuthConfigEncrypted != "" {
		h["auth_config"] = gin.H{"sealed": true}
	}
	return h
}

