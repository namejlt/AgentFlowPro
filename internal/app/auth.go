package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/namejlt/AgentFlowPro/internal/auth"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *App) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.WithFields(apperr.ErrBadRequest, gin.H{"error": err.Error()}))
		return
	}
	var u model.User
	if err := a.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&u).Error; err != nil {
		a.LogLogin(c, uuid.Nil, false)
		response.Fail(c, apperr.ErrUnauthorized)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		a.LogLogin(c, u.ID, false)
		response.Fail(c, apperr.ErrUnauthorized)
		return
	}
	tok, err := auth.SignJWT([]byte(a.Cfg.JWTSecret), u.ID, u.Role, a.Cfg.JWTExpire)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	now := time.Now()
	_ = a.DB.Model(&u).Update("last_login_at", now).Error
	a.LogLogin(c, u.ID, true)
	response.OK(c, gin.H{
		"access_token": tok,
		"expires_in":   int(a.Cfg.JWTExpire.Seconds()),
		"user": gin.H{
			"id": u.ID.String(), "username": u.Username, "email": u.Email, "role": u.Role,
		},
	})
}

func (a *App) Refresh(c *gin.Context) {
	// reuse Authorization bearer and issue new token
	h := c.GetHeader("Authorization")
	if len(h) < 8 {
		response.Fail(c, apperr.ErrUnauthorized)
		return
	}
	tok := strings.TrimSpace(h[7:])
	cl, err := auth.ParseJWT([]byte(a.Cfg.JWTSecret), tok)
	if err != nil {
		response.Fail(c, apperr.ErrUnauthorized)
		return
	}
	id, _ := parseUUID(cl.UserID)
	nt, err := auth.SignJWT([]byte(a.Cfg.JWTSecret), id, cl.Role, a.Cfg.JWTExpire)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	response.OK(c, gin.H{"access_token": nt, "expires_in": int(a.Cfg.JWTExpire.Seconds())})
}

func (a *App) Logout(c *gin.Context) {
	c.Status(http.StatusOK)
	response.OK(c, gin.H{"ok": true})
}

func (a *App) Me(c *gin.Context) {
	id := uid(c)
	var u model.User
	if err := a.DB.First(&u, "id = ?", id).Error; err != nil {
		response.Fail(c, apperr.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"id": u.ID.String(), "username": u.Username, "email": u.Email, "role": u.Role, "last_login_at": u.LastLoginAt, "created_at": u.CreatedAt})
}
