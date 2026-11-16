// This file defines the GORM Session and RefreshToken models mapped to the "sessions"
// and "refresh_tokens" PostgreSQL tables for login, session management, and refresh token rotation.
package models

import "time"

// Session represents an authenticated user session persisted in the "sessions" table.
// Linked to User and a chain of RefreshToken records via foreign keys with ON DELETE CASCADE.
type Session struct {
	ID            string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID        string         `gorm:"column:user_id;type:uuid;not null;index" json:"userId"`
	User          User           `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	UserAgent     *string        `gorm:"column:user_agent" json:"userAgent,omitempty"`
	IPAddress     *string        `gorm:"column:ip_address" json:"ipAddress,omitempty"`
	Revoked       bool           `gorm:"column:revoked;default:false;not null" json:"revoked"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	ExpiresAt     time.Time      `gorm:"column:expires_at;not null" json:"expiresAt"`
}

// TableName specifies the PostgreSQL table name for Session records.
func (Session) TableName() string {
	return "sessions"
}

// RefreshToken represents a hashed refresh token belonging to a Session in the "refresh_tokens" table.
// Supports single-use token rotation and reuse detection via ReplacedByTokenID and Revoked flags.
type RefreshToken struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	// TokenHash stores the SHA-256 hex digest of the raw refresh token secret and is excluded from JSON output.
	TokenHash         string    `gorm:"column:token_hash;not null" json:"-"`
	SessionID         string    `gorm:"column:session_id;type:uuid;not null;index" json:"sessionId"`
	Session           Session   `gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Revoked           bool      `gorm:"column:revoked;default:false;not null" json:"revoked"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	ExpiresAt         time.Time `gorm:"column:expires_at;not null" json:"expiresAt"`
	ReplacedByTokenID *string   `gorm:"column:replaced_by_token_id" json:"replacedByTokenId,omitempty"`
}

// TableName specifies the PostgreSQL table name for RefreshToken records.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
