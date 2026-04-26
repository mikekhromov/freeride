package hiddify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Domain     string
	AdminPath  string
	ClientPath string
	APIKey     string
	HTTP       *http.Client
}

type MTProxyEntry struct {
	Link  string `json:"link"`
	Title string `json:"title"`
}

func New(domain, adminPath, clientPath, apiKey string) *Client {
	return &Client{
		Domain:     strings.TrimSuffix(domain, "/"),
		AdminPath:  strings.Trim(strings.TrimSpace(adminPath), "/"),
		ClientPath: strings.Trim(strings.TrimSpace(clientPath), "/"),
		APIKey:     strings.TrimSpace(apiKey),
		HTTP: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *Client) enabled() bool {
	return c.Domain != "" && c.AdminPath != "" && c.ClientPath != "" && c.APIKey != ""
}

func (c *Client) adminURL(path string) string {
	return fmt.Sprintf("%s/%s/api/v2/admin%s", c.Domain, c.AdminPath, path)
}

func (c *Client) clientURL(path string) string {
	return fmt.Sprintf("%s/%s/api/v2/user%s", c.Domain, c.ClientPath, path)
}

func (c *Client) doJSON(ctx context.Context, method, rawURL string, body any, out any, apiKey string) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt-1) * 250 * time.Millisecond):
			}
		}

		var rdr io.Reader
		if bodyBytes != nil {
			rdr = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, rawURL, rdr)
		if err != nil {
			return err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Hiddify-API-Key", apiKey)

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("hiddify: %s %s failed: %d %s", method, rawURL, resp.StatusCode, strings.TrimSpace(string(b)))
			if resp.StatusCode >= 500 {
				continue
			}
			return lastErr
		}
		if out == nil {
			_ = resp.Body.Close()
			return nil
		}
		decErr := json.NewDecoder(resp.Body).Decode(out)
		_ = resp.Body.Close()
		if decErr == io.EOF {
			return nil
		}
		if decErr != nil {
			lastErr = decErr
			return decErr
		}
		return nil
	}

	if lastErr == nil {
		return nil
	}
	return lastErr
}

func (c *Client) CreateUser(ctx context.Context, name string, packageDays int, usageLimitGB int) (string, error) {
	if !c.enabled() {
		return "", fmt.Errorf("hiddify: config is incomplete")
	}

	req := map[string]any{
		"name":           name,
		"package_days":   packageDays,
		"usage_limit_GB": usageLimitGB,
	}
	var user struct {
		UUID string `json:"uuid"`
	}
	if err := c.doJSON(ctx, http.MethodPost, c.adminURL("/user/"), req, &user, c.APIKey); err != nil {
		return "", err
	}
	if user.UUID == "" {
		return "", fmt.Errorf("hiddify: empty uuid in CreateUser response")
	}
	return user.UUID, nil
}

func (c *Client) DeleteUser(ctx context.Context, userUUID string) error {
	if strings.TrimSpace(userUUID) == "" {
		return nil
	}
	if !c.enabled() {
		return fmt.Errorf("hiddify: config is incomplete")
	}
	return c.doJSON(ctx, http.MethodDelete, c.adminURL("/user/"+url.PathEscape(userUUID)+"/"), nil, nil, c.APIKey)
}

func (c *Client) ProfileURLByUUID(ctx context.Context, userUUID string) (string, error) {
	if strings.TrimSpace(userUUID) == "" {
		return "", fmt.Errorf("hiddify: empty user uuid")
	}
	if !c.enabled() {
		return "", fmt.Errorf("hiddify: config is incomplete")
	}
	var me struct {
		ProfileURL string `json:"profile_url"`
	}
	if err := c.doJSON(ctx, http.MethodGet, c.clientURL("/me/"), nil, &me, userUUID); err != nil {
		return "", err
	}
	if me.ProfileURL == "" {
		return "", fmt.Errorf("hiddify: profile_url is empty")
	}
	return me.ProfileURL, nil
}

func (c *Client) MTProxyLinkByUUID(ctx context.Context, userUUID string) (string, error) {
	if strings.TrimSpace(userUUID) == "" {
		return "", fmt.Errorf("hiddify: empty user uuid")
	}
	if !c.enabled() {
		return "", fmt.Errorf("hiddify: config is incomplete")
	}
	var links []MTProxyEntry
	if err := c.doJSON(ctx, http.MethodGet, c.clientURL("/mtproxies/"), nil, &links, userUUID); err != nil {
		return "", err
	}
	if len(links) == 0 || strings.TrimSpace(links[0].Link) == "" {
		return "", fmt.Errorf("hiddify: mtproxy link is empty")
	}
	return links[0].Link, nil
}

func (c *Client) UsageByUUID(ctx context.Context, userUUID string) (usedGB, limitGB float64, err error) {
	if strings.TrimSpace(userUUID) == "" {
		return 0, 0, fmt.Errorf("hiddify: empty user uuid")
	}
	if !c.enabled() {
		return 0, 0, fmt.Errorf("hiddify: config is incomplete")
	}
	var raw map[string]any
	if err := c.doJSON(ctx, http.MethodGet, c.clientURL("/me/"), nil, &raw, userUUID); err != nil {
		return 0, 0, err
	}
	usedGB = pickFloat(raw, "current_usage_GB", "usage_current_GB", "used_traffic_GB")
	limitGB = pickFloat(raw, "usage_limit_GB", "package_usage_limit_GB", "traffic_limit_GB")
	if limitGB <= 0 {
		return usedGB, 0, nil
	}
	return usedGB, limitGB, nil
}

func (c *Client) AdminUsersCount(ctx context.Context) (int, error) {
	if !c.enabled() {
		return 0, fmt.Errorf("hiddify: config is incomplete")
	}

	var raw any
	if err := c.doJSON(ctx, http.MethodGet, c.adminURL("/user/"), nil, &raw, c.APIKey); err != nil {
		return 0, err
	}

	switch v := raw.(type) {
	case []any:
		return len(v), nil
	case map[string]any:
		if count, ok := asInt(v["count"]); ok {
			return count, nil
		}
		if items, ok := v["items"].([]any); ok {
			return len(items), nil
		}
		if results, ok := v["results"].([]any); ok {
			return len(results), nil
		}
	}
	return 0, fmt.Errorf("hiddify: unsupported /admin/user response shape")
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	default:
		return 0, false
	}
}

func pickFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch x := v.(type) {
		case float64:
			return x
		case float32:
			return float64(x)
		case int:
			return float64(x)
		case int64:
			return float64(x)
		case string:
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			if f, err := strconv.ParseFloat(x, 64); err == nil {
				return f
			}
		}
	}
	return 0
}
