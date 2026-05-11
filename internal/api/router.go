package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/namejlt/AgentFlowPro/internal/api/middleware"
	v1 "github.com/namejlt/AgentFlowPro/internal/api/v1"
	"github.com/namejlt/AgentFlowPro/internal/app"
)

func Register(r *gin.Engine, a *app.App) {
	r.Use(middleware.RequestID(), middleware.Recovery(zap.L()), middleware.CORS())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	pub := r.Group("/api/v1")
	{
		pub.POST("/auth/login", a.Login)
		pub.POST("/auth/refresh", a.Refresh)
		pub.POST("/auth/logout", a.Logout)
	}

	prv := r.Group("/api/v1")
	prv.Use(middleware.JWT(middleware.JWTConfig{Secret: []byte(a.Cfg.JWTSecret)}))
	{
		prv.GET("/auth/me", a.Me)
		v1.Register(prv, a)
	}
}
