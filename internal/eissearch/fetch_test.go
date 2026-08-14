package eissearch

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rinat1313/zakupki-search/internal/models"
)

func TestEmbeddedMintsifryCA(t *testing.T) {
	if len(embeddedRootCA) == 0 || len(embeddedSubCA) == 0 || len(embeddedSubCA2024) == 0 {
		t.Fatal("embedded CA PEM empty")
	}
	pool := x509.NewCertPool()
	for i, pem := range [][]byte{embeddedRootCA, embeddedSubCA, embeddedSubCA2024} {
		if !pool.AppendCertsFromPEM(pem) {
			t.Fatalf("PEM %d not accepted", i)
		}
	}
	if p := loadCAPool(""); p == nil {
		t.Fatal("loadCAPool nil")
	}
}

func TestFetchPagesStopsWhenEmptyAndCaps(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("pageNumber")
		rpp := r.URL.Query().Get("recordsPerPage")
		seen = append(seen, page+"/"+rpp)
		w.Header().Set("Content-Type", "text/html")
		switch page {
		case "1":
			fmt.Fprint(w, searchBlock("1111111111111111111"))
		case "2":
			fmt.Fprint(w, searchBlock("2222222222222222222"))
		default:
			fmt.Fprint(w, "<html>empty</html>")
		}
	}))
	defer srv.Close()

	f := New(srv.URL)
	f.PageDelay = time.Millisecond
	cfg := models.DefaultSearcherConfig()
	cfg.RecordsPerPage = 10 // must still request _50
	total, pages, err := f.FetchPages(context.Background(), cfg, 1000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || pages != 3 {
		t.Fatalf("total=%d pages=%d want 2/3", total, pages)
	}
	if len(seen) != 3 {
		t.Fatalf("requests=%v", seen)
	}
	for _, s := range seen {
		if !strings.HasSuffix(s, "/_50") {
			t.Fatalf("expected recordsPerPage=_50, got %s", s)
		}
	}
}

func searchBlock(reg string) string {
	return `search-registry-entry-block <a href="/epz/order/notice/view.html?regNumber=` + reg + `">x</a> 44-ФЗ`
}
