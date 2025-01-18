package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSON represents a JSON value in the database
type JSON[T any] struct {
	Val T
}

// NewJSON creates a new database JSON value
func NewJSON[T any](v T) JSON[T] {
	return JSON[T]{Val: v}
}

func (s *JSON[T]) Scan(val any) error {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case string:
		err := json.Unmarshal([]byte(val), &s.Val)
		if err != nil {
			return err
		}
	case sql.NullString:
		if val.Valid {
			err := json.Unmarshal([]byte(val.String), &s.Val)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("sqlite json types must be strings")
	}

	return nil
}

func (s JSON[T]) Value() (driver.Value, error) {
	data, err := json.Marshal(s.Val)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (s JSON[T]) MarshalYAML() (any, error) {
	return s.Val, nil
}

func (s JSON[T]) String() string {
	return fmt.Sprint(s.Val)
}

func (s JSON[T]) GoString() string {
	return fmt.Sprintf("%#v", s.Val)
}
