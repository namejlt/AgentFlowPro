package sanitize

import (
	"encoding/json"
	"strings"
)

var sensitiveKeys = []string{
	"password", "api_key", "apikey", "token", "authorization", "secret",
}

// RedactJSON walks a JSON object and masks sensitive string values.
func RedactJSON(b []byte) []byte {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return b
	}
	redactValue(v)
	out, _ := json.Marshal(v)
	return out
}

func redactValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			lk := strings.ToLower(k)
			if containsSensitive(lk) {
				if _, ok := val.(string); ok {
					t[k] = "***"
				}
			} else {
				redactValue(val)
			}
		}
	case []any:
		for _, it := range t {
			redactValue(it)
		}
	}
}

func containsSensitive(k string) bool {
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
