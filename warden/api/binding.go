package api

import (
	"errors"
	"net/http"

	"github.com/gaucho-racing/warden/warden/model"
	"github.com/gaucho-racing/warden/warden/service"
	"github.com/gin-gonic/gin"
)

type bindingRequest struct {
	GroupID        string   `json:"group_id" binding:"required"`
	GroupName      string   `json:"group_name" binding:"required"`
	LuckPermsGroup string   `json:"luckperms_group"`
	Permissions    []string `json:"permissions"`
	Weight         int      `json:"weight"`
}

func ListBindings(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	bindings, err := service.ListBindings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindings)
}

func GetBinding(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	binding, err := service.GetBinding(c.Param("id"))
	if err != nil {
		respondBindingError(c, err)
		return
	}
	c.JSON(http.StatusOK, binding)
}

func CreateBinding(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	var req bindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	binding, err := service.CreateBinding(model.GroupPermissionBinding{
		GroupID:           req.GroupID,
		GroupName:         req.GroupName,
		LuckPermsGroup:    req.LuckPermsGroup,
		Permissions:       req.Permissions,
		Weight:            req.Weight,
		CreatedByEntityID: GetRequestTokenEntityID(c),
		UpdatedByEntityID: GetRequestTokenEntityID(c),
	})
	if err != nil {
		respondBindingError(c, err)
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBindingCreated,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        binding.ID,
		Detail:          binding.GroupName + " -> " + binding.LuckPermsGroup,
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.JSON(http.StatusCreated, binding)
}

func UpdateBinding(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	var req bindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	binding, err := service.UpdateBinding(c.Param("id"), model.GroupPermissionBinding{
		GroupID:           req.GroupID,
		GroupName:         req.GroupName,
		LuckPermsGroup:    req.LuckPermsGroup,
		Permissions:       req.Permissions,
		Weight:            req.Weight,
		UpdatedByEntityID: GetRequestTokenEntityID(c),
	})
	if err != nil {
		respondBindingError(c, err)
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBindingUpdated,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        binding.ID,
		Detail:          binding.GroupName + " -> " + binding.LuckPermsGroup,
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.JSON(http.StatusOK, binding)
}

func DeleteBinding(c *gin.Context) {
	Require(c, RequestTokenCanManageBindings(c))
	binding, err := service.GetBinding(c.Param("id"))
	if err != nil {
		respondBindingError(c, err)
		return
	}
	if err := service.DeleteBinding(binding.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	service.RecordAudit(model.AuditLog{
		Action:          model.AuditActionBindingDeleted,
		ActorEntityID:   GetRequestTokenEntityID(c),
		ActorGroupNames: GetRequestTokenGroupNames(c),
		TargetID:        binding.ID,
		Detail:          binding.GroupName + " -> " + binding.LuckPermsGroup,
		RequestMethod:   c.Request.Method,
		RequestPath:     c.Request.URL.Path,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.Request.UserAgent(),
	})
	c.Status(http.StatusNoContent)
}

func respondBindingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBindingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBindingConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidPermission):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
