package db

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetMiniUserProfile(userID string) (*CustomerDO, error) {
	var row CustomerDO
	if err := Get().First(&row, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetMiniUserByOpenID(openID string) (*CustomerDO, error) {
	var row CustomerDO
	if err := Get().First(&row, "openid = ?", openID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetMiniUserByDouyinOpenID(openID string) (*CustomerDO, error) {
	var row CustomerDO
	if err := Get().First(&row, "douyin_openid = ?", openID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetMiniUserByAlipayUserID(userID string) (*CustomerDO, error) {
	var row CustomerDO
	if err := Get().First(&row, "alipay_user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetMiniUserByPhone(phone string) (*CustomerDO, error) {
	var row CustomerDO
	if err := Get().First(&row, "phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func UpsertMiniUserProfile(row *CustomerDO) error {
	if row.ID == "" {
		return fmt.Errorf("user id required")
	}
	return Get().Save(row).Error
}

// CreateMiniUserWithoutPhone 用于已完成第三方身份授权、但尚未主动授权手机号
// 的用户。手机号列必须允许 NULL，避免空字符串触发唯一索引冲突。
func CreateMiniUserWithoutPhone(row *CustomerDO) error {
	if row.ID == "" {
		return fmt.Errorf("user id required")
	}
	return Get().Omit("phone").Create(row).Error
}

func GetAppConfig(key string) (*AppConfigDO, error) {
	var row AppConfigDO
	result := Get().Where("`key` = ?", key).Limit(1).Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func MustAppConfigJSON(key, fallback string) string {
	row, err := GetAppConfig(key)
	if err != nil || row.Value == "" {
		return fallback
	}
	return row.Value
}

func UpsertAppConfig(key, value, note string) error {
	now := time.Now()
	row := &AppConfigDO{Key: key, Value: value, Note: note, CreatedAt: now, UpdatedAt: now}
	// Save 会将零值 CreatedAt 写入已有记录；使用冲突更新仅更新可变字段，
	// 保留原有创建时间，兼容 MySQL 严格日期校验。
	return Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"value":      value,
			"note":       note,
			"updated_at": now,
		}),
	}).Create(row).Error
}

func MarshalStringSlice(values []string) string {
	if values == nil {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func UnmarshalStringSlice(value string) []string {
	if value == "" {
		return []string{}
	}
	var out []string
	if json.Unmarshal([]byte(value), &out) != nil {
		return []string{}
	}
	return out
}
