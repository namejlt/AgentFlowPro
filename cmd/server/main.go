package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/namejlt/AgentFlowPro/internal/api"
	"github.com/namejlt/AgentFlowPro/internal/app"
	"github.com/namejlt/AgentFlowPro/internal/config"
	"github.com/namejlt/AgentFlowPro/internal/database"
	"github.com/namejlt/AgentFlowPro/internal/pkg/logger"
)

func main() {
	cfg := config.Load()
	if err := logger.Init(cfg.GinMode); err != nil {
		panic(err)
	}
	defer logger.Sync()

	db, err := database.Connect(cfg.DatabaseURL, logger.L)
	if err != nil {
		logger.L.Fatal("db connect failed", zap.Error(err))
	}
	if err := database.AutoMigrate(db); err != nil {
		logger.L.Fatal("migrate failed", zap.Error(err))
	}

	a, err := app.New(cfg, db)
	if err != nil {
		logger.L.Fatal("app init failed", zap.Error(err))
	}
	if err := a.Seed(context.Background()); err != nil {
		logger.L.Fatal("seed failed", zap.Error(err))
	}

	gin.SetMode(cfg.GinMode)
	r := gin.New()
	if cfg.GinMode != gin.ReleaseMode {
		r.Use(gin.Logger())
		r.Use(gin.Recovery())
	}
	api.Register(r, a)
	srv := a.HTTPServer(r, cfg.HTTPAddr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			logger.L.Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
