package models

import (
	"net/url"
	"strconv"
	"strings"
)

// EISSearchConfig — параметры расширенного поиска ЕИС
// (см. /epz/order/extendedsearch/results.html и html/Закупки.html).
type EISSearchConfig struct {
	SearchString string `json:"search_string"`
	Morphology   bool   `json:"morphology"`
	StrictEqual  bool   `json:"strict_equal"`

	FZ44    bool `json:"fz44"`
	FZ223   bool `json:"fz223"`
	PPRF615 bool `json:"pp_rf_615"`

	// Этапы: af=подача заявок, ca=комиссия, pc=завершена, pa=отменена
	StageAF bool `json:"stage_af"`
	StageCA bool `json:"stage_ca"`
	StagePC bool `json:"stage_pc"`
	StagePA bool `json:"stage_pa"`

	SortBy        string `json:"sort_by"`         // UPDATE_DATE | PUBLISH_DATE | PRICE | RELEVANCE
	SortDirection bool   `json:"sort_direction"`  // false = как в UI ЕИС по умолчанию
	RecordsPerPage string `json:"records_per_page"` // _10 | _20 | _50

	PriceFrom *float64 `json:"price_from,omitempty"`
	PriceTo   *float64 `json:"price_to,omitempty"`

	PublishDateFrom string `json:"publish_date_from,omitempty"` // DD.MM.YYYY
	PublishDateTo   string `json:"publish_date_to,omitempty"`
	ApplCloseFrom   string `json:"appl_close_from,omitempty"`
	ApplCloseTo     string `json:"appl_close_to,omitempty"`

	// Заготовки под расширенные фильтры (пока как списки кодов/id).
	Regions []string `json:"regions,omitempty"`
	OKPD2   []string `json:"okpd2,omitempty"`
	CustomerTitle string `json:"customer_title,omitempty"`
}

// DefaultEISSearchConfig — разумный стартовый профиль для мониторинга.
func DefaultEISSearchConfig() EISSearchConfig {
	return EISSearchConfig{
		Morphology:     true,
		FZ44:           true,
		FZ223:          true,
		StageAF:        true,
		StageCA:        true,
		StagePC:        false,
		StagePA:        false,
		SortBy:         "UPDATE_DATE",
		SortDirection:  false,
		RecordsPerPage: "_10",
	}
}

const EISResultsPath = "/epz/order/extendedsearch/results.html"

// QueryValues строит query-параметры для results.html.
func (c EISSearchConfig) QueryValues() url.Values {
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
	rpp := c.RecordsPerPage
	if rpp == "" {
		rpp = "_10"
	}
	q.Set("recordsPerPage", rpp)
	q.Set("pageNumber", "1")

	if c.PriceFrom != nil {
		q.Set("priceFromGeneral", formatFloat(*c.PriceFrom))
	}
	if c.PriceTo != nil {
		q.Set("priceToGeneral", formatFloat(*c.PriceTo))
	}
	if c.PublishDateFrom != "" {
		q.Set("publishDateFrom", c.PublishDateFrom)
	}
	if c.PublishDateTo != "" {
		q.Set("publishDateTo", c.PublishDateTo)
	}
	if c.ApplCloseFrom != "" {
		q.Set("applSubmissionCloseDateFrom", c.ApplCloseFrom)
	}
	if c.ApplCloseTo != "" {
		q.Set("applSubmissionCloseDateTo", c.ApplCloseTo)
	}
	if t := strings.TrimSpace(c.CustomerTitle); t != "" {
		q.Set("customerTitle", t)
	}
	return q
}

// ResultsURL возвращает полный URL первой страницы поиска.
func (c EISSearchConfig) ResultsURL(base string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = "https://zakupki.gov.ru"
	}
	return base + EISResultsPath + "?" + c.QueryValues().Encode()
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
