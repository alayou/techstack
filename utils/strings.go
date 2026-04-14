package utils

import (
	"net"
	"strings"
)

func In(a string, s []string) bool {
	for _, b := range s {
		if a == b {
			return true
		}
	}
	return false
}

func InIP(ipStr string, s ...string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ipOrCidr := range s {
		if ipStr == ipOrCidr {
			return true
		}
		_, ipNet, err := net.ParseCIDR(ipOrCidr)
		if err != nil {
			continue
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// Comparestring 比较更新数组，返回，删除列表，创建列表，和更新列表
func CompareString(newList []string, oldList []string) (map[string]struct{}, map[string]struct{}, map[string]struct{}) {
	var deleteMap = make(map[string]struct{}, 0)
	var createdMap = make(map[string]struct{}, 0)
	var updateMap = make(map[string]struct{}, 0)
	for _, news := range newList {
		createdMap[news] = struct{}{} // 假定所有新项目为创建
		for _, old := range oldList {
			if old == news { // 如果已存在则不需要创建了
				delete(createdMap, old) // 如果已经不存在则需要创建
				updateMap[old] = struct{}{}
				break
			}
		}
	}
	for _, old := range oldList {
		deleteMap[old] = struct{}{} // 假定所有旧项目为删除
		for _, news := range newList {
			if old == news {
				delete(deleteMap, old) // 如果已经存在则不删除
				//updateMap[old] = struct{}{}  放入任意循环中
				break
			}
		}
	}
	return deleteMap, createdMap, updateMap
}

// IsParentChildPath a b路径是否是父子目录
func IsParentChildPath(a, b string) bool {
	if len(a) < len(b) {
		return strings.HasPrefix(b+"/", a+"/")
	}
	return strings.HasPrefix(a+"/", b+"/")
}

var TRUE_STRS = []string{"1", "true", "on", "yes"}

func ToBool(str string) bool {
	val := strings.ToLower(strings.TrimSpace(str))
	for _, v := range TRUE_STRS {
		if v == val {
			return true
		}
	}
	return false
}
