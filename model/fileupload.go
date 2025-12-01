package model

import (
	"time"

	"gorm.io/gorm"
)

type UserFiles struct {
	gorm.Model
	ID         uint64 `gorm:"primaryKey"`
	UUID       string `json:"uuid" gorm:"index:idx_uuid,unique"` // Unique identifier for the file
	UserID     uint64 `json:"user_id" gorm:"index:idx_user_id"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"file_path"`
	FileSize   int64  `json:"file_size"`
	FileType   string `json:"file_type"`
	Visibility string `json:"visibility"` // Visibility, public or private
	// Deleted    bool   `json:"deleted"`    // Whether it has been deleted
	// Other possible fields

	// UploadTime int64 `json:"upload_time"` // Upload time
}

func SQLGetUserFiles(gormDB *gorm.DB, userID uint64) (files []UserFiles, err error) {
	// Get all files for the specified user
	err = gormDB.Where("user_id = ?", userID).Find(&files).Error
	return
}

func getCurDate() string {
	// Get string representation of current date, compatible with MySQL and PostgreSQL
	return time.Now().Format("2006-01-02")
}

// Get the number of files uploaded by the user today
func SQLGetUserDailyUploads(gormDB *gorm.DB, userID uint64) (count int64, err error) {
	// Get the number of files uploaded by the user today
	err = gormDB.Model(&UserFiles{}).
		Where("user_id = ? AND DATE(created_at) = ?", userID, getCurDate()).
		Count(&count).Error
	return
}

func SQLSaveUserFile(gormDB *gorm.DB, file *UserFiles) error {
	// Save user file record
	return gormDB.Create(file).Error
}
