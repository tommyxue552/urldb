package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/services"
	"github.com/ctwj/urldb/task"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthorizedShareHandler struct{ service *services.AuthorizedShareService; taskManager *task.TaskManager }

func NewAuthorizedShareHandler(service *services.AuthorizedShareService, taskManager *task.TaskManager) *AuthorizedShareHandler {
	return &AuthorizedShareHandler{service: service, taskManager: taskManager}
}

func (h *AuthorizedShareHandler) UpsertAuthorization(c *gin.Context) {
	resourceID, err := parseResourceID(c)
	if err != nil { ErrorResponse(c, "invalid resource ID", http.StatusBadRequest); return }
	var req struct {
		Status string `json:"status" binding:"required,oneof=pending active revoked expired"`
		EvidenceType string `json:"evidence_type" binding:"required,max=50"`
		EvidenceRef string `json:"evidence_ref" binding:"required,max=500"`
		RetentionUntil *time.Time `json:"retention_until"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { ErrorResponse(c, "invalid authorization payload: "+err.Error(), http.StatusBadRequest); return }
	username, _ := c.Get("username")
	verifiedBy, _ := username.(string)
	now := time.Now()
	authorization := &entity.ResourceAuthorization{ResourceID: resourceID, Status: req.Status, EvidenceType: req.EvidenceType, EvidenceRef: req.EvidenceRef, RetentionUntil: req.RetentionUntil, VerifiedBy: verifiedBy}
	if req.Status == "active" { authorization.VerifiedAt = &now }
	if err := h.service.UpsertAuthorization(authorization); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { ErrorResponse(c, "resource not found", http.StatusNotFound); return }
		ErrorResponse(c, "save authorization failed: "+err.Error(), http.StatusInternalServerError); return
	}
	SuccessResponse(c, gin.H{"resource_id": resourceID, "status": req.Status})
}

func (h *AuthorizedShareHandler) ListOwnedShares(c *gin.Context) {
	resourceID, err := parseResourceID(c)
	if err != nil { ErrorResponse(c, "invalid resource ID", http.StatusBadRequest); return }
	panID, err := parseOptionalUint(c.Query("pan_id")); if err != nil { ErrorResponse(c, "invalid pan_id", http.StatusBadRequest); return }
	ckID, err := parseOptionalUint(c.Query("ck_id")); if err != nil { ErrorResponse(c, "invalid ck_id", http.StatusBadRequest); return }
	shares, err := h.service.ListActiveOwnedShares(resourceID, panID, ckID)
	if err != nil { ErrorResponse(c, "list owned shares failed: "+err.Error(), http.StatusInternalServerError); return }
	SuccessResponse(c, gin.H{"list": shares, "total": len(shares)})
}

func (h *AuthorizedShareHandler) CreateTransferTask(c *gin.Context) {
	resourceID, err := parseResourceID(c)
	if err != nil { ErrorResponse(c, "invalid resource ID", http.StatusBadRequest); return }
	var req struct { PanID uint `json:"pan_id" binding:"required"`; CkID uint `json:"ck_id" binding:"required"`; Channel string `json:"channel" binding:"omitempty,max=50"` }
	if err := c.ShouldBindJSON(&req); err != nil { ErrorResponse(c, "invalid transfer request: "+err.Error(), http.StatusBadRequest); return }
	share, task, created, err := h.service.CreateAuthorizedTransferTask(services.AuthorizedTransferRequest{ResourceID: resourceID, PanID: req.PanID, CkID: req.CkID, Channel: req.Channel})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound): ErrorResponse(c, "resource not found", http.StatusNotFound)
		case errors.Is(err, services.ErrAuthorizationRequired): ErrorResponse(c, "active authorization is required", http.StatusConflict)
		case errors.Is(err, services.ErrTargetAccountMismatch): ErrorResponse(c, "target account is invalid or belongs to another platform", http.StatusBadRequest)
		default: ErrorResponse(c, "create transfer task failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if share != nil { SuccessResponse(c, gin.H{"owned_share": share, "reused": true}); return }
	if created {
		if err := h.taskManager.StartTask(task.ID); err != nil { ErrorResponse(c, "transfer task was queued but could not start: "+err.Error(), http.StatusInternalServerError); return }
	}
	SuccessResponse(c, gin.H{"task_id": task.ID, "status": task.Status, "created": created})
}

func (h *AuthorizedShareHandler) RetryTransferTask(c *gin.Context) {
	resourceID, err := parseResourceID(c)
	if err != nil { ErrorResponse(c, "invalid resource ID", http.StatusBadRequest); return }
	taskID, err := parseOptionalUint(c.Param("taskID"))
	if err != nil || taskID == 0 { ErrorResponse(c, "invalid task ID", http.StatusBadRequest); return }
	retryTask, err := h.service.RetryAuthorizedTransferTask(resourceID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound): ErrorResponse(c, "authorized transfer task not found", http.StatusNotFound)
		case errors.Is(err, services.ErrTransferTaskNotRetryable): ErrorResponse(c, "only failed authorized transfer tasks can be retried", http.StatusConflict)
		default: ErrorResponse(c, "retry transfer task failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if err := h.taskManager.StartTask(retryTask.ID); err != nil { ErrorResponse(c, "transfer task was reset but could not start: "+err.Error(), http.StatusInternalServerError); return }
	SuccessResponse(c, gin.H{"task_id": retryTask.ID, "status": "pending", "retried": true})
}

func parseResourceID(c *gin.Context) (uint, error) { return parseOptionalUint(c.Param("id")) }
func parseOptionalUint(value string) (uint, error) {
	if value == "" { return 0, nil }
	parsed, err := strconv.ParseUint(value, 10, 32); return uint(parsed), err
}
