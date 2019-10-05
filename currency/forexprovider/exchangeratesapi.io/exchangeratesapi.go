package exchangerates

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	exchangeRatesAPI                 = "https://api.exchangeratesapi.io"
	exchangeRatesLatest              = "latest"
	exchangeRatesHistory             = "history"
	exchangeRatesSupportedCurrencies = "EUR,CHF,USD,BRL,ISK,PHP,KRW,BGN,MXN," +
		"RON,CAD,SGD,NZD,THB,HKD,JPY,NOK,HRK,ILS,GBP,DKK,HUF,MYR,RUB,TRY,IDR," +
		"ZAR,INR,AUD,CZK,SEK,CNY,PLN"

	authRate   = 0
	unAuthRate = 0
)

// ExchangeRates stores the struct for the ExchangeRatesAPI API
type ExchangeRates struct {
	base.Base
	Requester *request.Requester
}

// Setup sets appropriate values for CurrencyLayer
func (e *ExchangeRates) Setup(config base.Settings) error {
	e.Name = config.Name
	e.Enabled = config.Enabled
	e.RESTPollingDelay = config.RESTPollingDelay
	e.Verbose = config.Verbose
	e.PrimaryProvider = config.PrimaryProvider
	e.Requester = request.New(e.Name,
		request.NewRateLimit(time.Second*10, authRate),
		request.NewRateLimit(time.Second*10, unAuthRate),
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))
	return nil
}

func cleanCurrencies(baseCurrency, symbols string) string {
	var cleanedCurrencies []string
	symbols = strings.Replace(symbols, "RUR", "RUB", -1)
	var s = strings.Split(symbols, ",")
	for _, x := range s {
		// first make sure that the baseCurrency is not in the symbols list
		// if it is set
		if baseCurrency != "" {
			if x == baseCurrency {
				continue
			}
		} else {
			// otherwise since the baseCurrency is empty, make sure that it
			// does not exist in the symbols list
			if x == "EUR" {
				continue
			}
		}

		// remove and warn about any unsupported currencies
		if !common.StringContains(exchangeRatesSupportedCurrencies, x) {
			log.Warnf("Forex provider ExchangeRatesAPI does not support currency %s, removing from forex rates query.", x)
			continue
		}
		cleanedCurrencies = append(cleanedCurrencies, x)
	}
	return strings.Join(cleanedCurrencies, ",")
}

// GetLatestRates returns a map of forex rates based on the supplied params
// baseCurrency - USD	[optional] The base currency to use for forex rates, defaults to EUR
// symbols - AUD,USD	[optional] The symbols to query the forex rates for, default is
// all supported currencies
func (e *ExchangeRates) GetLatestRates(baseCurrency, symbols string) (Rates, error) {
	vals := url.Values{}

	if len(baseCurrency) > 0 {
		vals.Set("base", baseCurrency)
	}

	if len(symbols) > 0 {
		symbols = cleanCurrencies(baseCurrency, symbols)
		vals.Set("symbols", symbols)
	}

	var result Rates
	return result, e.SendHTTPRequest(exchangeRatesLatest, vals, &result)
}

// GetHistoricalRates returns historical exchange rate data for all available or
// a specific set of currencies.
// date - YYYY-MM-DD	[required] A date in the past
// baseCurrency - USD 			[optional] The base currency to use for forex rates, defaults to EUR
// symbols - AUD,USD	[optional] The symbols to query the forex rates for, default is
// all supported currencies
func (e *ExchangeRates) GetHistoricalRates(date, baseCurrency string, symbols []string) (HistoricalRates, error) {
	var resp HistoricalRates
	v := url.Values{}

	if len(symbols) > 0 {
		s := cleanCurrencies(baseCurrency, strings.Join(symbols, ","))
		v.Set("symbols", s)
	}

	if len(baseCurrency) > 0 {
		v.Set("base", baseCurrency)
	}

	return resp, e.SendHTTPRequest(date, v, &resp)
}

// GetTimeSeriesRates returns daily historical exchange rate data between two
// specified dates for all available or a specific set of currencies.
// startDate - YYYY-MM-DD	[required] A date in the past
// endDate - YYYY-MM-DD	[required] A date in the past but greater than the startDate
// baseCurrency - USD 	[optional] The base currency to use for forex rates, defaults to EUR
// symbols - AUD,USD 	[optional] The symbols to query the forex rates for, default is
// all supported currencies
func (e *ExchangeRates) GetTimeSeriesRates(startDate, endDate, baseCurrency string, symbols []string) (TimeSeriesRates, error) {
	var resp TimeSeriesRates
	if startDate == "" || endDate == "" {
		return resp, errors.New("startDate and endDate params must be set")
	}

	v := url.Values{}
	v.Set("start_at", startDate)
	v.Set("end_at", endDate)

	if len(baseCurrency) > 0 {
		v.Set("base", baseCurrency)
	}

	if len(symbols) > 0 {
		s := cleanCurrencies(baseCurrency, strings.Join(symbols, ","))
		v.Set("symbols", s)
	}

	return resp, e.SendHTTPRequest(exchangeRatesHistory, v, &resp)
}

// GetRates is a wrapper function to return forex rates
func (e *ExchangeRates) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	result, err := e.GetLatestRates(baseCurrency, symbols)
	if err != nil {
		return nil, err
	}

	standardisedRates := make(map[string]float64)
	for k, v := range result.Rates {
		curr := baseCurrency + k
		standardisedRates[curr] = v
	}

	return standardisedRates, nil
}

// GetSupportedCurrencies returns the supported currency list
func (e *ExchangeRates) GetSupportedCurrencies() ([]string, error) {
	return common.SplitStrings(exchangeRatesSupportedCurrencies, ","), nil
}

// SendHTTPRequest sends a HTTPS request to the desired endpoint and returns the result
func (e *ExchangeRates) SendHTTPRequest(endPoint string, values url.Values, result interface{}) error {
	path := common.EncodeURLValues(exchangeRatesAPI+"/"+endPoint, values)
	err := e.Requester.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&result,
		false,
		false,
		e.Verbose,
		false)
	if err != nil {
		return fmt.Errorf("exchangeRatesAPI SendHTTPRequest error %s with path %s",
			err,
			path)
	}
	return nil
}
