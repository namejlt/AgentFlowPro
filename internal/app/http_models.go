package app

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/namejlt/AgentFlowPro/internal/crypto"
	"github.com/namejlt/AgentFlowPro/internal/llm"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

type llmDTO struct {
	Name          string  `json:"name"`
	Vendor        string  `json:"vendor"`
	Endpoint      string  `json:"endpoint"`
	ModelID       string  `json:"model_id"`
	APIKey        string  `json:"api_key"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	TimeoutMS     int     `json:"timeout_ms"`
	RetryCount    int     `json:"retry_count"`
	StreamEnabled bool    `json:"stream_enabled"`
	Enabled       bool    `json:"enabled"`
}

func (a *App) CreateModel(c *gin.Context) {
	var dto llmDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.WithFields(apperr.ErrBadRequest, gin.H{"error": err.Error()}))
		return
	}
	sealed, err := crypto.Seal(a.Cfg.EncryptionKey, []byte(strings.TrimSpace(dto.APIKey)))
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	m := model.LLMModel{
		Name: dto.Name, Vendor: dto.Vendor, Endpoint: dto.Endpoint, ModelID: dto.ModelID,
		APIKeyEncrypted: sealed, Temperature: dto.Temperature, MaxTokens: dto.MaxTokens,
		TimeoutMS: dto.TimeoutMS, RetryCount: dto.RetryCount, StreamEnabled: dto.StreamEnabled,
		Enabled: dto.Enabled, IsDefault: false,
	}
	uid := uid(c)
	m.CreatedBy = &uid
	if err := a.DB.Create(&m).Error; err != nil {
		response.Fail(c, apperr.ErrConflict)
		return
	}
	response.OK(c, a.toLLMView(&m, true))
}

func (a *App) UpdateModel(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto llmDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.WithFields(apperr.ErrBadRequest, gin.H{"error": err.Error()}))
		return
	}
	var m model.LLMModel
	if err := a.DB.First(&m, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	updates := map[string]any{
		"name": dto.Name, "vendor": dto.Vendor, "endpoint": dto.Endpoint, "model_id": dto.ModelID,
		"temperature": dto.Temperature, "max_tokens": dto.MaxTokens, "timeout_ms": dto.TimeoutMS,
		"retry_count": dto.RetryCount, "stream_enabled": dto.StreamEnabled, "enabled": dto.Enabled,
	}
	if strings.TrimSpace(dto.APIKey) != "" {
		sealed, err := crypto.Seal(a.Cfg.EncryptionKey, []byte(strings.TrimSpace(dto.APIKey)))
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		updates["api_key_encrypted"] = sealed
	}
	if err := a.DB.Model(&m).Updates(updates).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	_ = a.DB.First(&m, "id = ?", id).Error
	response.OK(c, a.toLLMView(&m, true))
}

func (a *App) DeleteModel(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Delete(&model.LLMModel{}, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) ListModels(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.LLMModel
	q := a.DB.Model(&model.LLMModel{})
	if kw := p.Keyword; kw != "" {
		q = q.Where("name ILIKE ? OR vendor ILIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, a.toLLMView(&list[i], false))
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) GetModel(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var m model.LLMModel
	if err := a.DB.First(&m, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, a.toLLMView(&m, false))
}

// TestModel tests the connectivity of an LLM model by sending a ping request.
func (a *App) TestModel(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var m model.LLMModel
	if err := a.DB.First(&m, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	key, err := crypto.Open(a.Cfg.EncryptionKey, m.APIKeyEncrypted)
	if err != nil {
		response.OK(c, gin.H{"success": false, "latency_ms": 0, "error": "解密 API Key 失败: " + err.Error()})
		return
	}
	start := time.Now()
	client := llm.NewClient()
	_, err = client.Chat(c.Request.Context(), llm.ChatOpts{
		Endpoint: m.Endpoint, APIKey: string(key), Model: m.ModelID,
		Messages: []llm.Message{{Role: "user", Content: "ping"}},
		Temp: 0, MaxTokens: 8, Timeout: 15 * time.Second, Retries: 0, Stream: false,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		response.OK(c, gin.H{"success": false, "latency_ms": latency, "error": err.Error()})
		return
	}
	response.OK(c, gin.H{"success": true, "latency_ms": latency})
}

func (a *App) SetDefaultModel(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	_ = a.DB.Model(&model.LLMModel{}).Where("1=1").Update("is_default", false).Error
	if err := a.DB.Model(&model.LLMModel{}).Where("id = ?", id).Update("is_default", true).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) toLLMView(m *model.LLMModel, includeSecrets bool) gin.H {
	h := gin.H{
		"id": m.ID.String(), "name": m.Name, "vendor": m.Vendor, "endpoint": m.Endpoint, "model_id": m.ModelID,
		"temperature": m.Temperature, "max_tokens": m.MaxTokens, "timeout_ms": m.TimeoutMS, "retry_count": m.RetryCount,
		"stream_enabled": m.StreamEnabled, "enabled": m.Enabled, "is_default": m.IsDefault,
		"created_at": m.CreatedAt, "updated_at": m.UpdatedAt,
	}
	if includeSecrets {
		if plain, err := crypto.Open(a.Cfg.EncryptionKey, m.APIKeyEncrypted); err == nil {
			h["api_key"] = string(plain)
		}
		h["api_key_masked"] = "****"
	} else {
		h["api_key"] = "****"
		h["api_key_masked"] = "****"
	}
	return h
}

// --- users ---

type userDTO struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (a *App) CreateUser(c *gin.Context) {
	var dto userDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	hashb, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	hash := string(hashb)
	u := model.User{Username: dto.Username, Email: dto.Email, PasswordHash: hash, Role: dto.Role}
	if u.Role == "" {
		u.Role = "user"
	}
	if err := a.DB.Create(&u).Error; err != nil {
		response.Fail(c, apperr.ErrConflict)
		return
	}
	response.OK(c, gin.H{"id": u.ID.String(), "username": u.Username, "email": u.Email, "role": u.Role})
}
