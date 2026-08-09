package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearcherConfigUIRoundTrip(t *testing.T) {
	raw := []byte(`{
		"search_string":"сервер",
		"morphology":true,
		"strict_equal":false,
		"fz44":true,
		"fz223":false,
		"pp_rf615":false,
		"stage_af":true,
		"stage_ca":true,
		"stage_pc":false,
		"stage_pa":false,
		"publish_date_from":null,
		"records_per_page":50,
		"okpd2_codes":"62.01",
		"okpd2_with_nested":true
	}`)
	var c SearcherConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.RecordsPerPage != 50 || c.SearchString != "сервер" || !c.FZ44 {
		t.Fatalf("%+v", c)
	}
	q := c.QueryValues()
	if q.Get("recordsPerPage") != "_50" || q.Get("fz44") != "on" {
		t.Fatalf("%v", q)
	}
	if !strings.Contains(c.ResultsURL(""), "searchString=") {
		t.Fatal(c.ResultsURL(""))
	}
}

func TestSearcherConfigLegacyKeys(t *testing.T) {
	raw := []byte(`{"search_string":"x","pp_rf_615":true,"records_per_page":"_20","appl_close_from":"01.01.2026"}`)
	var c SearcherConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if !c.PPRF615 || c.RecordsPerPage != 20 || strPtrVal(c.ApplCloseFrom) != "01.01.2026" {
		t.Fatalf("%+v", c)
	}
}
