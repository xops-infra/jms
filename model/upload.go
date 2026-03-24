package model

import "time"

// UploadSession 用于分片上传元数据
type UploadSession struct {
	ID              string     `json:"id" gorm:"column:id;primary_key;not null"`
	Host            string     `json:"host" gorm:"column:host;not null"`
	SSHUser         *string    `json:"ssh_user" gorm:"column:ssh_user"`
	SSHKey          *string    `json:"ssh_key" gorm:"column:ssh_key"`
	Path            string     `json:"path" gorm:"column:path;not null"`
	Size            int64      `json:"size" gorm:"column:size;not null"`
	ChunkSize       int64      `json:"chunk_size" gorm:"column:chunk_size;not null"`
	ChunkCount      int        `json:"chunk_count" gorm:"column:chunk_count;not null"`
	CompletedChunks int        `json:"completed_chunks" gorm:"column:completed_chunks;not null"`
	SHA256          *string    `json:"sha256" gorm:"column:sha256"`
	Status          string     `json:"status" gorm:"column:status;not null"` // init,in_progress,completed,aborted
	ExpiresAt       time.Time  `json:"expires_at" gorm:"column:expires_at;not null"`
	CreatedAt       *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (UploadSession) TableName() string {
	return "jms_upload_sessions"
}

type UploadInitRequest struct {
	Host      string  `json:"host" binding:"required"`
	User      *string `json:"user"`
	Key       *string `json:"key"`
	Path      string  `json:"path" binding:"required"`
	Size      int64   `json:"size" binding:"required"`
	SHA256    *string `json:"sha256"`
	ChunkSize int64   `json:"chunk_size"`
}

type UploadInitResponse struct {
	UploadID  string `json:"upload_id"`
	ChunkSize int64  `json:"chunk_size"`
	ExpiresAt int64  `json:"expires_at"`
}

type UploadCompleteRequest struct {
	UploadID    string  `json:"upload_id" binding:"required"`
	TotalChunks int     `json:"total_chunks" binding:"required"`
	SHA256      *string `json:"sha256"`
}

type UploadAbortRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}
