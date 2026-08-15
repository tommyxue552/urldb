package db

import (
	"strings"

	"github.com/ctwj/urldb/db/credential"
	"github.com/ctwj/urldb/db/entity"
	"gorm.io/gorm"
)

func migrateLegacyCredentials(database *gorm.DB) error {
	protector, err := credential.LoadFromEnvironment()
	if err != nil {
		return err
	}
	var accounts []entity.Cks
	if err := database.Find(&accounts).Error; err != nil {
		return err
	}
	return database.Transaction(func(tx *gorm.DB) error {
		for _, account := range accounts {
			updates := map[string]interface{}{}
			if account.Ck != "" && !strings.HasPrefix(account.Ck, "enc:v1:") {
				encrypted, err := protector.Encrypt(account.Ck, account.ID, "ck")
				if err != nil {
					return err
				}
				updates["ck"] = encrypted
			}
			if account.Extra != "" && !strings.HasPrefix(account.Extra, "enc:v1:") {
				encrypted, err := protector.Encrypt(account.Extra, account.ID, "extra")
				if err != nil {
					return err
				}
				updates["extra"] = encrypted
			}
			if account.Ck != "" && account.CkFingerprint == "" {
				plainCK := account.Ck
				if strings.HasPrefix(plainCK, "enc:v1:") {
					plainCK, err = protector.Decrypt(plainCK, account.ID, "ck")
					if err != nil {
						return err
					}
				}
				updates["ck_fingerprint"] = protector.Fingerprint(plainCK)
			}
			if len(updates) == 0 {
				continue
			}
			if err := tx.Model(&entity.Cks{}).Where("id = ?", account.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Create(&entity.CredentialAudit{AccountID: account.ID, Provider: account.ServiceType, Action: "migration", Outcome: "success", Actor: "system", KeyVersion: "v1"}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
