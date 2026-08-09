package eissearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rinat1313/zakupki-search/internal/models"
)

var (
	reRegHref = regexp.MustCompile(`(?i)href="([^"]*(?:regNumber|purchaseNoticeNumber)=([0-9]{10,25})[^"]*)"`)
	reLaw44   = regexp.MustCompile(`(?i)44-ФЗ`)
	reLaw223  = regexp.MustCompile(`(?i)223-ФЗ`)
)

type Hit struct {
	RegNumber  string
	NoticeURL  string
	Law        string
	ObjectName string
}

type Fetcher struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Fetcher {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://zakupki.gov.ru"
	}
	return &Fetcher{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 45 * time.Second},
	}
}

// FetchFirstPages loads up to maxPages of EIS search results and extracts hits.
func (f *Fetcher) FetchFirstPages(ctx context.Context, cfg models.SearcherConfig, maxPages int) ([]Hit, error) {
	if maxPages < 1 {
		maxPages = 1
	}
	if maxPages > 5 {
		maxPages = 5
	}
	seen := map[string]bool{}
	var out []Hit
	for page := 1; page <= maxPages; page++ {
		q := cfg.QueryValues()
		q.Set("pageNumber", fmt.Sprintf("%d", page))
		u := f.BaseURL + models.EISResultsPath + "?" + q.Encode()
		body, err := f.get(ctx, u)
		if err != nil {
			return out, err
		}
		hits := parseHits(body, f.BaseURL)
		added := 0
		for _, h := range hits {
			if seen[h.RegNumber] {
				continue
			}
			seen[h.RegNumber] = true
			out = append(out, h)
			added++
		}
		if added == 0 {
			break
		}
	}
	return out, nil
}

func (f *Fetcher) get(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "zakupki-search/0.2")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	res, err := f.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("eis %s", res.Status)
	}
	return string(b), nil
}

func parseHits(html, base string) []Hit {
	blocks := strings.Split(html, "search-registry-entry-block")
	if len(blocks) < 2 {
		// fallback: global regex
		return parseHitsGlobal(html, base)
	}
	var out []Hit
	for _, block := range blocks[1:] {
		m := reRegHref.FindStringSubmatch(block)
		if m == nil {
			continue
		}
		href := absURL(base, htmlUnescape(m[1]))
		reg := m[2]
		law := ""
		if reLaw44.MatchString(block) {
			law = "44"
		} else if reLaw223.MatchString(block) {
			law = "223"
		}
		out = append(out, Hit{
			RegNumber: reg,
			NoticeURL: href,
			Law:       law,
		})
	}
	return out
}

func parseHitsGlobal(html, base string) []Hit {
	ms := reRegHref.FindAllStringSubmatch(html, -1)
	seen := map[string]bool{}
	var out []Hit
	for _, m := range ms {
		reg := m[2]
		if seen[reg] {
			continue
		}
		seen[reg] = true
		out = append(out, Hit{
			RegNumber: reg,
			NoticeURL: absURL(base, htmlUnescape(m[1])),
		})
	}
	return out
}

func absURL(base, href string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return strings.TrimRight(base, "/") + href
	}
	return href
}

func htmlUnescape(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&quot;", `"`, "&#39;", "'", "&lt;", "<", "&gt;", ">")
	return r.Replace(s)
}
