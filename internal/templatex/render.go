package templatex

import (
	"fmt"
	"regexp"
	"strings"
)

var reVar = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// Render replaces {{path}} placeholders using flat map keys (e.g. "ticker", "agent:uuid:output").
func Render(tpl string, vars map[string]string) string {
	return reVar.ReplaceAllStringFunc(tpl, func(s string) string {
		m := reVar.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		key := strings.TrimSpace(m[1])
		if v, ok := vars[key]; ok {
			return v
		}
		// support dotted access like datasource.result handled upstream as flat key
		return ""
	})
}

// MergeStringMap merges maps (b overrides a).
func MergeStringMap(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// Expression is a very small safe evaluator: only supports "{{var}}" +/- number suffix like doc example simplified.
func EvalSimple(expr string, vars map[string]string) (string, error) {
	expr = strings.TrimSpace(expr)
	// fixed string quoted
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) && len(expr) >= 2 {
		return strings.Trim(expr, `"`), nil
	}
	return Render(expr, vars), nil
}

// BuildPromptVarMap builds variables for agent prompt from globals + special keys.
func BuildPromptVarMap(global map[string]any, extras map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range global {
		out[k] = fmt.Sprint(v)
	}
	for k, v := range extras {
		out[k] = v
	}
	return out
}
