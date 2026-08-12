package eissearch

import (
	"crypto/x509"
	"testing"
)

func TestEmbeddedMintsifryCA(t *testing.T) {
	if len(embeddedRootCA) == 0 || len(embeddedSubCA) == 0 {
		t.Fatal("embedded CA PEM empty")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(embeddedRootCA) {
		t.Fatal("root CA not PEM")
	}
	if !pool.AppendCertsFromPEM(embeddedSubCA) {
		t.Fatal("sub CA not PEM")
	}
	p2 := loadCAPool("")
	if p2 == nil {
		t.Fatal("loadCAPool nil")
	}
}
