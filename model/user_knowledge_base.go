package model

import (
	"errors"

	"gorm.io/gorm"
)

// UserKnowledgeBase records the third-party knowledge base organization that a
// new-api user has been provisioned with. Since new-api has no organization
// concept, each user who enables the knowledge base feature gets their own
// organization on the third-party service and is registered as its admin.
type UserKnowledgeBase struct {
	Id            int    `json:"id"`
	UserId        int    `json:"user_id" gorm:"uniqueIndex;column:user_id"`
	OrgId         string `json:"org_id" gorm:"type:varchar(128);column:org_id;index"`
	OrgName       string `json:"org_name" gorm:"type:varchar(255);column:org_name"`
	KbUserId      string `json:"kb_user_id" gorm:"type:varchar(128);column:kb_user_id"` // user id on the knowledge base side
	BackendUserId string `json:"backend_user_id" gorm:"type:varchar(255);column:backend_user_id"`
	Status        string `json:"status" gorm:"type:varchar(32);column:status;default:'active'"`
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt     int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (UserKnowledgeBase) TableName() string {
	return "user_knowledge_bases"
}

// GetUserKnowledgeBase returns the knowledge base binding for a user, or nil if
// the user has not enabled the knowledge base feature.
func GetUserKnowledgeBase(userId int) (*UserKnowledgeBase, error) {
	var kb UserKnowledgeBase
	err := DB.Where("user_id = ?", userId).First(&kb).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &kb, nil
}

// CreateUserKnowledgeBase persists a new knowledge base binding.
func CreateUserKnowledgeBase(kb *UserKnowledgeBase) error {
	return DB.Create(kb).Error
}

// DeleteUserKnowledgeBase removes a user's knowledge base binding.
func DeleteUserKnowledgeBase(userId int) error {
	return DB.Where("user_id = ?", userId).Delete(&UserKnowledgeBase{}).Error
}

// GetAllUserKnowledgeBases returns all knowledge base bindings (admin use).
func GetAllUserKnowledgeBases() ([]UserKnowledgeBase, error) {
	var list []UserKnowledgeBase
	err := DB.Find(&list).Error
	return list, err
}
