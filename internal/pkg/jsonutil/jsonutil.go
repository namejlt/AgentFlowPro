package jsonutil

import (
	"encoding/json"
)

func MustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func Unmarshal[T any](b []byte) (T, error) {
	var z T
	err := json.Unmarshal(b, &z)
	return z, err
}

func UnmarshalMap(b []byte) (map[string]any, error) {
	var m map[string]any
	if len(b) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func UnmarshalArray(b []byte) ([]any, error) {
	var a []any
	if len(b) == 0 {
		return []any{}, nil
	}
	err := json.Unmarshal(b, &a)
	return a, err
}
