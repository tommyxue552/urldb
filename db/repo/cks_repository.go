package repo

import (
	"fmt"

	"github.com/ctwj/urldb/db/credential"
	"github.com/ctwj/urldb/db/entity"
	"gorm.io/gorm"
)

type CredentialAuditContext struct{ Actor, SourceIP, RequestID string }

type CksRepository interface {
	BaseRepository[entity.Cks]
	FindByPanID(uint) ([]entity.Cks, error)
	FindByIds([]uint) ([]*entity.Cks, error)
	FindByIsValid(bool) ([]entity.Cks, error)
	FindByPanIDAndCk(uint, string) (*entity.Cks, error)
	UpdateSpace(uint, int64, int64) error
	DeleteByPanID(uint) error
	UpdateWithAllFields(*entity.Cks) error
	WithAuditContext(CredentialAuditContext) CksRepository
}

// CksRepositoryImpl is the sole persistence boundary for provider secrets.
type CksRepositoryImpl struct {
	BaseRepositoryImpl[entity.Cks]
	protector *credential.Protector
	audit     CredentialAuditContext
}

func NewCksRepository(db *gorm.DB) CksRepository {
	p, err := credential.LoadFromEnvironment()
	if err != nil {
		panic(fmt.Sprintf("credential protector was not initialized: %v", err))
	}
	return &CksRepositoryImpl{BaseRepositoryImpl: BaseRepositoryImpl[entity.Cks]{db: db}, protector: p}
}
func (r *CksRepositoryImpl) WithAuditContext(c CredentialAuditContext) CksRepository {
	clone := *r
	clone.audit = c
	return &clone
}

func (r *CksRepositoryImpl) Create(a *entity.Cks) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		ck, extra := a.Ck, a.Extra
		a.Ck, a.Extra, a.CkFingerprint = "", "", ""
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		if err := r.encrypt(tx, a, ck, extra); err != nil {
			return err
		}
		a.Ck, a.Extra = ck, extra
		return r.auditEvent(tx, a, "create", "success")
	})
}
func (r *CksRepositoryImpl) FindByPanID(id uint) ([]entity.Cks, error) {
	var v []entity.Cks
	if err := r.db.Where("pan_id = ?", id).Find(&v).Error; err != nil {
		return nil, err
	}
	return v, r.decryptAll(v)
}
func (r *CksRepositoryImpl) FindByIsValid(valid bool) ([]entity.Cks, error) {
	var v []entity.Cks
	if err := r.db.Where("is_valid = ?", valid).Find(&v).Error; err != nil {
		return nil, err
	}
	return v, r.decryptAll(v)
}
func (r *CksRepositoryImpl) FindAll() ([]entity.Cks, error) {
	var v []entity.Cks
	if err := r.db.Preload("Pan").Find(&v).Error; err != nil {
		return nil, err
	}
	return v, r.decryptAll(v)
}
func (r *CksRepositoryImpl) FindByID(id uint) (*entity.Cks, error) {
	var v entity.Cks
	if err := r.db.Preload("Pan").First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, r.decrypt(&v)
}
func (r *CksRepositoryImpl) FindByIds(ids []uint) ([]*entity.Cks, error) {
	var v []*entity.Cks
	if err := r.db.Preload("Pan").Where("id IN ?", ids).Find(&v).Error; err != nil {
		return nil, err
	}
	for _, a := range v {
		if err := r.decrypt(a); err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (r *CksRepositoryImpl) FindByPanIDAndCk(id uint, ck string) (*entity.Cks, error) {
	var v entity.Cks
	if err := r.db.Preload("Pan").Where("pan_id = ? AND ck_fingerprint = ?", id, r.protector.Fingerprint(ck)).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, r.decrypt(&v)
}
func (r *CksRepositoryImpl) FindWithPagination(page, limit int) ([]entity.Cks, int64, error) {
	var accounts []entity.Cks
	var total int64
	if err := r.db.Model(&entity.Cks{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.Offset((page - 1) * limit).Limit(limit).Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	if err := r.decryptAll(accounts); err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

func (r *CksRepositoryImpl) Update(a *entity.Cks) error { return r.UpdateWithAllFields(a) }
func (r *CksRepositoryImpl) UpdateWithAllFields(a *entity.Cks) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		ck, extra := a.Ck, a.Extra
		if err := r.encrypt(tx, a, ck, extra); err != nil {
			return err
		}
		if err := tx.Omit("Pan").Save(a).Error; err != nil {
			return err
		}
		a.Ck, a.Extra = ck, extra
		return r.auditEvent(tx, a, "update", "success")
	})
}
func (r *CksRepositoryImpl) UpdateSpace(id uint, space, left int64) error {
	return r.db.Model(&entity.Cks{}).Where("id = ?", id).Updates(map[string]interface{}{"space": space, "left_space": left}).Error
}
func (r *CksRepositoryImpl) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var a entity.Cks
		if err := tx.First(&a, id).Error; err != nil {
			return err
		}
		if err := r.auditEvent(tx, &a, "delete", "success"); err != nil {
			return err
		}
		return tx.Delete(&entity.Cks{}, id).Error
	})
}
func (r *CksRepositoryImpl) DeleteByPanID(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var v []entity.Cks
		if err := tx.Where("pan_id = ?", id).Find(&v).Error; err != nil {
			return err
		}
		for i := range v {
			if err := r.auditEvent(tx, &v[i], "delete", "success"); err != nil {
				return err
			}
		}
		return tx.Where("pan_id = ?", id).Delete(&entity.Cks{}).Error
	})
}

func (r *CksRepositoryImpl) encrypt(tx *gorm.DB, a *entity.Cks, ck, extra string) error {
	eck, err := r.protector.Encrypt(ck, a.ID, "ck")
	if err != nil {
		return err
	}
	eextra, err := r.protector.Encrypt(extra, a.ID, "extra")
	if err != nil {
		return err
	}
	a.Ck, a.Extra, a.CkFingerprint = eck, eextra, r.protector.Fingerprint(ck)
	return tx.Model(&entity.Cks{}).Where("id = ?", a.ID).Updates(map[string]interface{}{"ck": eck, "extra": eextra, "ck_fingerprint": a.CkFingerprint}).Error
}
func (r *CksRepositoryImpl) decrypt(a *entity.Cks) error {
	ck, err := r.protector.Decrypt(a.Ck, a.ID, "ck")
	if err != nil {
		r.decryptFailure(a)
		return err
	}
	extra, err := r.protector.Decrypt(a.Extra, a.ID, "extra")
	if err != nil {
		r.decryptFailure(a)
		return err
	}
	a.Ck, a.Extra = ck, extra
	return nil
}
func (r *CksRepositoryImpl) decryptAll(v []entity.Cks) error {
	for i := range v {
		if err := r.decrypt(&v[i]); err != nil {
			return err
		}
	}
	return nil
}
func (r *CksRepositoryImpl) decryptFailure(a *entity.Cks) {
	_ = r.auditEvent(r.db, a, "decrypt", "failure")
}
func (r *CksRepositoryImpl) auditEvent(tx *gorm.DB, a *entity.Cks, action, outcome string) error {
	actor := r.audit.Actor
	if actor == "" {
		actor = "system"
	}
	provider := a.ServiceType
	if provider == "" {
		provider = a.Pan.Name
	}
	return tx.Create(&entity.CredentialAudit{AccountID: a.ID, Provider: provider, Action: action, Outcome: outcome, Actor: actor, SourceIP: r.audit.SourceIP, RequestID: r.audit.RequestID, KeyVersion: "v1"}).Error
}
