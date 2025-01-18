package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"golang.org/x/exp/slices"
	"modernc.org/sqlite"
)

func init() {
	sqlite.MustRegisterScalarFunction("json_array_contains", 2, jsonArrayContains)
}

// jsonArrayContains is an SQLite function that checks if a JSON array
// in the database contains a given value
func jsonArrayContains(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	value, ok := args[0].(string)
	if !ok {
		return nil, errors.New("both arguments to json_array_contains must be strings")
	}

	item, ok := args[1].(string)
	if !ok {
		return nil, errors.New("both arguments to json_array_contains must be strings")
	}

	var array []string
	err := json.Unmarshal([]byte(value), &array)
	if err != nil {
		return nil, err
	}

	return slices.Contains(array, item), nil
}
