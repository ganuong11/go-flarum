package model

import (
	"fmt"

	"gorm.io/gorm"
)

type BlogMeta struct {
	gorm.Model
	ID      uint64 `gorm:"primaryKey"`
	TopicID uint64 `gorm:"column:topic_id;index"` // Associated topic ID

	Summary         string `json:"summary"`           // Summary
	FeaturedImage   string `json:"featured_image"`    // Featured image
	IsFeatured      bool   `json:"is_featured"`       // Whether it is a featured article
	IsPendingReview bool   `json:"is_pending_review"` // Whether it is pending review
	IsSized         bool   `json:"is_sized"`          // Whether it has been resized

	// TopicData Topic `gorm:"foreignKey:ID;references:TopicID"` // Associated topic
}

func (blogMeta *BlogMeta) CreateFlarumBlogMeta(gormDB *gorm.DB) (bool, error) {
	// Create or update blog metadata
	if err := gormDB.Create(blogMeta).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (blogMeta *BlogMeta) CreateOrUpdate(gormDB *gorm.DB) error {
	// Create or update blog metadata
	if blogMeta.ID == 0 {
		return gormDB.Create(blogMeta).Error
	}
	return gormDB.Save(blogMeta).Error
}

func (blogMeta *BlogMeta) GetFormatedString() string {
	return fmt.Sprintf(
		"[BlogMeta] ID: %d, TopicID: %d, Summary: %s",
		blogMeta.ID,
		blogMeta.TopicID,
		blogMeta.Summary,
	)
}

func SQLGetAllBlogMeta(gormDB *gorm.DB) (metas []BlogMeta, err error) {
	// Get all blog metadata
	// err = gormDB.Preload("TopicData").Find(&metas).Error
	err = gormDB.Find(&metas).Error
	if err != nil {
		return nil, err
	}
	return metas, nil
}

func SQLGetBlogMetaByTopicID(gormDB *gorm.DB, topicID uint64) (meta BlogMeta, err error) {
	// Get blog metadata for specified topic
	err = gormDB.First(&meta, "topic_id = ?", topicID).Error
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func SQLGetBlogMetaByID(gormDB *gorm.DB, id uint64) (meta BlogMeta, err error) {
	// Get blog metadata for specified ID
	err = gormDB.First(&meta, "id = ?", id).Error
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func SQLSaveBlogMeta(gormDB *gorm.DB, meta *BlogMeta) error {
	// Save or update blog metadata
	if meta.ID == 0 {
		return gormDB.Create(meta).Error
	}
	return gormDB.Save(meta).Error
}
