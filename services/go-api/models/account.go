// This file defines the GORM Account model mapped to the "accounts" PostgreSQL table
// for federated OAuth identity linking (such as Google OAuth) to a User record.
package models

import "time"

// Account represents an external OAuth provider account linked to a User in the "accounts" table.
// Enforces uniqueness across (provider, provider_user_id) via the idx_provider_user composite unique index.
type Account struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID         string    `gorm:"column:user_id;type:uuid;not null;index" json:"userId"`
	User           User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Provider       string    `gorm:"column:provider;size:50;not null;uniqueIndex:idx_provider_user" json:"provider"`
	ProviderUserID string    `gorm:"column:provider_user_id;size:255;not null;uniqueIndex:idx_provider_user" json:"providerUserId"`
	ProviderEmail  string    `gorm:"column:provider_email;size:255;not null" json:"providerEmail"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName specifies the PostgreSQL table name for Account records.
func (Account) TableName() string {
	return "accounts"
}
