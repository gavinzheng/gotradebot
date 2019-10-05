package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// AllEnabledExchangeOrderbooks holds the enabled exchange orderbooks
type AllEnabledExchangeOrderbooks struct {
	Data []EnabledExchangeOrderbooks `json:"data"`
}

// EnabledExchangeOrderbooks is a sub type for singular exchanges and respective
// orderbooks
type EnabledExchangeOrderbooks struct {
	ExchangeName   string           `json:"exchangeName"`
	ExchangeValues []orderbook.Base `json:"exchangeValues"`
}

// AllEnabledExchangeCurrencies holds the enabled exchange currencies
type AllEnabledExchangeCurrencies struct {
	Data []EnabledExchangeCurrencies `json:"data"`
}

// EnabledExchangeCurrencies is a sub type for singular exchanges and respective
// currencies
type EnabledExchangeCurrencies struct {
	ExchangeName   string         `json:"exchangeName"`
	ExchangeValues []ticker.Price `json:"exchangeValues"`
}

// AllEnabledExchangeAccounts holds all enabled accounts info
type AllEnabledExchangeAccounts struct {
	Data []exchange.AccountInfo `json:"data"`
}

// RESTfulJSONResponse outputs a JSON response of the response interface
func RESTfulJSONResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// RESTfulError prints the REST method and error
func RESTfulError(method string, err error) {
	log.Errorf("RESTful %s: server failed to send JSON response. Error %s",
		method, err)
}

// RESTGetAllSettings replies to a request with an encoded JSON response about the
// trading bots configuration.
func RESTGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := RESTfulJSONResponse(w, bot.config)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTSaveAllSettings saves all current settings from request body as a JSON
// document then reloads state and returns the settings
func RESTSaveAllSettings(w http.ResponseWriter, r *http.Request) {
	// Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.Post
	err := decoder.Decode(&responseData)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	// Save change the settings
	err = bot.config.UpdateConfig(bot.configFile, &responseData.Data)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	err = RESTfulJSONResponse(w, bot.config)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	SetupExchanges()
}

// RESTGetOrderbook returns orderbook info for a given currency, exchange and
// asset type
func RESTGetOrderbook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	exchangeName := vars["exchangeName"]
	assetType := vars["assetType"]

	if assetType == "" {
		assetType = orderbook.Spot
	}

	response, err := GetSpecificOrderbook(currency, exchangeName, assetType)
	if err != nil {
		log.Errorf("Failed to fetch orderbook for %s currency: %s\n", exchangeName,
			currency)
		return
	}

	err = RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// GetAllActiveOrderbooks returns all enabled exchanges orderbooks
func GetAllActiveOrderbooks() []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks

	for _, individualBot := range bot.exchanges {
		if individualBot == nil || !individualBot.IsEnabled() {
			continue
		}

		var individualExchange EnabledExchangeOrderbooks
		exchangeName := individualBot.GetName()
		individualExchange.ExchangeName = exchangeName
		currencies := individualBot.GetEnabledCurrencies()
		assetTypes, err := exchange.GetExchangeAssetTypes(exchangeName)
		if err != nil {
			log.Errorf("failed to get %s exchange asset types. Error: %s",
				exchangeName, err)
			continue
		}
		for _, x := range currencies {
			pair := x

			var ob orderbook.Base
			if len(assetTypes) > 1 {
				for y := range assetTypes {
					ob, err = individualBot.GetOrderbookEx(pair,
						assetTypes[y])
				}
			} else {
				ob, err = individualBot.GetOrderbookEx(pair,
					assetTypes[0])
			}

			if err != nil {
				log.Errorf("failed to get %s %s orderbook. Error: %s",
					pair,
					exchangeName,
					err)
				continue
			}

			individualExchange.ExchangeValues = append(
				individualExchange.ExchangeValues, ob,
			)
		}
		orderbookData = append(orderbookData, individualExchange)

	}
	return orderbookData
}

// RESTGetAllActiveOrderbooks returns all enabled exchange orderbooks
func RESTGetAllActiveOrderbooks(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = GetAllActiveOrderbooks()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetPortfolio returns the bot portfolio
func RESTGetPortfolio(w http.ResponseWriter, r *http.Request) {
	result := bot.portfolio.GetPortfolioSummary()
	err := RESTfulJSONResponse(w, result)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetTicker returns ticker info for a given currency, exchange and
// asset type
func RESTGetTicker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	exchangeName := vars["exchangeName"]
	assetType := vars["assetType"]

	if assetType == "" {
		assetType = ticker.Spot
	}
	response, err := GetSpecificTicker(currency, exchangeName, assetType)
	if err != nil {
		log.Errorf("Failed to fetch ticker for %s currency: %s\n", exchangeName,
			currency)
		return
	}
	err = RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// GetAllActiveTickers returns all enabled exchange tickers
func GetAllActiveTickers() []EnabledExchangeCurrencies {
	var tickerData []EnabledExchangeCurrencies

	for _, individualBot := range bot.exchanges {
		if individualBot == nil || !individualBot.IsEnabled() {
			continue
		}

		var individualExchange EnabledExchangeCurrencies
		exchangeName := individualBot.GetName()
		individualExchange.ExchangeName = exchangeName
		currencies := individualBot.GetEnabledCurrencies()
		for _, x := range currencies {
			pair := x
			assetTypes, err := exchange.GetExchangeAssetTypes(exchangeName)
			if err != nil {
				log.Errorf("failed to get %s exchange asset types. Error: %s",
					exchangeName, err)
				continue
			}
			var tickerPrice ticker.Price
			if len(assetTypes) > 1 {
				for y := range assetTypes {
					tickerPrice, err = individualBot.GetTickerPrice(pair,
						assetTypes[y])
				}
			} else {
				tickerPrice, err = individualBot.GetTickerPrice(pair,
					assetTypes[0])
			}

			if err != nil {
				log.Errorf("failed to get %s %s ticker. Error: %s",
					pair,
					exchangeName,
					err)
				continue
			}

			individualExchange.ExchangeValues = append(
				individualExchange.ExchangeValues, tickerPrice,
			)
		}
		tickerData = append(tickerData, individualExchange)
	}
	return tickerData
}

// RESTGetAllActiveTickers returns all active tickers
func RESTGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = GetAllActiveTickers()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// GetAllEnabledExchangeAccountInfo returns all the current enabled exchanges
func GetAllEnabledExchangeAccountInfo() AllEnabledExchangeAccounts {
	var response AllEnabledExchangeAccounts
	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			if !individualBot.GetAuthenticatedAPISupport(exchange.RestAuthentication) {
				log.Warnf("GetAllEnabledExchangeAccountInfo: Skippping %s due to disabled authenticated API support.", individualBot.GetName())
				continue
			}
			individualExchange, err := individualBot.GetAccountInfo()
			if err != nil {
				log.Errorf("Error encountered retrieving exchange account info for %s. Error %s",
					individualBot.GetName(), err)
				continue
			}
			response.Data = append(response.Data, individualExchange)
		}
	}
	return response
}

// RESTGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func RESTGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := GetAllEnabledExchangeAccountInfo()
	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}
