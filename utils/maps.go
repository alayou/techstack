package utils

func GetMapKeys(m map[string]interface{}) []string {
	// 数组默认长度为map长度,后面append时,不需要重新申请内存和拷贝,效率很高
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// CheckKeyInMap  检查key是否在map中
func CheckKeyInMap(key string, m map[string]interface{}) bool {
	for k := range m {
		if k == key {
			return true
		}
	}
	return false
}
