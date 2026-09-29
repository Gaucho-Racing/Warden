package model

import "time"

// GroupPermissionBinding maps one Sentinel group onto what it grants inside
// Minecraft. The direction matters: Sentinel groups are the source,
// Minecraft permissions are the sink. Nothing in this repo ever writes group
// membership back into Sentinel — Warden's service account could not do so
// even if it tried.
//
// LuckPermsGroup is the managed LuckPerms group this binding drives. Warden
// only ever adds and removes groups it manages (see service.ManagedGroupName),
// so permissions granted by hand in-game survive a reconcile untouched.
type GroupPermissionBinding struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	GroupID           string    `json:"group_id" gorm:"uniqueIndex"`
	GroupName         string    `json:"group_name"`
	LuckPermsGroup    string    `json:"luckperms_group"`
	Permissions       []string  `json:"permissions" gorm:"type:jsonb;serializer:json"`
	Weight            int       `json:"weight"`
	CreatedByEntityID string    `json:"created_by_entity_id" gorm:"index"`
	UpdatedByEntityID string    `json:"updated_by_entity_id" gorm:"index"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (GroupPermissionBinding) TableName() string {
	return "warden_group_permission_binding"
}
