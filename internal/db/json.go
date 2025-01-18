// ALR - Any Linux Repository
// Copyright (C) 2025 Евгений Храмов
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
