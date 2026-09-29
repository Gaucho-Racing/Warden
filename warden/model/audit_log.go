package model

import "time"

const (
	AuditActionLinkTokenIssued   = "link_token.issued"
	AuditActionAccountLinked     = "account.linked"
	AuditActionAccountUnlinked   = "account.unlinked"
	AuditActionBindingCreated    = "binding.created"
	AuditActionBindingUpdated    = "binding.updated"
	AuditActionBindingDeleted    = "binding.deleted"
	AuditActionPermissionsServed = "permissions.served"
)

type AuditLog struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	Action          string    `json:"action" gorm:"index"`
	ActorEntityID   string    `json:"actor_entity_id" gorm:"index"`
	ActorGroupNames []string  `json:"actor_group_names" gorm:"type:jsonb;serializer:json"`
	MinecraftUUID   string    `json:"minecraft_uuid" gorm:"index"`
	MinecraftName   string    `json:"minecraft_name"`
	TargetID        string    `json:"target_id" gorm:"index"`
	Detail          string    `json:"detail"`
	RequestMethod   string    `json:"request_method"`
	RequestPath     string    `json:"request_path"`
	IPAddress       string    `json:"ip_address"`
	UserAgent       string    `json:"user_agent"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

func (AuditLog) TableName() string {
	return "warden_audit_log"
}
