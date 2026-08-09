package models

import (
	"strings"
	"testing"
)

func TestResultsURLContainsCoreParams(t *testing.T) {
	cfg := DefaultEISSearchConfig()
	cfg.SearchString = "Разработка ПО"
	cfg.FZ223 = false
	u := cfg.ResultsURL("https://zakupki.gov.ru")
	if !strings.Contains(u, "/epz/order/extendedsearch/results.html?") {
		t.Fatalf("path: %s", u)
	}
	for _, want := range []string{
		"searchString=",
		"fz44=on",
		"af=on",
		"recordsPerPage=_10",
		"sortBy=UPDATE_DATE",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("missing %q in %s", want, u)
		}
	}
	if strings.Contains(u, "fz223=on") {
		t.Fatalf("fz223 should be off: %s", u)
	}
}

func TestStrictEqualDisablesMorphologyFlag(t *testing.T) {
	cfg := DefaultEISSearchConfig()
	cfg.SearchString = "0134300097526000797"
	cfg.StrictEqual = true
	q := cfg.QueryValues()
	if q.Get("strictEqual") != "true" {
		t.Fatal("strictEqual")
	}
	if q.Get("morphology") != "" {
		t.Fatalf("morphology should be empty, got %q", q.Get("morphology"))
	}
}
