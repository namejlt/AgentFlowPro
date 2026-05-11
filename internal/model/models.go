package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Username     string         `gorm:"size:64;not null"`
	Email        string         `gorm:"size:255;not null;index"`
	PasswordHash string         `gorm:"not null"`
	Role         string         `gorm:"size:32;not null;default:user;index"` // user | creator | admin
	AvatarURL    *string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string { return "users" }

type LLMModel struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name             string         `gorm:"size:128;not null"`
	Vendor           string         `gorm:"size:64;not null;index"`
	Endpoint         string         `gorm:"not null"`
	ModelID          string         `gorm:"size:128;not null"`
	APIKeyEncrypted  string         `gorm:"not null"` // base64 sealed blob
	Temperature      float64        `gorm:"not null;default:0.7"`
	MaxTokens        int            `gorm:"not null;default:4096"`
	TimeoutMS        int            `gorm:"not null;default:60000"`
	RetryCount       int            `gorm:"not null;default:3"`
	StreamEnabled    bool           `gorm:"not null;default:false"`
	Enabled          bool           `gorm:"not null;default:true;index"`
	IsDefault        bool           `gorm:"not null;default:false;index"`
	CreatedBy        *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (LLMModel) TableName() string { return "llm_models" }

type UploadedFile struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	StorageKey   string         `gorm:"not null"`
	OriginalName string         `gorm:"not null"`
	MimeType     *string
	SizeBytes    int64          `gorm:"not null;default:0"`
	SHA256       *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (UploadedFile) TableName() string { return "uploaded_files" }

type DataSource struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID              uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name                 string         `gorm:"size:128;not null"`
	Description          *string
	Category             *string        `gorm:"size:64;index"`
	Tags                 []byte         `gorm:"type:jsonb;not null"` // json array
	Icon                 *string        `gorm:"size:64"`
	DSType               string         `gorm:"size:32;not null;index"` // HTTP_GET | HTTP_POST | FILE_UPLOAD | MANUAL_INPUT | WEBSOCKET_STREAM
	URLTemplate          *string
	HTTPMethod           *string        `gorm:"size:16"`
	ContentType          *string        `gorm:"size:128"`
	TimeoutMS            int            `gorm:"not null;default:30000"`
	RetryCount           int            `gorm:"not null;default:2"`
	HeadersEncrypted     *string        // sealed json map string->string
	BodyTemplate         []byte         `gorm:"type:jsonb"`
	AuthType             string         `gorm:"size:32;not null;default:none"`
	AuthConfigEncrypted  *string
	ParamsSchema         []byte         `gorm:"type:jsonb;not null"`
	CachePolicy          string         `gorm:"size:32;not null;default:none"` // none | ttl | fixed
	CacheTTLSeconds      *int
	ResponseJSONPath     *string
	ExtraConfig          []byte         `gorm:"type:jsonb;not null"`
	UploadedFileID       *uuid.UUID     `gorm:"type:uuid"`
	Enabled              bool           `gorm:"not null;default:true;index"`
	LastTestedAt         *time.Time
	LastTestStatus       *string        `gorm:"size:32"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (DataSource) TableName() string { return "data_sources" }

type Agent struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name           string         `gorm:"size:128;not null"`
	RoleDesc       *string
	Tags           []byte         `gorm:"type:jsonb;not null"`
	Icon           *string
	SystemPrompt   string         `gorm:"type:text;not null"`
	LLMModelID     *uuid.UUID     `gorm:"type:uuid;index"`
	DataSourceID   *uuid.UUID     `gorm:"type:uuid;index"`
	ParamMappings  []byte         `gorm:"type:jsonb;not null"`
	OutputFormat   string         `gorm:"size:16;not null;default:markdown"`
	OutputLang     string         `gorm:"size:16;not null;default:zh-CN"`
	MaxOutputChars int            `gorm:"not null;default:8000"`
	Enabled        bool           `gorm:"not null;default:true;index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (Agent) TableName() string { return "agents" }

type Workflow struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID         uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"size:256;not null"`
	Description     *string
	Tags            []byte         `gorm:"type:jsonb;not null"`
	GlobalParams    []byte         `gorm:"type:jsonb;not null"`
	Nodes           []byte         `gorm:"type:jsonb;not null"`
	Edges           []byte         `gorm:"type:jsonb;not null"`
	ExecConfig      []byte         `gorm:"type:jsonb;not null"`
	DefaultModelID  *uuid.UUID     `gorm:"type:uuid"`
	Version         int            `gorm:"not null;default:1"`
	Visibility      string         `gorm:"size:16;not null;default:private;index"`
	ShareCode       *string        `gorm:"size:32;uniqueIndex"`
	Archived        bool           `gorm:"not null;default:false"`
	LastRunAt       *time.Time
	RunCount         int64         `gorm:"not null;default:0"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (Workflow) TableName() string { return "workflows" }

type WorkflowVersion struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkflowID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Version    int            `gorm:"not null"`
	Snapshot   []byte         `gorm:"type:jsonb;not null"`
	CreatedBy  *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (WorkflowVersion) TableName() string { return "workflow_versions" }

type ShareToken struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkflowID  uuid.UUID      `gorm:"type:uuid;not null;index"`
	Code        string         `gorm:"size:64;not null;uniqueIndex"`
	Visibility  string         `gorm:"size:16;not null;default:shared"`
	ExpiresAt   *time.Time
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ShareToken) TableName() string { return "share_tokens" }

type Task struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"`
	WorkflowID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	WorkflowVersion  int            `gorm:"not null"`
	OwnerID          uuid.UUID      `gorm:"type:uuid;not null;index"`
	InputParams      []byte         `gorm:"type:jsonb;not null"`
	Mode             string         `gorm:"size:32;not null;default:normal"` // normal | debug
	Status           string         `gorm:"size:32;not null;default:pending;index"`
	ReportID         *uuid.UUID     `gorm:"type:uuid"`
	ErrorMessage       *string
	ErrorStepID        *string        `gorm:"size:128"`
	StartedAt          *time.Time
	FinishedAt         *time.Time
	DurationMS         *int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (Task) TableName() string { return "tasks" }

type TaskStep struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TaskID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	NodeID       string         `gorm:"size:128;not null;index"`
	NodeType     string         `gorm:"size:64;not null"`
	AgentID      *uuid.UUID     `gorm:"type:uuid"`
	Status       string         `gorm:"size:32;not null;default:pending;index"`
	DebateRound  *int
	Input        []byte         `gorm:"type:jsonb"`
	Output       []byte         `gorm:"type:jsonb"`
	TokensUsed   *int
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (TaskStep) TableName() string { return "task_steps" }

type Report struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TaskID        uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	WorkflowID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	OwnerID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	Title         string         `gorm:"type:text;not null"`
	ContentMD     string         `gorm:"type:text;not null"`
	AgentOutputs  []byte         `gorm:"type:jsonb;not null"`
	DebateLogs    []byte         `gorm:"type:jsonb;not null"`
	RiskReviews   []byte         `gorm:"type:jsonb;not null"`
	ExecLogs      []byte         `gorm:"type:jsonb;not null"`
	InputSnapshot []byte         `gorm:"type:jsonb;not null"`
	Status        string         `gorm:"size:32;not null;default:completed"`
	Archived      bool           `gorm:"not null;default:false;index"`
	TotalTokens   *int
	DurationMS    *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (Report) TableName() string { return "reports" }

type SystemConfig struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Key         string         `gorm:"size:128;not null;uniqueIndex;column:cfg_key"`
	Value       []byte         `gorm:"type:jsonb;not null"`
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (SystemConfig) TableName() string { return "system_config" }

type OperationLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID       *uuid.UUID     `gorm:"type:uuid;index"`
	Action       string         `gorm:"size:64;not null;index"`
	ResourceType *string        `gorm:"size:64;index"`
	ResourceID   *uuid.UUID     `gorm:"type:uuid"`
	Detail       []byte         `gorm:"type:jsonb"`
	IP           *string
	UserAgent    *string
	CreatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (OperationLog) TableName() string { return "operation_logs" }

type WorkflowImportSession struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null;index"`
	Payload     []byte         `gorm:"type:jsonb;not null"`
	MatchReport []byte         `gorm:"type:jsonb;not null"`
	Status      string         `gorm:"size:32;not null;default:pending"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (WorkflowImportSession) TableName() string { return "workflow_import_sessions" }

func assignUUID(id *uuid.UUID) {
	if id != nil && *id == uuid.Nil {
		*id = uuid.New()
	}
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&u.ID)
	return nil
}
func (m *LLMModel) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&m.ID)
	return nil
}
func (f *UploadedFile) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&f.ID)
	return nil
}
func (d *DataSource) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&d.ID)
	return nil
}
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&a.ID)
	return nil
}
func (w *Workflow) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&w.ID)
	return nil
}
func (w *WorkflowVersion) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&w.ID)
	return nil
}
func (s *ShareToken) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&s.ID)
	return nil
}
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&t.ID)
	return nil
}
func (t *TaskStep) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&t.ID)
	return nil
}
func (r *Report) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&r.ID)
	return nil
}
func (s *SystemConfig) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&s.ID)
	return nil
}
func (o *OperationLog) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&o.ID)
	return nil
}
func (w *WorkflowImportSession) BeforeCreate(tx *gorm.DB) error {
	assignUUID(&w.ID)
	return nil
}
