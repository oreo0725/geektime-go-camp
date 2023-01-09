package homework_delete

import (
	"reflect"
)

type Deleter[T any] struct {
	whereBuilder
	table string
	where []Predicate
}

func (d *Deleter[T]) Build() (*Query, error) {
	d.sb.WriteString(`DELETE FROM `)
	if d.table == "" {
		var t T
		d.sb.WriteByte('`')
		d.sb.WriteString(reflect.TypeOf(t).Name())
		d.sb.WriteByte('`')
	} else {
		d.sb.WriteString(d.table)
	}

	if len(d.where) > 0 {
		d.args = make([]any, 0, 4)
		d.sb.WriteString(` WHERE `)
		p := d.where[0]
		for i := 1; i < len(d.where); i++ {
			p = p.And(d.where[i])
		}
		if err := d.buildExpression(p); err != nil {
			return nil, err
		}
	}
	d.sb.WriteByte(';')
	return &Query{
		SQL:  d.sb.String(),
		Args: d.args,
	}, nil
}

// From accepts model definition
func (d *Deleter[T]) From(table string) *Deleter[T] {
	d.table = table
	return d
}

// Where accepts predicates
func (d *Deleter[T]) Where(predicates ...Predicate) *Deleter[T] {
	d.where = predicates
	return d
}
