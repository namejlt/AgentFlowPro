package app

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/namejlt/AgentFlowPro/internal/model"
)

func (a *App) Seed(ctx context.Context) error {
	var count int64
	if err := a.DB.WithContext(ctx).Model(&model.User{}).Where("email = ?", "admin@agentflow.local").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := model.User{
		Username:     "admin",
		Email:        "admin@agentflow.local",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	if err := a.DB.WithContext(ctx).Create(&u).Error; err != nil {
		return err
	}
	// default LLM placeholder (disabled until user configures real key)
	m := model.LLMModel{
		Name:            "Placeholder",
		Vendor:          "openai-compatible",
		Endpoint:        "https://api.openai.com/v1",
		ModelID:         "gpt-4o-mini",
		APIKeyEncrypted: strings.Repeat("A", 32), // invalid until replaced - user must update
		Enabled:         false,
		IsDefault:       false,
	}
	_ = a.DB.WithContext(ctx).Create(&m).Error
	return nil
}
