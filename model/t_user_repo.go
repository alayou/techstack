package model

type UserRepoStar struct {
	ID           ID    `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`
	UserID       int64 `json:"user_id,string,omitempty" gorm:"index"`
	PublicRepoID int64 `json:"public_repo_id,string,omitempty" gorm:"index"`
	CreatedAt    int64 `json:"created_at" gorm:"autoCreateTime;not null"` // 创建时间,用于排序
}

func (UserRepoStar) TableName() string {
	return UserRepoStarTableName
}
