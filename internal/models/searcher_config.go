package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SearcherConfig — конфиг фильтров ЕИС в форме, которую ждёт UI gateway
// (/api/v1/searchers, ui/searchers.js).
type SearcherConfig struct {
	SearchString string `json:"search_string"`
	Morphology   bool   `json:"morphology"`
	StrictEqual  bool   `json:"strict_equal"`

	FZ44    bool `json:"fz44"`
	FZ223   bool `json:"fz223"`
	PPRF615 bool `json:"pp_rf615"`

	StageAF bool `json:"stage_af"`
	StageCA bool `json:"stage_ca"`
	StagePC bool `json:"stage_pc"`
	StagePA bool `json:"stage_pa"`

	PublishDateFrom *string `json:"publish_date_from"`
	PublishDateTo   *string `json:"publish_date_to"`
	ApplCloseFrom   *string `json:"appl_submission_close_date_from"`
	ApplCloseTo     *string `json:"appl_submission_close_date_to"`

	PriceFrom *float64 `json:"price_from"`
	PriceTo   *float64 `json:"price_to"`
	CurrencyCode string `json:"currency_code"`

	OKPD2Codes      string `json:"okpd2_codes"`
	OKPD2WithNested bool   `json:"okpd2_with_nested"`
	OKPD2Several    bool   `json:"okpd2_several"`

	CustomerTitle  string `json:"customer_title"`
	PlacingWayList string `json:"placing_way_list"`
	SMPSono        bool   `json:"smp_sono"`
	JointPurchase  bool   `json:"joint_purchase"`

	SortBy         string `json:"sort_by"`
	SortDirection  bool   `json:"sort_direction"`
	RecordsPerPage int    `json:"records_per_page"` // 10|20|50 (UI); stored also accepts "_10"
}

func DefaultSearcherConfig() SearcherConfig {
	return SearcherConfig{
		Morphology:      true,
		FZ44:            true,
		FZ223:           true,
		StageAF:         true,
		StageCA:         true,
		OKPD2WithNested: true,
		CurrencyCode:    "RUB",
		SortBy:          "UPDATE_DATE",
		RecordsPerPage:  50,
	}
}

// UnmarshalJSON accepts both UI field names and legacy eis_config keys.
func (c *SearcherConfig) UnmarshalJSON(b []byte) error {
	type alias SearcherConfig
	aux := struct {
		alias
		LegacyPP      *bool            `json:"pp_rf_615"`
		LegacyRPP     json.RawMessage  `json:"records_per_page"`
		LegacyApplFrom string          `json:"appl_close_from"`
		LegacyApplTo   string          `json:"appl_close_to"`
		LegacyOKPD2    []string        `json:"okpd2"`
	}{alias: alias(DefaultSearcherConfig())}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*c = SearcherConfig(aux.alias)
	if aux.LegacyPP != nil && !c.PPRF615 {
		c.PPRF615 = *aux.LegacyPP
	}
	if len(aux.LegacyRPP) > 0 {
		c.RecordsPerPage = parseRecordsPerPage(aux.LegacyRPP)
	}
	if c.RecordsPerPage == 0 {
		c.RecordsPerPage = 50
	}
	if (c.ApplCloseFrom == nil || *c.ApplCloseFrom == "") && aux.LegacyApplFrom != "" {
		v := aux.LegacyApplFrom
		c.ApplCloseFrom = &v
	}
	if (c.ApplCloseTo == nil || *c.ApplCloseTo == "") && aux.LegacyApplTo != "" {
		v := aux.LegacyApplTo
		c.ApplCloseTo = &v
	}
	if c.OKPD2Codes == "" && len(aux.LegacyOKPD2) > 0 {
		c.OKPD2Codes = strings.Join(aux.LegacyOKPD2, ",")
	}
	return nil
}

func parseRecordsPerPage(raw json.RawMessage) int {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil && n > 0 {
		return n
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimPrefix(strings.TrimSpace(s), "_")
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 50
}

func (c SearcherConfig) recordsPerPageParam() string {
	n := c.RecordsPerPage
	if n != 10 && n != 20 && n != 50 {
		n = 50
	}
	return fmt.Sprintf("_%d", n)
}

func strPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// QueryValues maps config to EIS extendedsearch query.
func (c SearcherConfig) QueryValues() url.Values {
	q := url.Values{}
	if s := strings.TrimSpace(c.SearchString); s != "" {
		q.Set("searchString", s)
	}
	if c.Morphology && !c.StrictEqual {
		q.Set("morphology", "on")
	}
	if c.StrictEqual {
		q.Set("strictEqual", "true")
	}
	if c.FZ44 {
		q.Set("fz44", "on")
	}
	if c.FZ223 {
		q.Set("fz223", "on")
	}
	if c.PPRF615 {
		q.Set("ppRf615", "on")
	}
	if c.StageAF {
		q.Set("af", "on")
	}
	if c.StageCA {
		q.Set("ca", "on")
	}
	if c.StagePC {
		q.Set("pc", "on")
	}
	if c.StagePA {
		q.Set("pa", "on")
	}
	sortBy := c.SortBy
	if sortBy == "" {
		sortBy = "UPDATE_DATE"
	}
	q.Set("sortBy", sortBy)
	q.Set("sortDirection", strconv.FormatBool(c.SortDirection))
	q.Set("recordsPerPage", c.recordsPerPageParam())
	q.Set("pageNumber", "1")

	if c.PriceFrom != nil {
		q.Set("priceFromGeneral", formatFloat(*c.PriceFrom))
	}
	if c.PriceTo != nil {
		q.Set("priceToGeneral", formatFloat(*c.PriceTo))
	}
	if v := strPtrVal(c.PublishDateFrom); v != "" {
		q.Set("publishDateFrom", v)
	}
	if v := strPtrVal(c.PublishDateTo); v != "" {
		q.Set("publishDateTo", v)
	}
	if v := strPtrVal(c.ApplCloseFrom); v != "" {
		q.Set("applSubmissionCloseDateFrom", v)
	}
	if v := strPtrVal(c.ApplCloseTo); v != "" {
		q.Set("applSubmissionCloseDateTo", v)
	}
	if t := strings.TrimSpace(c.CustomerTitle); t != "" {
		q.Set("customerTitle", t)
	}
	if codes := strings.TrimSpace(c.OKPD2Codes); codes != "" {
		q.Set("okpd2IdsCodes", codes)
	}
	if c.OKPD2WithNested {
		q.Set("okpd2IdsWithNested", "on")
	}
	if c.OKPD2Several {
		q.Set("okpd2IdsSeveral", "on")
	}
	if p := strings.TrimSpace(c.PlacingWayList); p != "" {
		q.Set("placingWayList", p)
	}
	if c.SMPSono {
		q.Set("procurementSMPAndSONO", "on")
	}
	if c.JointPurchase {
		q.Set("jointPurchase", "on")
	}
	if cur := strings.TrimSpace(c.CurrencyCode); cur != "" && cur != "RUB" {
		q.Set("currencyCodeGeneral", cur)
	}
	return q
}

func (c SearcherConfig) ResultsURL(base string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = "https://zakupki.gov.ru"
	}
	return base + EISResultsPath + "?" + c.QueryValues().Encode()
}

// ToEISSearchConfig adapts to the older typed shape used by /search-profiles.
func (c SearcherConfig) ToEISSearchConfig() EISSearchConfig {
	out := DefaultEISSearchConfig()
	out.SearchString = c.SearchString
	out.Morphology = c.Morphology
	out.StrictEqual = c.StrictEqual
	out.FZ44 = c.FZ44
	out.FZ223 = c.FZ223
	out.PPRF615 = c.PPRF615
	out.StageAF = c.StageAF
	out.StageCA = c.StageCA
	out.StagePC = c.StagePC
	out.StagePA = c.StagePA
	out.SortBy = c.SortBy
	out.SortDirection = c.SortDirection
	out.RecordsPerPage = c.recordsPerPageParam()
	out.PriceFrom = c.PriceFrom
	out.PriceTo = c.PriceTo
	out.PublishDateFrom = strPtrVal(c.PublishDateFrom)
	out.PublishDateTo = strPtrVal(c.PublishDateTo)
	out.ApplCloseFrom = strPtrVal(c.ApplCloseFrom)
	out.ApplCloseTo = strPtrVal(c.ApplCloseTo)
	out.CustomerTitle = c.CustomerTitle
	if codes := strings.TrimSpace(c.OKPD2Codes); codes != "" {
		for _, p := range strings.Split(codes, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out.OKPD2 = append(out.OKPD2, p)
			}
		}
	}
	return out
}

func SearcherConfigFromEIS(c EISSearchConfig) SearcherConfig {
	out := DefaultSearcherConfig()
	out.SearchString = c.SearchString
	out.Morphology = c.Morphology
	out.StrictEqual = c.StrictEqual
	out.FZ44 = c.FZ44
	out.FZ223 = c.FZ223
	out.PPRF615 = c.PPRF615
	out.StageAF = c.StageAF
	out.StageCA = c.StageCA
	out.StagePC = c.StagePC
	out.StagePA = c.StagePA
	out.SortBy = c.SortBy
	out.SortDirection = c.SortDirection
	out.RecordsPerPage = parseRecordsPerPage([]byte(strconv.Quote(c.RecordsPerPage)))
	if out.RecordsPerPage == 0 {
		out.RecordsPerPage = 50
	}
	out.PriceFrom = c.PriceFrom
	out.PriceTo = c.PriceTo
	if c.PublishDateFrom != "" {
		v := c.PublishDateFrom
		out.PublishDateFrom = &v
	}
	if c.PublishDateTo != "" {
		v := c.PublishDateTo
		out.PublishDateTo = &v
	}
	if c.ApplCloseFrom != "" {
		v := c.ApplCloseFrom
		out.ApplCloseFrom = &v
	}
	if c.ApplCloseTo != "" {
		v := c.ApplCloseTo
		out.ApplCloseTo = &v
	}
	out.CustomerTitle = c.CustomerTitle
	if len(c.OKPD2) > 0 {
		out.OKPD2Codes = strings.Join(c.OKPD2, ",")
	}
	return out
}
