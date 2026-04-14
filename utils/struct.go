package utils

import "reflect"

// StructAssign binding 要修改的结构体  value有数据的结构体 args是否更新空值,默认更新空值
func StructAssign(binding interface{}, value interface{}, args ...bool) {
	argLen := len(args)
	// 获取reflect.Type类型
	bVal := reflect.ValueOf(binding).Elem()
	vVal := reflect.ValueOf(value).Elem()
	vTypeOfT := vVal.Type()
	for i := 0; i < vVal.NumField(); i++ {
		// 在要修改的结构体中查询有数据结构体中相同属性的字段，有则修改其值
		name := vTypeOfT.Field(i).Name
		if ok := bVal.FieldByName(name).IsValid(); ok {
			if argLen > 0 && !args[0] && vVal.Field(i).IsZero() { // 空值不更新
				continue
			}

			if bVal.FieldByName(name).Kind() != vVal.Field(i).Kind() { //类型不相同不更新
				continue
			}

			if vVal.Field(i).Kind() == reflect.Ptr { // 指针类型
				bVal.FieldByName(name).Set(reflect.ValueOf(vVal.Field(i).Elem().Interface()))
			} else {
				bVal.FieldByName(name).Set(reflect.ValueOf(vVal.Field(i).Interface()))
			}
		}
	}
}

// StructToMap 结构体转map
func StructToMap(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Tag.Get("json")
		data[name] = v.Field(i).Interface()
	}
	return data
}
