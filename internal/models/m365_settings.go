package models

import "time"

type M365Settings struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID        string    `gorm:"column:tenant_id;not null;default:''" json:"tenantId"`
	ClientID        string    `gorm:"column:client_id;not null;default:''" json:"clientId"`
	ClientSecret    string    `gorm:"column:client_secret;not null;default:''" json:"clientSecret"`
	MailboxID       string    `gorm:"column:mailbox_id;not null;default:''" json:"mailboxId"`
	SyncInterval    string    `gorm:"column:sync_interval;not null;default:'5m'" json:"syncInterval"`
	CalendarMailbox string    `gorm:"column:calendar_mailbox;not null;default:''" json:"calendarMailbox"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (M365Settings) TableName() string { return "m365_settings" }
