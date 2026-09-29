package service

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gaucho-racing/ulid-go"
	"github.com/gaucho-racing/warden/warden/database"
	"github.com/gaucho-racing/warden/warden/model"
	"gorm.io/gorm"
)

// ManagedGroupPrefix marks the LuckPerms groups Warden owns. The reconcile
// only ever adds or removes groups carrying this prefix, so permissions an
// admin grants by hand in-game survive every sync untouched. Changing this
// constant orphans every group Warden previously managed.
const ManagedGroupPrefix = "warden-"

var (
	ErrBindingNotFound   = errors.New("binding not found")
	ErrBindingConflict   = errors.New("a binding already exists for this group")
	ErrInvalidPermission = errors.New("permission nodes may only contain letters, digits, dots, dashes, underscores, asterisks and colons")
)

var permissionNodeRe = regexp.MustCompile(`^[A-Za-z0-9_.*:-]+$`)

// ManagedGroupName derives the LuckPerms group for a Sentinel group name.
// Lowercased and slugified because LuckPerms group names are case-sensitive
// and awkward with spaces.
func ManagedGroupName(sentinelGroupName string) string {
	slug := strings.ToLower(strings.TrimSpace(sentinelGroupName))
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	return ManagedGroupPrefix + slug
}

func IsManagedGroup(luckPermsGroup string) bool {
	return strings.HasPrefix(luckPermsGroup, ManagedGroupPrefix)
}

func ListBindings() ([]model.GroupPermissionBinding, error) {
	bindings := []model.GroupPermissionBinding{}
	if err := database.DB.Order("weight desc, group_name asc").Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

func GetBinding(id string) (model.GroupPermissionBinding, error) {
	var binding model.GroupPermissionBinding
	if err := database.DB.Where("id = ?", id).First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.GroupPermissionBinding{}, ErrBindingNotFound
		}
		return model.GroupPermissionBinding{}, err
	}
	return binding, nil
}

func CreateBinding(binding model.GroupPermissionBinding) (model.GroupPermissionBinding, error) {
	if err := validateBinding(&binding); err != nil {
		return model.GroupPermissionBinding{}, err
	}
	binding.ID = ulid.Make().Prefixed("bind")
	if err := database.DB.Create(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.GroupPermissionBinding{}, ErrBindingConflict
		}
		return model.GroupPermissionBinding{}, err
	}
	return binding, nil
}

func UpdateBinding(id string, binding model.GroupPermissionBinding) (model.GroupPermissionBinding, error) {
	existing, err := GetBinding(id)
	if err != nil {
		return model.GroupPermissionBinding{}, err
	}
	if err := validateBinding(&binding); err != nil {
		return model.GroupPermissionBinding{}, err
	}
	existing.GroupID = binding.GroupID
	existing.GroupName = binding.GroupName
	existing.LuckPermsGroup = binding.LuckPermsGroup
	existing.Permissions = binding.Permissions
	existing.Weight = binding.Weight
	existing.UpdatedByEntityID = binding.UpdatedByEntityID
	if err := database.DB.Save(&existing).Error; err != nil {
		return model.GroupPermissionBinding{}, err
	}
	return existing, nil
}

func DeleteBinding(id string) error {
	return database.DB.Where("id = ?", id).Delete(&model.GroupPermissionBinding{}).Error
}

func validateBinding(binding *model.GroupPermissionBinding) error {
	binding.GroupID = strings.TrimSpace(binding.GroupID)
	binding.GroupName = strings.TrimSpace(binding.GroupName)
	if binding.GroupID == "" || binding.GroupName == "" {
		return errors.New("group_id and group_name are required")
	}
	// An empty LuckPerms group means "derive it" — the common case, and it
	// keeps the managed prefix invariant from depending on the caller.
	if strings.TrimSpace(binding.LuckPermsGroup) == "" {
		binding.LuckPermsGroup = ManagedGroupName(binding.GroupName)
	}
	if !IsManagedGroup(binding.LuckPermsGroup) {
		return errors.New("luckperms_group must start with " + ManagedGroupPrefix)
	}
	cleaned := make([]string, 0, len(binding.Permissions))
	for _, node := range binding.Permissions {
		node = strings.TrimSpace(node)
		if node == "" {
			continue
		}
		if !permissionNodeRe.MatchString(node) {
			return ErrInvalidPermission
		}
		cleaned = append(cleaned, node)
	}
	binding.Permissions = cleaned
	return nil
}
