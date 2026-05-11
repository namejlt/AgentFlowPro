package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	gocache "github.com/patrickmn/go-cache"
	"github.com/tidwall/gjson"
	"github.com/gorilla/websocket"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/jsonutil"
	"github.com/namejlt/AgentFlowPro/internal/templatex"
)

var memCache = gocache.New(5*time.Minute, 10*time.Minute)

type Executor struct {
	HTTP *http.Client
}

func NewExecutor() *Executor {
	return &Executor{HTTP: &http.Client{Timeout: 0}}
}

func (e *Executor) Execute(ctx context.Context, ds *model.DataSource, in ExecuteInput, decryptHeaders func(string) (map[string]string, error), decryptAuth func(string) (map[string]any, error)) (*ExecuteResult, error) {
	cacheKey := cacheKeyFor(ds, in)
	switch ds.CachePolicy {
	case "ttl", "fixed":
		if ds.CacheTTLSeconds != nil && *ds.CacheTTLSeconds > 0 {
			if v, ok := memCache.Get(cacheKey); ok {
				if rr, ok2 := v.(*ExecuteResult); ok2 {
					cp := *rr
					cp.FromCache = true
					return &cp, nil
				}
			}
		}
	}

	var raw []byte
	var err error
	switch ds.DSType {
	case "HTTP_GET", "HTTP_POST":
		raw, err = e.execHTTP(ctx, ds, in, decryptHeaders, decryptAuth)
	case "FILE_UPLOAD":
		raw, err = e.execFile(ctx, ds)
	case "MANUAL_INPUT":
		raw, err = e.execManual(in)
	case "WEBSOCKET_STREAM":
		raw, err = e.execWebSocket(ctx, ds, in, decryptHeaders, decryptAuth)
	default:
		return nil, fmt.Errorf("unsupported datasource type: %s", ds.DSType)
	}
	if err != nil {
		return nil, err
	}

	extracted := extractFromResponse(ds, raw)

	res := &ExecuteResult{Raw: raw, Extracted: extracted, FromCache: false}
	if (ds.CachePolicy == "ttl" || ds.CachePolicy == "fixed") && ds.CacheTTLSeconds != nil && *ds.CacheTTLSeconds > 0 {
		memCache.Set(cacheKey, res, time.Duration(*ds.CacheTTLSeconds)*time.Second)
	}
	return res, nil
}

func cacheKeyFor(ds *model.DataSource, in ExecuteInput) string {
	b, _ := json.Marshal(in)
	return ds.ID.String() + ":" + string(b)
}

func (e *Executor) execHTTP(ctx context.Context, ds *model.DataSource, in ExecuteInput, decryptHeaders func(string) (map[string]string, error), decryptAuth func(string) (map[string]any, error)) ([]byte, error) {
	if ds.URLTemplate == nil {
		return nil, errors.New("missing url template")
	}
	flat := flatten(in.GlobalVars, in.Params)
	urlStr := templatex.Render(*ds.URLTemplate, flat)

	to := time.Duration(ds.TimeoutMS) * time.Millisecond
	if to <= 0 {
		to = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	headers := map[string]string{"Content-Type": "application/json"}
	if ds.ContentType != nil && *ds.ContentType != "" {
		headers["Content-Type"] = *ds.ContentType
	}
	if ds.HeadersEncrypted != nil && *ds.HeadersEncrypted != "" {
		h, err := decryptHeaders(*ds.HeadersEncrypted)
		if err != nil {
			return nil, err
		}
		for k, v := range h {
			headers[k] = v
		}
	}
	auth, _ := decryptAuthPtr(ds, decryptAuth)
	applyAuth(headers, ds.AuthType, auth)

	method := "GET"
	if ds.DSType == "HTTP_POST" {
		method = "POST"
	}
	if ds.HTTPMethod != nil && *ds.HTTPMethod != "" {
		method = strings.ToUpper(*ds.HTTPMethod)
	}

	var body io.Reader
	if method == http.MethodGet {
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		// query template already rendered in urlStr
		_ = u
	} else {
		if len(ds.BodyTemplate) > 0 {
			bt := string(ds.BodyTemplate)
			rendered := templatex.Render(bt, flat)
			body = strings.NewReader(rendered)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= ds.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(200*(1<<min(attempt, 4))) * time.Millisecond):
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := e.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("http %d: %s", resp.StatusCode, string(bytes.TrimSpace(b)))
			if resp.StatusCode >= 500 || resp.StatusCode == 429 {
				continue
			}
			return nil, lastErr
		}
		return b, nil
	}
	if lastErr == nil {
		lastErr = errors.New("http request failed")
	}
	return nil, lastErr
}

func decryptAuthPtr(ds *model.DataSource, decryptAuth func(string) (map[string]any, error)) (map[string]any, error) {
	if ds.AuthConfigEncrypted == nil || *ds.AuthConfigEncrypted == "" {
		return map[string]any{}, nil
	}
	return decryptAuth(*ds.AuthConfigEncrypted)
}

func applyAuth(h map[string]string, authType string, auth map[string]any) {
	switch authType {
	case "bearer":
		if t, ok := auth["token"].(string); ok && t != "" {
			h["Authorization"] = "Bearer " + t
		}
	case "api_key_header":
		hk, _ := auth["header"].(string)
		hv, _ := auth["value"].(string)
		if hk != "" {
			h[hk] = hv
		}
	case "custom_header":
		hk, _ := auth["header"].(string)
		hv, _ := auth["value"].(string)
		if hk != "" {
			h[hk] = hv
		}
	}
}

func (e *Executor) execFile(ctx context.Context, ds *model.DataSource) ([]byte, error) {
	ex, _ := jsonutil.UnmarshalMap(ds.ExtraConfig)
	path, _ := ex["local_path"].(string)
	if path == "" && ds.UploadedFileID != nil {
		// storage key resolved by service layer normally; here allow extra_config.storage_key
		path, _ = ex["storage_key"].(string)
	}
	if path == "" {
		return nil, errors.New("file datasource missing local_path/storage_key in extra_config")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (e *Executor) execManual(in ExecuteInput) ([]byte, error) {
	// manual: return JSON of params/global subset
	return json.Marshal(in.Params)
}

func (e *Executor) execWebSocket(ctx context.Context, ds *model.DataSource, in ExecuteInput, decryptHeaders func(string) (map[string]string, error), decryptAuth func(string) (map[string]any, error)) ([]byte, error) {
	if ds.URLTemplate == nil {
		return nil, errors.New("missing ws url")
	}
	flat := flatten(in.GlobalVars, in.Params)
	u := templatex.Render(*ds.URLTemplate, flat)
	to := time.Duration(ds.TimeoutMS) * time.Millisecond
	if to <= 0 {
		to = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	header := http.Header{}
	if ds.HeadersEncrypted != nil && *ds.HeadersEncrypted != "" {
		h, err := decryptHeaders(*ds.HeadersEncrypted)
		if err != nil {
			return nil, err
		}
		for k, v := range h {
			header.Set(k, v)
		}
	}
	auth, _ := decryptAuthPtr(ds, decryptAuth)
	applyAuthMap(header, ds.AuthType, auth)

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, u, header)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var buf strings.Builder
	deadline, _ := ctx.Deadline()
	_ = conn.SetReadDeadline(deadline)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		buf.Write(msg)
		buf.WriteByte('\n')
		if buf.Len() > 1<<20 {
			break
		}
	}
	return []byte(buf.String()), nil
}

func applyAuthMap(h http.Header, authType string, auth map[string]any) {
	tmp := map[string]string{}
	applyAuth(tmp, authType, auth)
	for k, v := range tmp {
		h.Set(k, v)
	}
}

func flatten(global map[string]any, params map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range global {
		out[k] = fmt.Sprint(v)
	}
	for k, v := range params {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func extractFromResponse(ds *model.DataSource, raw []byte) string {
	if ds.ResponseJSONPath == nil || *ds.ResponseJSONPath == "" {
		return string(raw)
	}
	path := strings.TrimSpace(*ds.ResponseJSONPath)
	// support gjson path like "data.0.price"
	res := gjson.GetBytes(raw, path)
	if res.Exists() {
		return res.String()
	}
	return string(raw)
}

func ResolveParams(schema []byte, global map[string]any, overrides map[string]any) (map[string]any, error) {
	arr, err := jsonutil.UnmarshalArray(schema)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if name == "" {
			name, _ = m["param_name"].(string)
		}
		required, _ := m["required"].(bool)
		defVal, hasDef := m["default"]
		src, _ := m["source"].(map[string]any)
		if src == nil {
			// flat keys
			st, _ := m["source_type"].(string)
			sv, _ := m["source_value"]
			src = map[string]any{"type": st, "value": sv}
		}
		typ, _ := src["type"].(string)
		if typ == "" {
			typ, _ = src["source_type"].(string)
		}
		val, _ := src["value"]
		if typ == "" {
			typ, _ = m["source_type"].(string)
		}
		if val == nil {
			val = m["source_value"]
		}
		var resolved any
		switch typ {
		case "global_var", "GLOBAL_VAR":
			key := fmt.Sprint(val)
			key = strings.Trim(key, "{} ")
			if v, ok := global[key]; ok {
				resolved = v
			}
		case "fixed_value", "FIXED":
			resolved = val
		case "expression", "EXPR":
			flat := map[string]string{}
			for k, v := range global {
				flat[k] = fmt.Sprint(v)
			}
			s, _ := templatex.EvalSimple(fmt.Sprint(val), flat)
			resolved = s
		case "runtime_input", "RUNTIME":
			if v, ok := overrides[name]; ok {
				resolved = v
			}
		default:
			if v, ok := global[name]; ok {
				resolved = v
			} else if v, ok := overrides[name]; ok {
				resolved = v
			}
		}
		if resolved == nil {
			if hasDef {
				resolved = defVal
			}
		}
		if resolved == nil && required {
			return nil, fmt.Errorf("missing required param: %s", name)
		}
		if name != "" {
			out[name] = resolved
		}
	}
	return out, nil
}

// TestUUID is a helper for tests.
var _ = uuid.Nil
