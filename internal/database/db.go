package database

import (
	"time"

	"github.com/namejlt/AgentFlowPro/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Connect(dsn string, log *zap.Logger) (*gorm.DB, error) {
	gl := gormlogger.Default.LogMode(gormlogger.Warn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gl,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.LLMModel{},
		&model.UploadedFile{},
		&model.DataSource{},
		&model.Agent{},
		&model.Workflow{},
		&model.WorkflowVersion{},
		&model.ShareToken{},
		&model.Task{},
		&model.TaskStep{},
		&model.Report{},
		&model.SystemConfig{},
		&model.OperationLog{},
		&model.WorkflowImportSession{},
	)
}
