package coreclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client — тонкий клиент к zakupki-core для списка тендеров и sync пула.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Enabled() bool { return c != nil && c.BaseURL != "" }

type SyncItem struct {
	RegNumber  string `json:"reg_number"`
	SourceSite string `json:"source_site,omitempty"`
	NoticeURL  string `json:"notice_url,omitempty"`
	Law        string `json:"law,omitempty"`
	ObjectName string `json:"object_name,omitempty"`
}

type SyncRequest struct {
	Title         string     `json:"title,omitempty"`
	ConfigVersion int64      `json:"config_version,omitempty"`
	Items         []SyncItem `json:"items"`
	Enqueue       bool       `json:"enqueue"`
}

func (c *Client) SyncSearchConfig(ctx context.Context, searchConfigID, title string, configVersion int64, items []SyncItem, enqueue bool) error {
	if !c.Enabled() {
		return fmt.Errorf("core client disabled")
	}
	body, _ := json.Marshal(SyncRequest{
		Title:         title,
		ConfigVersion: configVersion,
		Items:         items,
		Enqueue:       enqueue,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/api/v1/categories/by-search-config/"+url.PathEscape(searchConfigID)+"/sync",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("core sync %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) ListTendersBySearchConfig(ctx context.Context, searchConfigID, q string) (json.RawMessage, error) {
	if !c.Enabled() {
		return json.RawMessage(`{"items":[],"total":0}`), nil
	}
	u, _ := url.Parse(c.BaseURL + "/api/v1/tenders")
	qs := u.Query()
	qs.Set("search_config_id", searchConfigID)
	if strings.TrimSpace(q) != "" {
		qs.Set("q", q)
	}
	u.RawQuery = qs.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("core tenders %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	return json.RawMessage(b), nil
}

func (c *Client) TendersCount(ctx context.Context, searchConfigID string) int {
	raw, err := c.ListTendersBySearchConfig(ctx, searchConfigID, "")
	if err != nil {
		return 0
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asMap); err == nil {
		if t, ok := asMap["total"]; ok {
			var n int
			if json.Unmarshal(t, &n) == nil {
				return n
			}
		}
		if items, ok := asMap["items"]; ok {
			var arr []json.RawMessage
			if json.Unmarshal(items, &arr) == nil {
				return len(arr)
			}
		}
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return len(arr)
	}
	return 0
}

func (c *Client) EnsureCategory(ctx context.Context, searchConfigID, title string) error {
	if !c.Enabled() {
		return nil
	}
	// Try lookup first.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/api/v1/categories/by-search-config/"+url.PathEscape(searchConfigID), nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"title":            title,
		"search_config_id": searchConfigID,
	})
	creq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/categories", bytes.NewReader(body))
	if err != nil {
		return err
	}
	creq.Header.Set("Content-Type", "application/json")
	cres, err := c.HTTP.Do(creq)
	if err != nil {
		return err
	}
	defer cres.Body.Close()
	if cres.StatusCode >= 300 && cres.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(io.LimitReader(cres.Body, 2048))
		return fmt.Errorf("core create category %s: %s", cres.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
