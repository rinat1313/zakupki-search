package eissearch

import (
	"crypto/x509"
	"testing"
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
