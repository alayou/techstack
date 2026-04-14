package bind

type QueryPage struct {
	Page           int   `form:"page,default=1" validate:"omitempty,gte=1"`
	Size           int   `form:"size,default=20" validate:"omitempty,gte=1,lte=1000"`
	CreatedAtStart int64 `json:"created_at_start" form:"created_at_start"`
	CreatedAtEnd   int64 `json:"created_at_end" form:"created_at_end"`

	Order string `json:"order"` // asc/desc
	Sort  string `json:"sort"`  // 排序的字段
}
