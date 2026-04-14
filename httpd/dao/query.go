package dao

import "fmt"

func OffsetAndLimit(pageNo, pageSize int64) (offset, limit int) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset = int((pageNo - 1) * pageSize)
	limit = int(pageSize)
	return
}

func Like(s string) string {
	return "%" + s + "%"
}

type Where struct {
	start string
	end   string
	Key   string
	Value interface{}
	Opt   string
	OrAnd string
}

func (q *Where) String() string {
	if q.Key == "" {
		return ""
	}
	return fmt.Sprintf("%s %s ?", q.Key, q.Opt)
}

type Query struct {
	Offset int
	Limit  int
	Where  []*Where
}

const (
	WhereEq   = "="
	WhereNe   = "!="
	WhereGt   = ">"
	WhereGte  = ">="
	WhereLt   = "<"
	WhereLte  = "<="
	WhereLike = "like"
	WhereIn   = "in"
	WhereNin  = "not in"
	WhereOr   = "or"
	WhereAnd  = "and"
)

func NewQuery() *Query {
	q := &Query{
		Where: make([]*Where, 0),
	}
	return q
}

func NewWhere(k, opt string, v interface{}) *Where {
	q := &Where{
		Key:   k,
		Value: v,
		Opt:   opt,
		OrAnd: WhereAnd,
	}
	return q
}

func (q *Query) WithPage(pageNo, pageSize int) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (pageNo - 1) * pageSize
	q.Offset = int(offset)
	q.Limit = int(pageSize)
}

func (q *Query) WithEq(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereEq, v))
}

func (q *Query) WithNe(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereNe, v))
}

func (q *Query) WithGt(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereGt, v))
}

func (q *Query) WithGte(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereGte, v))
}

func (q *Query) WithLt(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereLt, v))
}

func (q *Query) WithLte(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereLte, v))
}

func (q *Query) WithLike(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereLike, fmt.Sprintf("%%%s%%", v)))
}

func (q *Query) WithOrLike(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereLike, fmt.Sprintf("%%%s%%", v)))
}

func (q *Query) WithIn(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereIn, fmt.Sprintf("(%s)", v)))
}

func (q *Query) WithNin(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereNin, fmt.Sprintf("(%s)", v)))
}

func (q *Query) WithOrNin(k string, v interface{}) {
	q.Where = append(q.Where, NewWhere(k, WhereNin, fmt.Sprintf("(%s)", v)))
}

func (q *Query) WithOr(wheres ...*Where) {
	q.With(wheres, WhereOr)
}

func (q *Query) With(wheres []*Where, opt string) {
	for i, where := range wheres {
		isStart := i == 0
		isEnd := len(wheres)-1 == i
		where.OrAnd = opt
		if isStart {
			where.start = "("
			where.OrAnd = WhereAnd
		}
		if isEnd {
			where.end = ")"
		}
		q.Where = append(q.Where, where)
	}
}

func (q *Query) WithAnd(wheres ...*Where) {
	q.With(wheres, WhereAnd)
}

func (q *Query) Params() []interface{} {
	var params = make([]interface{}, len(q.Where))
	for i, w := range q.Where {
		params[i] = w.Value
	}
	return params
}

func (q *Query) Query() string {
	var s string
	var skipEnd bool
	for i, where := range q.Where {
		w := where.String()
		if i == 0 {
			s = w
			if where.start != "" {
				skipEnd = true
			}
			continue
		}
		if skipEnd && where.end != "" {
			skipEnd = false
			s = s + " " + where.OrAnd + " " + where.start + w
			continue
		}
		s = s + " " + where.OrAnd + " " + where.start + w + where.end
	}
	return s
}
