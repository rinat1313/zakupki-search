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

// Official Minцифры roots (zakupki.gov.ru since 2026). Embedded so Docker/cwd paths cannot break TLS.
//
//go:embed certs/russian_trusted_root_ca.crt
var embeddedRootCA []byte

//go:embed certs/russian_trusted_sub_ca.crt
var embeddedSubCA []byte

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
	Insecure   bool   // EIS_TLS_INSECURE — last resort
	HTTPClient *http.Client
}

type Fetcher struct {
	BaseURL string
	HTTP    *http.Client
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
			Timeout:   45 * time.Second,
			Transport: newTransport(opt.CADir, opt.Insecure),
		}
	}
	return &Fetcher{BaseURL: baseURL, HTTP: client}
}

func newTransport(caDir string, insecure bool) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if insecure {
		log.Printf("eissearch: TLS InsecureSkipVerify enabled (EIS_TLS_INSECURE)")
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.RootCAs = loadCAPool(caDir)
	}
	t.TLSClientConfig = tlsCfg
	return t
}

// loadCAPool = system roots + embedded Minцифры CA + optional EIS_CA_DIR PEMs.
func loadCAPool(caDir string) *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	added := 0
	for _, pem := range [][]byte{embeddedRootCA, embeddedSubCA} {
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
	log.Printf("eissearch: TLS trust pool ready (embedded/disk PEM certs loaded≈%d)", added)
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
