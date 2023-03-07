package orm

import (
	"fmt"
	"strings"

	"github.com/oreo0725/geektime-go-camp/orm/howework_select/internal/errs"
	"github.com/oreo0725/geektime-go-camp/orm/howework_select/model"
)

// Selector 用于构造 SELECT 语句
type Selector[T any] struct {
	db   *DB
	sb   strings.Builder
	args []any

	model *model.Model

	selects  []Selectable
	table    string
	where    []Predicate
	groupBy  []Column
	having   []Predicate
	orderBys []OrderBy
	offset   int
	limit    int
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	s.selects = cols
	return s
}

// From 指定表名，如果是空字符串，那么将会使用默认表名
func (s *Selector[T]) From(tbl string) *Selector[T] {
	s.table = tbl
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	m, err := s.db.r.Register(new(T))
	if err != nil {
		return nil, err
	}
	s.model = m
	// select columns
	s.sb.WriteString(`SELECT `)
	if len(s.selects) == 0 {
		s.sb.WriteByte('*')
	} else {
		for i, col := range s.selects {
			if i > 0 {
				s.sb.WriteByte(',')
			}

			switch typ := col.(type) {
			case Column:
				fd, ok := s.model.FieldMap[typ.name]
				if !ok {
					return nil, errs.NewErrUnknownField(typ.name)
				}
				s.sb.WriteByte('`')
				s.sb.WriteString(fd.ColName)
				s.sb.WriteByte('`')
				if typ.alias != "" {
					s.sb.WriteString(" AS `")
					s.sb.WriteString(typ.alias)
					s.sb.WriteByte('`')
				}
			case Aggregate:
				fd, ok := s.model.FieldMap[typ.arg]
				if !ok {
					return nil, errs.NewErrUnknownField(typ.arg)
				}
				s.sb.WriteString(typ.fn)
				s.sb.WriteString("(`")
				s.sb.WriteString(fd.ColName)
				s.sb.WriteString("`)")
				if typ.alias != "" {
					s.sb.WriteString(" AS `")
					s.sb.WriteString(typ.alias)
					s.sb.WriteByte('`')
				}
			case RawExpr:
				s.sb.WriteString(typ.raw)
			}

		}
	}

	s.sb.WriteString(` FROM `)
	if s.table == "" {
		s.sb.WriteByte('`')
		s.sb.WriteString(s.model.TableName)
		s.sb.WriteByte('`')
	} else {
		s.sb.WriteString(s.table)
	}

	if len(s.where) > 0 {
		s.args = make([]any, 0, 4)
		s.sb.WriteString(` WHERE `)
		p := s.where[0]
		for i := 1; i < len(s.where); i++ {
			p = p.And(s.where[i])
		}
		if err := s.buildWhereExpr(p); err != nil {
			return nil, err
		}
	}

	// group by

	// order by
	if len(s.orderBys) > 0 {
		s.sb.WriteString(` ORDER BY `)
		for i, ob := range s.orderBys {
			fd, ok := s.model.FieldMap[ob.col]
			if !ok {
				return nil, errs.NewErrUnknownField(ob.col)
			}
			if i > 0 {
				s.sb.WriteByte(',')
			}
			s.sb.WriteByte('`')
			s.sb.WriteString(fd.ColName)
			s.sb.WriteString("` ")
			s.sb.WriteString(ob.order)
		}
	}

	// limit
	if s.limit > 0 {
		s.sb.WriteString(" LIMIT ?")
		s.args = append(s.args, s.limit)
	}
	// offset
	if s.offset > 0 {
		s.sb.WriteString(" OFFSET ?")
		s.args = append(s.args, s.offset)
	}

	s.sb.WriteByte(';')
	return &Query{
		SQL:  s.sb.String(),
		Args: s.args,
	}, nil
}

// Where 用于构造 WHERE 查询条件。如果 ps 长度为 0，那么不会构造 WHERE 部分
func (s *Selector[T]) Where(ps ...Predicate) *Selector[T] {
	s.where = ps
	return s
}

// GroupBy 设置 group by 子句
func (s *Selector[T]) GroupBy(cols ...Column) *Selector[T] {
	s.groupBy = cols
	return s
}

func (s *Selector[T]) Having(ps ...Predicate) *Selector[T] {
	s.having = ps
	return s
}

func (s *Selector[T]) Offset(offset int) *Selector[T] {
	s.offset = offset
	return s
}

func (s *Selector[T]) Limit(limit int) *Selector[T] {
	s.limit = limit
	return s
}

func (s *Selector[T]) OrderBy(orderBys ...OrderBy) *Selector[T] {
	s.orderBys = orderBys
	return s
}

func NewSelector[T any](db *DB) *Selector[T] {
	return &Selector[T]{
		db: db,
	}
}

type Selectable interface {
	selectable()
}

type OrderBy struct {
	col   string
	order string
}

func Asc(col string) OrderBy {
	return OrderBy{
		col: col, order: "ASC",
	}
}

func Desc(col string) OrderBy {
	return OrderBy{
		col: col, order: "DESC",
	}
}

func (s *Selector[T]) buildWhereExpr(e Expression) error {
	if e == nil {
		return nil
	}
	switch exp := e.(type) {
	case Column:
		fd, ok := s.model.FieldMap[exp.name]
		if !ok {
			return errs.NewErrUnknownField(exp.name)
		}
		s.sb.WriteByte('`')
		s.sb.WriteString(fd.ColName)
		s.sb.WriteByte('`')
	case value:
		s.sb.WriteByte('?')
		s.args = append(s.args, exp.val)
	case Predicate:
		_, lp := exp.left.(Predicate)
		if lp {
			s.sb.WriteByte('(')
		}
		if err := s.buildWhereExpr(exp.left); err != nil {
			return err
		}
		if lp {
			s.sb.WriteByte(')')
		}

		s.sb.WriteByte(' ')
		s.sb.WriteString(exp.op.String())
		s.sb.WriteByte(' ')

		_, rp := exp.right.(Predicate)
		if rp {
			s.sb.WriteByte('(')
		}
		if err := s.buildWhereExpr(exp.right); err != nil {
			return err
		}
		if rp {
			s.sb.WriteByte(')')
		}
	case RawExpr:
		s.sb.WriteString(exp.raw)
		s.args = append(s.args, exp.args...)
	default:
		return fmt.Errorf("orm: unsupported expression %v", exp)
	}
	return nil
}
