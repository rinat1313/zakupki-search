package eissearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rinat1313/zakupki-search/internal/models"
)

// Minцифры RSA trust anchors for zakupki.gov.ru (since 2026).
// Include both Sub CA generations — wrong intermediate yields:
// "crypto/rsa: verification error" while verifying "Russian Trusted Sub CA".
//
//go:embed certs/russian_trusted_root_ca.crt
var embeddedRootCA []byte

//go:embed certs/russian_trusted_sub_ca.crt
var embeddedSubCA []byte

//go:embed certs/russian_trusted_sub_ca_2024.crt
var embeddedSubCA2024 []byte

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

type Options struct {
	BaseURL    string
	CADir      string // optional extra PEMs on disk
	Insecure   bool   // EIS_TLS_INSECURE — skip verify (GOST chains / broken intermediates)
	HTTPClient *http.Client
}

type Fetcher struct {
	BaseURL   string
	HTTP      *http.Client
	PageDelay time.Duration // 0 → DefaultPageDelay (1s)
}

func New(baseURL string) *Fetcher {
	return NewWithOptions(Options{BaseURL: baseURL})
}

func NewWithOptions(opt Options) *Fetcher {
	baseURL := strings.TrimRight(opt.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://zakupki.gov.ru"
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout:   90 * time.Second,
			Transport: newTransport(opt.CADir, opt.Insecure),
		}
	}
	return &Fetcher{BaseURL: baseURL, HTTP: client}
}

func (f *Fetcher) pageDelay() time.Duration {
	if f != nil && f.PageDelay > 0 {
		return f.PageDelay
	}
	return DefaultPageDelay
}

func newTransport(caDir string, insecure bool) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if insecure {
		log.Printf("eissearch: TLS InsecureSkipVerify=true (EIS_TLS_INSECURE)")
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.RootCAs = loadCAPool(caDir)
	}
	t.TLSClientConfig = tlsCfg
	return t
}

func loadCAPool(caDir string) *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	added := 0
	for _, pem := range [][]byte{embeddedRootCA, embeddedSubCA, embeddedSubCA2024} {
		if len(pem) == 0 {
			continue
		}
		if pool.AppendCertsFromPEM(pem) {
			added++
		}
	}
	for _, dir := range uniqueDirs(caDir) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			low := strings.ToLower(name)
			if !strings.HasSuffix(low, ".crt") && !strings.HasSuffix(low, ".pem") && !strings.HasSuffix(low, ".cer") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if pool.AppendCertsFromPEM(b) {
				added++
			}
		}
	}
	log.Printf("eissearch: TLS trust pool ready (PEM certs loaded≈%d, incl. Sub CA 2024)", added)
	return pool
}

func uniqueDirs(caDir string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(d string) {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		out = append(out, d)
	}
	add(caDir)
	add(os.Getenv("EIS_CA_DIR"))
	add("certs")
	add("/app/certs")
	return out
}

const (
	// DefaultMaxPages — потолок обхода выдачи ЕИС (или пока страницы не кончатся).
	DefaultMaxPages = 1000
	// DefaultPageDelay — пауза между страницами, чтобы не словить капчу/бан.
	DefaultPageDelay = time.Second
	pageSizeParam    = "_50"
	pageFetchRetries = 3
)

// FetchFirstPages walks EIS result pages until empty or maxPages (capped at 1000).
// Always requests 50 rows per page. Sleeps DefaultPageDelay between pages.
func (f *Fetcher) FetchFirstPages(ctx context.Context, cfg models.SearcherConfig, maxPages int) ([]Hit, error) {
	var out []Hit
	_, _, err := f.FetchPages(ctx, cfg, maxPages, func(_ int, hits []Hit) error {
		out = append(out, hits...)
		return nil
	})
	return out, err
}

// FetchPages visits page 1..maxPages (default/cap 1000) or stops when a page
// adds no new reg numbers. onPage is called with *new* hits of that page (may be empty at end).
func (f *Fetcher) FetchPages(ctx context.Context, cfg models.SearcherConfig, maxPages int, onPage func(page int, newHits []Hit) error) (total, pages int, err error) {
	if maxPages < 1 {
		maxPages = DefaultMaxPages
	}
	if maxPages > DefaultMaxPages {
		maxPages = DefaultMaxPages
	}
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return total, pages, err
		}
		if page > 1 {
			select {
			case <-ctx.Done():
				return total, pages, ctx.Err()
			case <-time.After(f.pageDelay()):
			}
		}
		q := cfg.QueryValues()
		q.Set("pageNumber", fmt.Sprintf("%d", page))
		q.Set("recordsPerPage", pageSizeParam)
		u := f.BaseURL + models.EISResultsPath + "?" + q.Encode()
		body, gerr := f.getRetry(ctx, u)
		if gerr != nil {
			return total, pages, gerr
		}
		hits := parseHits(body, f.BaseURL)
		var fresh []Hit
		for _, h := range hits {
			if seen[h.RegNumber] {
				continue
			}
			seen[h.RegNumber] = true
			fresh = append(fresh, h)
		}
		pages = page
		total += len(fresh)
		if onPage != nil {
			if err := onPage(page, fresh); err != nil {
				return total, pages, err
			}
		}
		if len(fresh) == 0 {
			break
		}
	}
	return total, pages, nil
}

func (f *Fetcher) getRetry(ctx context.Context, u string) (string, error) {
	var last error
	for i := 0; i < pageFetchRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(f.pageDelay() * time.Duration(i+1)):
			}
			log.Printf("eissearch: retry %d GET %s", i+1, u)
		}
		body, err := f.get(ctx, u)
		if err == nil {
			return body, nil
		}
		last = err
	}
	return "", last
}

func (f *Fetcher) get(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; zakupki-search/0.3; +https://github.com/rinat1313/zakupki-search)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
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
