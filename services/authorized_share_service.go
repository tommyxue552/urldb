package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrAuthorizationRequired = errors.New("resource has no active authorization")
	ErrTargetAccountMismatch = errors.New("target account does not belong to target platform")
	ErrTransferTaskNotRetryable = errors.New("authorized transfer task is not retryable")
)

// AuthorizedShareService owns the Phase 1 persistence boundary. It never calls
// a cloud provider: later phases attach a processor to the queued task.
type AuthorizedShareService struct {
	db *gorm.DB
}

// RetryAuthorizedTransferTask resets the one failed item of an authorized
// transfer. The stable idempotency key remains unchanged, so retries cannot
// create a duplicate transfer for the same resource/account target.
func (s *AuthorizedShareService) RetryAuthorizedTransferTask(resourceID, taskID uint) (*entity.Task, error) {
	var task entity.Task
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND type = ?", taskID, entity.TaskTypeAuthorizedTransfer).First(&task).Error; err != nil {
			return err
		}
		var input AuthorizedTransferRequest
		if err := json.Unmarshal([]byte(task.Config), &input); err != nil || input.ResourceID != resourceID {
			return ErrTransferTaskNotRetryable
		}
		if task.Status != entity.TaskStatusFailed {
			return ErrTransferTaskNotRetryable
		}
		result := tx.Model(&entity.Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"status": "pending", "message": "Retry queued by administrator", "processed_items": 0,
			"success_items": 0, "failed_items": 0, "progress": 0, "started_at": nil, "completed_at": nil,
		})
		if result.Error != nil { return result.Error }
		result = tx.Model(&entity.TaskItem{}).Where("task_id = ? AND status = ?", task.ID, entity.TaskItemStatusFailed).Updates(map[string]interface{}{
			"status": "pending", "output_data": "", "error_message": "", "processed_at": nil,
		})
		if result.Error != nil { return result.Error }
		if result.RowsAffected != 1 { return ErrTransferTaskNotRetryable }
		task.Status = entity.TaskStatusPending
		return nil
	})
	if err != nil { return nil, err }
	return &task, nil
}

func NewAuthorizedShareService(db *gorm.DB) *AuthorizedShareService {
	return &AuthorizedShareService{db: db}
}

func (s *AuthorizedShareService) UpsertAuthorization(authorization *entity.ResourceAuthorization) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var resource entity.Resource
		if err := tx.First(&resource, authorization.ResourceID).Error; err != nil {
			return err
		}

		var existing entity.ResourceAuthorization
		err := tx.Where("resource_id = ?", authorization.ResourceID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(authorization).Error
		}
		if err != nil {
			return err
		}

		authorization.ID = existing.ID
		return tx.Model(&existing).Updates(map[string]interface{}{
			"status":          authorization.Status,
			"evidence_type":   authorization.EvidenceType,
			"evidence_ref":    authorization.EvidenceRef,
			"retention_until": authorization.RetentionUntil,
			"verified_by":     authorization.VerifiedBy,
			"verified_at":     authorization.VerifiedAt,
		}).Error
	})
}

func (s *AuthorizedShareService) ListActiveOwnedShares(resourceID, panID, ckID uint) ([]entity.OwnedShare, error) {
	query := s.db.Where("resource_id = ? AND status = ?", resourceID, "active").
		Where("expires_at IS NULL OR expires_at > ?", time.Now())
	if panID != 0 {
		query = query.Where("pan_id = ?", panID)
	}
	if ckID != 0 {
		query = query.Where("ck_id = ?", ckID)
	}
	var shares []entity.OwnedShare
	return shares, query.Order("updated_at DESC").Find(&shares).Error
}

type AuthorizedTransferRequest struct {
	ResourceID uint `json:"resource_id"`
	PanID      uint `json:"pan_id"`
	CkID       uint `json:"ck_id"`
	Channel    string `json:"channel"`
}

// CreateAuthorizedTransferTask returns an existing active owned share first;
// otherwise it atomically creates or reuses one pending task for this target.
func (s *AuthorizedShareService) CreateAuthorizedTransferTask(req AuthorizedTransferRequest) (*entity.OwnedShare, *entity.Task, bool, error) {
	if req.Channel == "" {
		req.Channel = "system"
	}
	if req.ResourceID == 0 || req.PanID == 0 || req.CkID == 0 {
		return nil, nil, false, fmt.Errorf("resource_id, pan_id and ck_id are required")
	}

	var resultShare *entity.OwnedShare
	var resultTask *entity.Task
	created := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var resource entity.Resource
		if err := tx.First(&resource, req.ResourceID).Error; err != nil {
			return err
		}

		var authorization entity.ResourceAuthorization
		if err := tx.Where("resource_id = ? AND status = ?", req.ResourceID, "active").First(&authorization).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAuthorizationRequired
			}
			return err
		}
		if authorization.RetentionUntil != nil && !authorization.RetentionUntil.After(time.Now()) {
			return ErrAuthorizationRequired
		}

		var account entity.Cks
		if err := tx.Where("id = ? AND pan_id = ? AND is_valid = ?", req.CkID, req.PanID, true).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTargetAccountMismatch
			}
			return err
		}

		var share entity.OwnedShare
		err := tx.Where("resource_id = ? AND pan_id = ? AND ck_id = ? AND status = ?", req.ResourceID, req.PanID, req.CkID, "active").
			Where("expires_at IS NULL OR expires_at > ?", time.Now()).First(&share).Error
		if err == nil {
			resultShare = &share
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		key := fmt.Sprintf("authorized-transfer:%d:%d:%d", req.ResourceID, req.PanID, req.CkID)
		var existingTask entity.Task
		err = tx.Where("idempotency_key = ?", key).First(&existingTask).Error
		if err == nil {
			resultTask = &existingTask
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		input, err := json.Marshal(req)
		if err != nil {
			return err
		}
		task := entity.Task{
			Title:          fmt.Sprintf("Authorized transfer for resource %d", req.ResourceID),
			Name:           "authorized_transfer",
			Type:           entity.TaskTypeAuthorizedTransfer,
			Status:         entity.TaskStatusPending,
			Description:    "Queued after authorization verification; execution is enabled in the transfer retry phase.",
			TotalItems:     1,
			Config:         string(input),
			IdempotencyKey: &key,
		}
		create := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "idempotency_key"}},
			DoNothing: true,
		}).Create(&task)
		if create.Error != nil {
			return create.Error
		}
		if create.RowsAffected == 0 {
			if err := tx.Where("idempotency_key = ?", key).First(&existingTask).Error; err != nil {
				return err
			}
			resultTask = &existingTask
			return nil
		}
		item := entity.TaskItem{TaskID: task.ID, Status: entity.TaskItemStatusPending, InputData: string(input)}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		resultTask = &task
		created = true
		return nil
	})
	return resultShare, resultTask, created, err
}
