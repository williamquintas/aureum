package model

import (
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type Date struct {
	Time time.Time
}

func (d *Date) UnmarshalGQL(v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("Date must be a string")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("Date must be in YYYY-MM-DD format")
	}
	d.Time = t
	return nil
}

func (d Date) MarshalGQL(w io.Writer) {
	io.WriteString(w, d.Time.Format(`"2006-01-02"`))
}

func MarshalDate(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, t.Format(`"2006-01-02"`))
	})
}

func UnmarshalDate(v interface{}) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("Date must be a string")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("Date must be in YYYY-MM-DD format")
	}
	return t, nil
}
