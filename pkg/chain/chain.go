package chain

type Stream[T any] struct {
	items []T
}

func From[T any](items []T) *Stream[T] {
	return &Stream[T]{items: items}
}

// UniqueBy 按自定义key去重
func (s *Stream[T]) UniqueBy(keyFunc func(T) any) *Stream[T] {
	seen := make(map[any]bool)
	res := make([]T, 0, len(s.items))

	for _, item := range s.items {
		key := keyFunc(item)
		// 检查key是否可哈希（必须是可比较的类型）
		if key == nil {
			continue
		}
		if _, exists := seen[key]; !exists {
			seen[key] = true
			res = append(res, item)
		}
	}
	s.items = res
	return s
}

func (s *Stream[T]) ToSlice() []T {
	return s.items
}
