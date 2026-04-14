package model

type UserApiKey struct {
	ID         ID          `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`
	UserID     int64       `json:"user_id,string,omitempty" gorm:"not null;index;"`
	Name       string      `gorm:"type:text;not null" json:"name"`
	KeyPrefix  string      `gorm:"type:text;not null;index:idx_api_keys_key_prefix" json:"key_prefix"`
	KeyHash    string      `gorm:"type:text;not null" json:"-"`
	Scopes     StringArray `gorm:"type:jsonb;not null" json:"scopes"`
	LastUsedAt int64       `gorm:"type:timestamp" json:"last_used_at"`
	RevokedAt  int64       `gorm:"type:timestamp" json:"revoked_at"`
	CreatedAt  int64       `json:"created_at" gorm:"autoCreateTime;not null"` // 创建时间
	UpdatedAt  int64       `json:"updated_at" gorm:"autoUpdateTime;not null"` // 更新时间
}

func (UserApiKey) TableName() string {
	return UserApiKeyTableName
}
