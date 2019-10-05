package kraken

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	krakenAPIURL           = "https://api.kraken.com"
	krakenAPIVersion       = "0"
	krakenServerTime       = "Time"
	krakenAssets           = "Assets"
	krakenAssetPairs       = "AssetPairs"
	krakenTicker           = "Ticker"
	krakenOHLC             = "OHLC"
	krakenDepth            = "Depth"
	krakenTrades           = "Trades"
	krakenSpread           = "Spread"
	krakenBalance          = "Balance"
	krakenTradeBalance     = "TradeBalance"
	krakenOpenOrders       = "OpenOrders"
	krakenClosedOrders     = "ClosedOrders"
	krakenQueryOrders      = "QueryOrders"
	krakenTradeHistory     = "TradesHistory"
	krakenQueryTrades      = "QueryTrades"
	krakenOpenPositions    = "OpenPositions"
	krakenLedgers          = "Ledgers"
	krakenQueryLedgers     = "QueryLedgers"
	krakenTradeVolume      = "TradeVolume"
	krakenOrderCancel      = "CancelOrder"
	krakenOrderPlace       = "AddOrder"
	krakenWithdrawInfo     = "WithdrawInfo"
	krakenWithdraw         = "Withdraw"
	krakenDepositMethods   = "DepositMethods"
	krakenDepositAddresses = "DepositAddresses"
	krakenWithdrawStatus   = "WithdrawStatus"
	krakenWithdrawCancel   = "WithdrawCancel"

	krakenAuthRate   = 0
	krakenUnauthRate = 0
)

// Kraken is the overarching type across the alphapoint package
type Kraken struct {
	exchange.Base
	WebsocketConn      *wshandler.WebsocketConnection
	CryptoFee, FiatFee float64
	wsRequestMtx       sync.Mutex
}

// SetDefaults sets current default settings
func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = false
	k.FiatFee = 0.35
	k.CryptoFee = 0.10
	k.Verbose = false
	k.RESTPollingDelay = 10
	k.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithSetup |
		exchange.WithdrawCryptoWith2FA |
		exchange.AutoWithdrawFiatWithSetup |
		exchange.WithdrawFiatWith2FA
	k.RequestCurrencyPairFormat.Delimiter = ""
	k.RequestCurrencyPairFormat.Uppercase = true
	k.RequestCurrencyPairFormat.Separator = ","
	k.ConfigCurrencyPairFormat.Delimiter = "-"
	k.ConfigCurrencyPairFormat.Uppercase = true
	k.AssetTypes = []string{ticker.Spot}
	k.SupportsAutoPairUpdating = true
	k.SupportsRESTTickerBatching = true
	k.Requester = request.New(k.Name,
		request.NewRateLimit(time.Second, krakenAuthRate),
		request.NewRateLimit(time.Second, krakenUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	k.APIUrlDefault = krakenAPIURL
	k.APIUrl = k.APIUrlDefault
	k.Websocket = wshandler.New()
	k.WebsocketURL = krakenWSURL
	k.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketMessageCorrelationSupported
	k.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	k.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout

}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		k.SetEnabled(false)
	} else {
		k.Enabled = true
		k.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		k.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		k.SetHTTPClientTimeout(exch.HTTPTimeout)
		k.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		k.RESTPollingDelay = exch.RESTPollingDelay
		k.Verbose = exch.Verbose
		k.HTTPDebugging = exch.HTTPDebugging
		k.Websocket.SetWsStatusAndConnection(exch.Websocket)
		k.BaseCurrencies = exch.BaseCurrencies
		k.AvailablePairs = exch.AvailablePairs
		k.EnabledPairs = exch.EnabledPairs
		err := k.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = k.Websocket.Setup(k.WsConnect,
			k.Subscribe,
			k.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			krakenWSURL,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		k.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         k.Name,
			URL:                  k.Websocket.GetWebsocketURL(),
			ProxyURL:             k.Websocket.GetProxyAddress(),
			Verbose:              k.Verbose,
			RateLimit:            krakenWsRateLimit,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
	}
}

// GetServerTime returns current server time
func (k *Kraken) GetServerTime() (TimeResponse, error) {
	path := fmt.Sprintf("%s/%s/public/%s", k.APIUrl, krakenAPIVersion, krakenServerTime)

	var response struct {
		Error  []string     `json:"error"`
		Result TimeResponse `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets() (map[string]Asset, error) {
	path := fmt.Sprintf("%s/%s/public/%s", k.APIUrl, krakenAPIVersion, krakenAssets)

	var response struct {
		Error  []string         `json:"error"`
		Result map[string]Asset `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetAssetPairs returns a full asset pair list
func (k *Kraken) GetAssetPairs() (map[string]AssetPairs, error) {
	path := fmt.Sprintf("%s/%s/public/%s", k.APIUrl, krakenAPIVersion, krakenAssetPairs)

	var response struct {
		Error  []string              `json:"error"`
		Result map[string]AssetPairs `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTicker returns ticker information from kraken
func (k *Kraken) GetTicker(symbol string) (Ticker, error) {
	tick := Ticker{}
	values := url.Values{}
	values.Set("pair", symbol)

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", k.APIUrl, krakenAPIVersion, krakenTicker, values.Encode())

	err := k.SendHTTPRequest(path, &resp)
	if err != nil {
		return tick, err
	}

	if len(resp.Error) > 0 {
		return tick, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}

	for i := range resp.Data {
		tick.Ask, _ = strconv.ParseFloat(resp.Data[i].Ask[0], 64)
		tick.Bid, _ = strconv.ParseFloat(resp.Data[i].Bid[0], 64)
		tick.Last, _ = strconv.ParseFloat(resp.Data[i].Last[0], 64)
		tick.Volume, _ = strconv.ParseFloat(resp.Data[i].Volume[1], 64)
		tick.VWAP, _ = strconv.ParseFloat(resp.Data[i].VWAP[1], 64)
		tick.Trades = resp.Data[i].Trades[1]
		tick.Low, _ = strconv.ParseFloat(resp.Data[i].Low[1], 64)
		tick.High, _ = strconv.ParseFloat(resp.Data[i].High[1], 64)
		tick.Open, _ = strconv.ParseFloat(resp.Data[i].Open, 64)
	}
	return tick, nil
}

// GetTickers supports fetching multiple tickers from Kraken
// pairList must be in the format pairs separated by commas
// ("LTCUSD,ETCUSD")
func (k *Kraken) GetTickers(pairList string) (Tickers, error) {
	values := url.Values{}
	values.Set("pair", pairList)

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenTicker, values.Encode())

	err := k.SendHTTPRequest(path, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Error) > 0 {
		return nil, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}

	tickers := make(Tickers)

	for i := range resp.Data {
		tick := Ticker{}
		tick.Ask, _ = strconv.ParseFloat(resp.Data[i].Ask[0], 64)
		tick.Bid, _ = strconv.ParseFloat(resp.Data[i].Bid[0], 64)
		tick.Last, _ = strconv.ParseFloat(resp.Data[i].Last[0], 64)
		tick.Volume, _ = strconv.ParseFloat(resp.Data[i].Volume[1], 64)
		tick.VWAP, _ = strconv.ParseFloat(resp.Data[i].VWAP[1], 64)
		tick.Trades = resp.Data[i].Trades[1]
		tick.Low, _ = strconv.ParseFloat(resp.Data[i].Low[1], 64)
		tick.High, _ = strconv.ParseFloat(resp.Data[i].High[1], 64)
		tick.Open, _ = strconv.ParseFloat(resp.Data[i].Open, 64)
		tickers[i] = tick
	}
	return tickers, nil
}

// GetOHLC returns an array of open high low close values of a currency pair
func (k *Kraken) GetOHLC(symbol string) ([]OpenHighLowClose, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	type Response struct {
		Error []interface{}          `json:"error"`
		Data  map[string]interface{} `json:"result"`
	}

	var OHLC []OpenHighLowClose
	var result Response

	path := fmt.Sprintf("%s/%s/public/%s?%s", k.APIUrl, krakenAPIVersion, krakenOHLC, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return OHLC, err
	}

	if len(result.Error) != 0 {
		return OHLC, fmt.Errorf("getOHLC error: %s", result.Error)
	}

	for _, y := range result.Data[symbol].([]interface{}) {
		o := OpenHighLowClose{}
		for i, x := range y.([]interface{}) {
			switch i {
			case 0:
				o.Time = x.(float64)
			case 1:
				o.Open, _ = strconv.ParseFloat(x.(string), 64)
			case 2:
				o.High, _ = strconv.ParseFloat(x.(string), 64)
			case 3:
				o.Low, _ = strconv.ParseFloat(x.(string), 64)
			case 4:
				o.Close, _ = strconv.ParseFloat(x.(string), 64)
			case 5:
				o.Vwap, _ = strconv.ParseFloat(x.(string), 64)
			case 6:
				o.Volume, _ = strconv.ParseFloat(x.(string), 64)
			case 7:
				o.Count = x.(float64)
			}
		}
		OHLC = append(OHLC, o)
	}
	return OHLC, nil
}

// GetDepth returns the orderbook for a particular currency
func (k *Kraken) GetDepth(symbol string) (Orderbook, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	var orderBook Orderbook

	path := fmt.Sprintf("%s/%s/public/%s?%s", k.APIUrl, krakenAPIVersion, krakenDepth, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return orderBook, err
	}

	data := result.(map[string]interface{})
	orderbookData := data["result"].(map[string]interface{})

	var bidsData []interface{}
	var asksData []interface{}
	for _, y := range orderbookData {
		yData := y.(map[string]interface{})
		bidsData = yData["bids"].([]interface{})
		asksData = yData["asks"].([]interface{})
	}

	processOrderbook := func(data []interface{}) ([]OrderbookBase, error) {
		var result []OrderbookBase
		for x := range data {
			entry := data[x].([]interface{})

			price, priceErr := strconv.ParseFloat(entry[0].(string), 64)
			if priceErr != nil {
				return nil, priceErr
			}

			amount, amountErr := strconv.ParseFloat(entry[1].(string), 64)
			if amountErr != nil {
				return nil, amountErr
			}

			result = append(result, OrderbookBase{Price: price, Amount: amount})
		}
		return result, nil
	}

	orderBook.Bids, err = processOrderbook(bidsData)
	if err != nil {
		return orderBook, err
	}

	orderBook.Asks, err = processOrderbook(asksData)
	return orderBook, err
}

// GetTrades returns current trades on Kraken
func (k *Kraken) GetTrades(symbol string) ([]RecentTrades, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var recentTrades []RecentTrades
	var result interface{}

	path := fmt.Sprintf("%s/%s/public/%s?%s", k.APIUrl, krakenAPIVersion, krakenTrades, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return recentTrades, err
	}

	data := result.(map[string]interface{})
	tradeInfo := data["result"].(map[string]interface{})

	for _, x := range tradeInfo[symbol].([]interface{}) {
		r := RecentTrades{}
		for i, y := range x.([]interface{}) {
			switch i {
			case 0:
				r.Price, _ = strconv.ParseFloat(y.(string), 64)
			case 1:
				r.Volume, _ = strconv.ParseFloat(y.(string), 64)
			case 2:
				r.Time = y.(float64)
			case 3:
				r.BuyOrSell = y.(string)
			case 4:
				r.MarketOrLimit = y.(string)
			case 5:
				r.Miscellaneous = y.(string)
			}
		}
		recentTrades = append(recentTrades, r)
	}
	return recentTrades, nil
}

// GetSpread returns the full spread on Kraken
func (k *Kraken) GetSpread(symbol string) ([]Spread, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var peanutButter []Spread
	var response interface{}

	path := fmt.Sprintf("%s/%s/public/%s?%s", k.APIUrl, krakenAPIVersion, krakenSpread, values.Encode())

	err := k.SendHTTPRequest(path, &response)
	if err != nil {
		return peanutButter, err
	}

	data := response.(map[string]interface{})
	result := data["result"].(map[string]interface{})

	for _, x := range result[symbol].([]interface{}) {
		s := Spread{}
		for i, y := range x.([]interface{}) {
			switch i {
			case 0:
				s.Time = y.(float64)
			case 1:
				s.Bid, _ = strconv.ParseFloat(y.(string), 64)
			case 2:
				s.Ask, _ = strconv.ParseFloat(y.(string), 64)
			}
		}
		peanutButter = append(peanutButter, s)
	}
	return peanutButter, nil
}

// GetBalance returns your balance associated with your keys
func (k *Kraken) GetBalance() (map[string]float64, error) {
	var response struct {
		Error  []string          `json:"error"`
		Result map[string]string `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenBalance, url.Values{}, &response); err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for curency, balance := range response.Result {
		var err error
		if result[curency], err = strconv.ParseFloat(balance, 64); err != nil {
			return nil, err
		}
	}

	return result, GetError(response.Error)
}

// GetWithdrawInfo gets withdrawal fees
func (k *Kraken) GetWithdrawInfo(currency string, amount float64) (WithdrawInformation, error) {
	var response struct {
		Error  []string            `json:"error"`
		Result WithdrawInformation `json:"result"`
	}
	params := url.Values{}
	params.Set("asset ", currency)
	params.Set("key  ", "")
	params.Set("amount ", fmt.Sprintf("%f", amount))

	if err := k.SendAuthenticatedHTTPRequest(krakenWithdrawInfo, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// Withdraw withdraws funds
func (k *Kraken) Withdraw(asset, key string, amount float64) (string, error) {
	var response struct {
		Error       []string `json:"error"`
		ReferenceID string   `json:"refid"`
	}
	params := url.Values{}
	params.Set("asset", asset)
	params.Set("key", key)
	params.Set("amount", fmt.Sprintf("%f", amount))

	if err := k.SendAuthenticatedHTTPRequest(krakenWithdraw, params, &response); err != nil {
		return response.ReferenceID, err
	}

	return response.ReferenceID, GetError(response.Error)
}

// GetDepositMethods gets withdrawal fees
func (k *Kraken) GetDepositMethods(currency string) ([]DepositMethods, error) {
	var response struct {
		Error  []string         `json:"error"`
		Result []DepositMethods `json:"result"`
	}
	params := url.Values{}
	params.Set("asset", currency)

	err := k.SendAuthenticatedHTTPRequest(krakenDepositMethods, params, &response)
	if err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(args ...TradeBalanceOptions) (TradeBalanceInfo, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Aclass) > 0 {
			params.Set("aclass", args[0].Aclass)
		}

		if len(args[0].Asset) > 0 {
			params.Set("asset", args[0].Asset)
		}

	}

	var response struct {
		Error  []string         `json:"error"`
		Result TradeBalanceInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeBalance, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(args OrderInfoOptions) (OpenOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	var response struct {
		Error  []string   `json:"error"`
		Result OpenOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOpenOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(args GetClosedOrdersOptions) (ClosedOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	if len(args.Start) > 0 {
		params.Set("start", args.Start)
	}

	if len(args.End) > 0 {
		params.Set("end", args.End)
	}

	if args.Ofs > 0 {
		params.Set("ofs", strconv.FormatInt(args.Ofs, 10))
	}

	if len(args.CloseTime) > 0 {
		params.Set("closetime", args.CloseTime)
	}

	var response struct {
		Error  []string     `json:"error"`
		Result ClosedOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenClosedOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryOrdersInfo returns order information
func (k *Kraken) QueryOrdersInfo(args OrderInfoOptions, txid string, txids ...string) (map[string]OrderInfo, error) {
	params := url.Values{
		"txid": {txid},
	}

	if txids != nil {
		params.Set("txid", txid+","+strings.Join(txids, ","))
	}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	var response struct {
		Error  []string             `json:"error"`
		Result map[string]OrderInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(args ...GetTradesHistoryOptions) (TradesHistory, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Type) > 0 {
			params.Set("type", args[0].Type)
		}

		if args[0].Trades {
			params.Set("trades", "true")
		}

		if len(args[0].Start) > 0 {
			params.Set("start", args[0].Start)
		}

		if len(args[0].End) > 0 {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs > 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}
	}

	var response struct {
		Error  []string      `json:"error"`
		Result TradesHistory `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeHistory, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryTrades returns information on a specific trade
func (k *Kraken) QueryTrades(trades bool, txid string, txids ...string) (map[string]TradeInfo, error) {
	params := url.Values{
		"txid": {txid},
	}

	if trades {
		params.Set("trades", "true")
	}

	if txids != nil {
		params.Set("txid", txid+","+strings.Join(txids, ","))
	}

	var response struct {
		Error  []string             `json:"error"`
		Result map[string]TradeInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryTrades, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// OpenPositions returns current open positions
func (k *Kraken) OpenPositions(docalcs bool, txids ...string) (map[string]Position, error) {
	params := url.Values{}

	if txids != nil {
		params.Set("txid", strings.Join(txids, ","))
	}

	if docalcs {
		params.Set("docalcs", "true")
	}

	var response struct {
		Error  []string            `json:"error"`
		Result map[string]Position `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOpenPositions, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(args ...GetLedgersOptions) (Ledgers, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Aclass == "" {
			params.Set("aclass", args[0].Aclass)
		}

		if args[0].Asset == "" {
			params.Set("asset", args[0].Asset)
		}

		if args[0].Type == "" {
			params.Set("type", args[0].Type)
		}

		if args[0].Start == "" {
			params.Set("start", args[0].Start)
		}

		if args[0].End == "" {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs != 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}
	}

	var response struct {
		Error  []string `json:"error"`
		Result Ledgers  `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryLedgers queries an individual ledger by ID
func (k *Kraken) QueryLedgers(id string, ids ...string) (map[string]LedgerInfo, error) {
	params := url.Values{
		"id": {id},
	}

	if ids != nil {
		params.Set("id", id+","+strings.Join(ids, ","))
	}

	var response struct {
		Error  []string              `json:"error"`
		Result map[string]LedgerInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(feeinfo bool, symbol ...string) (TradeVolumeResponse, error) {
	params := url.Values{}

	if symbol != nil {
		params.Set("pair", strings.Join(symbol, ","))
	}

	if feeinfo {
		params.Set("fee-info", "true")
	}

	var response struct {
		Error  []string            `json:"error"`
		Result TradeVolumeResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeVolume, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(symbol, side, orderType string, volume, price, price2, leverage float64, args *AddOrderOptions) (AddOrderResponse, error) {
	params := url.Values{
		"pair":      {symbol},
		"type":      {common.StringToLower(side)},
		"ordertype": {common.StringToLower(orderType)},
		"volume":    {strconv.FormatFloat(volume, 'f', -1, 64)},
	}

	if orderType == "limit" || price > 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}

	if price2 != 0 {
		params.Set("price2", strconv.FormatFloat(price2, 'f', -1, 64))
	}

	if leverage != 0 {
		params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	}

	if args.Oflags == "" {
		params.Set("oflags", args.Oflags)
	}

	if args.StartTm == "" {
		params.Set("starttm", args.StartTm)
	}

	if args.ExpireTm == "" {
		params.Set("expiretm", args.ExpireTm)
	}

	if args.CloseOrderType != "" {
		params.Set("close[ordertype]", args.ExpireTm)
	}

	if args.ClosePrice != 0 {
		params.Set("close[price]", strconv.FormatFloat(args.ClosePrice, 'f', -1, 64))
	}

	if args.ClosePrice2 != 0 {
		params.Set("close[price2]", strconv.FormatFloat(args.ClosePrice2, 'f', -1, 64))
	}

	if args.Validate {
		params.Set("validate", "true")
	}

	var response struct {
		Error  []string         `json:"error"`
		Result AddOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOrderPlace, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// CancelExistingOrder cancels order by orderID
func (k *Kraken) CancelExistingOrder(txid string) (CancelOrderResponse, error) {
	values := url.Values{
		"txid": {txid},
	}

	var response struct {
		Error  []string            `json:"error"`
		Result CancelOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOrderCancel, values, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetError parse Exchange errors in response and return the first one
// Error format from API doc:
//   error = array of error messages in the format of:
//       <char-severity code><string-error category>:<string-error type>[:<string-extra info>]
//       severity code can be E for error or W for warning
func GetError(apiErrors []string) error {
	const exchangeName = "Kraken"
	for _, e := range apiErrors {
		switch e[0] {
		case 'W':
			log.Warnf("%s API warning: %v\n", exchangeName, e[1:])
		default:
			return fmt.Errorf("%s API error: %v", exchangeName, e[1:])
		}
	}

	return nil
}

// SendHTTPRequest sends an unauthenticated HTTP requests
func (k *Kraken) SendHTTPRequest(path string, result interface{}) error {
	return k.SendPayload(http.MethodGet, path, nil, nil, result, false, false, k.Verbose, k.HTTPDebugging)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(method string, params url.Values, result interface{}) (err error) {
	if !k.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			k.Name)
	}

	path := fmt.Sprintf("/%s/private/%s", krakenAPIVersion, method)

	n := k.Requester.GetNonce(true).String()
	params.Set("nonce", n)

	secret, err := common.Base64Decode(k.APISecret)
	if err != nil {
		return err
	}

	encoded := params.Encode()
	shasum := common.GetSHA256([]byte(params.Get("nonce") + encoded))
	signature := common.Base64Encode(common.GetHMAC(common.HashSHA512,
		append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Debugf("Sending POST request to %s, path: %s, params: %s",
			k.APIUrl,
			path,
			encoded)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.APIKey
	headers["API-Sign"] = signature

	return k.SendPayload(http.MethodPost,
		k.APIUrl+path,
		headers,
		strings.NewReader(encoded),
		result,
		true,
		true,
		k.Verbose,
		k.HTTPDebugging)
}

// GetFee returns an estimate of fee based on type of transaction
func (k *Kraken) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	c := feeBuilder.Pair.Base.String() +
		feeBuilder.Pair.Delimiter +
		feeBuilder.Pair.Quote.String()

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePair, err := k.GetTradeVolume(true, c)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = calculateTradingFee(c,
				feePair.FeesMaker,
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		} else {
			fee = calculateTradingFee(c,
				feePair.Fees,
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		}
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankDepositFee:
		depositMethods, err := k.GetDepositMethods(feeBuilder.FiatCurrency.String())
		if err != nil {
			return 0, err
		}

		for _, i := range depositMethods {
			if feeBuilder.BankTransactionType == exchange.WireTransfer {
				if i.Method == "SynapsePay (US Wire)" {
					fee = i.Fee
					return fee, nil
				}
			}
		}
	case exchange.CyptocurrencyDepositFee:
		fee = getCryptocurrencyDepositFee(feeBuilder.Pair.Base)

	case exchange.InternationalBankWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FiatCurrency)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0016 * price * amount
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getCryptocurrencyDepositFee(c currency.Code) float64 {
	return DepositFees[c]
}

func calculateTradingFee(currency string, feePair map[string]TradeVolumeFee, purchasePrice, amount float64) float64 {
	return (feePair[currency].Fee / 100) * purchasePrice * amount
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (k *Kraken) GetCryptoDepositAddress(method, code string) (string, error) {
	var resp = struct {
		Error  []string         `json:"error"`
		Result []DepositAddress `json:"result"`
	}{}

	values := url.Values{}
	values.Set("asset", code)
	values.Set("method", method)

	err := k.SendAuthenticatedHTTPRequest(krakenDepositAddresses, values, &resp)
	if err != nil {
		return "", err
	}

	for _, a := range resp.Result {
		return a.Address, nil
	}

	return "", errors.New("no addresses returned")
}

// WithdrawStatus gets the status of recent withdrawals
func (k *Kraken) WithdrawStatus(c currency.Code, method string) ([]WithdrawStatusResponse, error) {
	var response struct {
		Error  []string                 `json:"error"`
		Result []WithdrawStatusResponse `json:"result"`
	}

	params := url.Values{}
	params.Set("asset ", c.String())
	if method != "" {
		params.Set("method", method)
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenWithdrawStatus, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// WithdrawCancel sends a withdrawal cancelation request
func (k *Kraken) WithdrawCancel(c currency.Code, refID string) (bool, error) {
	var response struct {
		Error  []string `json:"error"`
		Result bool     `json:"result"`
	}

	params := url.Values{}
	params.Set("asset ", c.String())
	params.Set("refid", refID)

	if err := k.SendAuthenticatedHTTPRequest(krakenWithdrawCancel, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}
