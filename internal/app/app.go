package app

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/namejlt/AgentFlowPro/internal/config"
	"github.com/namejlt/AgentFlowPro/internal/engine"
	"github.com/namejlt/AgentFlowPro/internal/repository"
)

type App struct {
	Cfg    *config.Config
	DB     *gorm.DB
	Store  *repository.Store
	Hub    *engine.Hub
	Log    *zap.Logger
	runner *engine.Runner
}

func New(cfg *config.Config, db *gorm.DB) (*App, error) {
	hub := engine.NewHub()
	store := repository.New(db)
	rn := engine.NewRunner(db, store, cfg.EncryptionKey, hub, cfg.EngineMaxWorkers)
	return &App{
		Cfg:    cfg,
		DB:     db,
		Store:  store,
		Hub:    hub,
		Log:    zap.L(),
		runner: rn,
	}, nil
}

func (a *App) Runner() *engine.Runner { return a.runner }

func (a *App) HTTPServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}
}
