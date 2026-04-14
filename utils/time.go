package utils

import (
	"time"
)

func DiffDays(t1, t2 int64) int {
	// 计算时间差
	diff := time.Unix(t2, 0).Sub(time.Unix(t1, 0))
	// 将时间差转换为天数
	days := int(diff.Hours() / 24)
	return days
}

// GetTimeAsMsSinceEpoch returns unix timestamp as milliseconds from a time struct
func GetTimeAsMsSinceEpoch(t time.Time) int64 {
	return t.UnixMilli()
}
