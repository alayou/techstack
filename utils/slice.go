package utils

import (
	"strconv"

	"github.com/samber/lo"
)

// SliceString2Int64 字符串切片转成int64切片
func SliceString2Int64(arrays []string) []int64 {
	return lo.Map(arrays, func(item string, index int) int64 {
		v, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			return 0
		}
		return v
	})
}

// RemoveSliceValue 删除切片中指定值的元素
func RemoveSliceValue(s []int64, value int64) []int64 {
	index := 0
	for _, v := range s {
		if v != value {
			s[index] = v
			index++
		}
	}
	return s[:index]
}

func Repeat(s string, count int) []string {
	data := make([]string, count)
	for i := range data {
		data[i] = s
	}
	return data
}

// SliceContains 小集合是否被大集合包含
func SliceContains(min, max []string) bool {
	for _, k := range min {
		if In(k, max) {
			return true
		}
	}
	return false
}
