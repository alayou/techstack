package buserr

import "fmt"

type BusinessMultiError []BusinessError

func (b BusinessMultiError) Local(lang string) string {
	var msg string
	for i := len(b) - 1; i >= 0; i-- {
		v := b[i]
		if v.Detail == "" {
			v.Detail = msg
		}
		msg = v.Local(lang)
	}
	return msg
}

func (b BusinessMultiError) Error() string {
	if len(b) == 0 {
		return "MultiError"
	}
	return b[0].Error()
}

type BusinessError struct {
	Key    string         // 键
	Detail string         // 值
	Map    map[string]any // 自定义模板
}

func (b BusinessError) Error() string {
	return b.Key
}

func (b BusinessError) Local(lang string) string {
	if b.Detail != "" {
		if b.Map == nil {
			b.Map = make(map[string]any, 1)
		}
		b.Map["detail"] = b.Detail
	}
	// TODO i18n
	return fmt.Sprintf("%s:%s", b.Key, b.Detail)
}

func New(Key string) BusinessError {
	return BusinessError{
		Key: Key,
	}
}

func NewWithDetail(Key, detail string) BusinessError {
	return BusinessError{
		Key:    Key,
		Detail: detail,
	}
}

// Wrap 包装错误，错误详情可以是BusinessError
func Wrap(a BusinessError, arr ...BusinessError) BusinessMultiError {
	me := make(BusinessMultiError, len(arr)+1)
	me[0] = a
	for i, a := range arr {
		me[i+1] = a
	}
	return me
}

// WrapMulti 包装错误，错误详情可以是BusinessError
func WrapMulti(a BusinessMultiError, arr ...BusinessError) BusinessMultiError {
	for _, v := range arr {
		a = append(a, v)
	}
	return a
}

func WrapDetail(bs BusinessError, detail string) BusinessError {
	bs.Detail = detail
	return bs
}
