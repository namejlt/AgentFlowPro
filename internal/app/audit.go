package app

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/namejlt/AgentFlowPro/internal/model"
)

// AuditLog records an operation log entry.
func (a *App) AuditLog(db *gorm.DB, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, detail map[string]any, c *gin.Context) {
	var ip, ua string
	if c != nil {
		ip = c.ClientIP()
		ua = c.Request.UserAgent()
	}
	detailBytes, _ := json.Marshal(detail)
	log := model.OperationLog{
		UserID:       userID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   resourceID,
		Detail:       detailBytes,
		IP:           &ip,
		UserAgent:    &ua,
		CreatedAt:    time.Now(),
	}
	_ = db.Create(&log).Error
}

// LogCreate records a creation operation.
func (a *App) LogCreate(c *gin.Context, resourceType string, resourceID uuid.UUID, detail map[string]any) {
	uid := uid(c)
	a.AuditLog(a.DB, &uid, "create", resourceType, &resourceID, detail, c)
}

// LogUpdate records an update operation.
func (a *App) LogUpdate(c *gin.Context, resourceType string, resourceID uuid.UUID, detail map[string]any) {
	uid := uid(c)
	a.AuditLog(a.DB, &uid, "update", resourceType, &resourceID, detail, c)
}

// LogDelete records a deletion operation.
func (a *App) LogDelete(c *gin.Context, resourceType string, resourceID uuid.UUID, detail map[string]any) {
	uid := uid(c)
	a.AuditLog(a.DB, &uid, "delete", resourceType, &resourceID, detail, c)
}

// LogExecute records an execution operation.
func (a *App) LogExecute(c *gin.Context, resourceType string, resourceID uuid.UUID, detail map[string]any) {
	uid := uid(c)
	a.AuditLog(a.DB, &uid, "execute", resourceType, &resourceID, detail, c)
}

// LogLogin records a login operation.
func (a *App) LogLogin(c *gin.Context, userID uuid.UUID, success bool) {
	detail := map[string]any{"success": success}
	a.AuditLog(a.DB, &userID, "login", "user", &userID, detail, c)
}
