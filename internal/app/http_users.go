package app

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/pagination"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

func (a *App) ListUsers(c *gin.Context) {
	p := pagination.FromQuery(c)
	var list []model.User
	q := a.DB.Model(&model.User{})
	if kw := p.Keyword; kw != "" {
		q = q.Where("username ILIKE ? OR email ILIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("created_at desc").Offset(p.Offset).Limit(p.PageSize).Find(&list).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, u := range list {
		out = append(out, gin.H{"id": u.ID.String(), "username": u.Username, "email": u.Email, "role": u.Role})
	}
	response.OKMeta(c, out, pagination.Meta(p, total))
}

func (a *App) GetUser(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var u model.User
	if err := a.DB.First(&u, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"id": u.ID.String(), "username": u.Username, "email": u.Email, "role": u.Role})
}

func (a *App) PatchUser(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	var dto struct {
		Role     *string `json:"role"`
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	updates := map[string]any{}
	if dto.Role != nil {
		updates["role"] = *dto.Role
	}
	if dto.Password != nil && *dto.Password != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(*dto.Password), bcrypt.DefaultCost)
		if err != nil {
			response.Fail(c, apperr.ErrInternal)
			return
		}
		updates["password_hash"] = string(b)
	}
	if len(updates) == 0 {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	if err := a.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
