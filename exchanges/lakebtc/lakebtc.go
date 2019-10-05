package lakebtc

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	lakeBTCAPIURL              = "https://api.lakebtc.com/api_v2"
	lakeBTCAPIVersion          = "2"
	lakeBTCTicker              = "ticker"
	lakeBTCOrderbook           = "bcorderbook"
	lakeBTCTrades              = "bctrades"
	lakeBTCGetAccountInfo      = "getAccountInfo"
	lakeBTCBuyOrder            = "buyOrder"
	lakeBTCSellOrder           = "sellOrder"
	lakeBTCOpenOrders          = "openOrders"
	lakeBTCGetOrders           = "getOrders"
	lakeBTCCancelOrder         = "cancelOrders"
	lakeBTCGetTrades           = "getTrades"
	lakeBTCGetExternalAccounts = "getExternalAccounts"
	lakeBTCCreateWithdraw      = "createWithdraw"

	lakeBTCAuthRate = 0
	lakeBTCUnauth   = 0
)

// LakeBTC is the overarching type across the LakeBTC package
type LakeBTC struct {
	exchange.Base
}

// SetDefaults sets LakeBTC defaults
func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = false
	l.TakerFee = 0.2
	l.MakerFee = 0.15
	l.Verbose = false
	l.RESTPollingDelay = 10
	l.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.WithdrawFiatViaWebsiteOnly
	l.RequestCurrencyPairFormat.Delimiter = ""
	l.RequestCurrencyPairFormat.Uppercase = true
	l.ConfigCurrencyPairFormat.Delimiter = ""
	l.ConfigCurrencyPairFormat.Uppercase = true
	l.AssetTypes = []string{ticker.Spot}
	l.SupportsAutoPairUpdating = true
	l.SupportsRESTTickerBatching = true
	l.Requester = request.New(l.Name,
		request.NewRateLimit(time.Second, lakeBTCAuthRate),
		request.NewRateLimit(time.Second, lakeBTCUnauth),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	l.APIUrlDefault = lakeBTCAPIURL
	l.APIUrl = l.APIUrlDefault
	l.Websocket = wshandler.New()
}

// Setup sets exchange configuration profile
func (l *LakeBTC) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.SetHTTPClientTimeout(exch.HTTPTimeout)
		l.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.HTTPDebugging = exch.HTTPDebugging
		l.BaseCurrencies = exch.BaseCurrencies
		l.AvailablePairs = exch.AvailablePairs
		l.EnabledPairs = exch.EnabledPairs
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetTradablePairs returns a list of available pairs from the exchange
func (l *LakeBTC) GetTradablePairs() ([]string, error) {
	result, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result {
		currencies = append(currencies, common.StringToUpper(x))
	}

	return currencies, nil
}

// GetTicker returns the current ticker from lakeBTC
func (l *LakeBTC) GetTicker() (map[string]Ticker, error) {
	response := make(map[string]TickerResponse)
	path := fmt.Sprintf("%s/%s", l.APIUrl, lakeBTCTicker)

	if err := l.SendHTTPRequest(path, &response); err != nil {
		return nil, err
	}

	result := make(map[string]Ticker)

	for k, v := range response {
		var tick Ticker
		key := common.StringToUpper(k)
		if v.Ask != nil {
			tick.Ask, _ = strconv.ParseFloat(v.Ask.(string), 64)
		}
		if v.Bid != nil {
			tick.Bid, _ = strconv.ParseFloat(v.Bid.(string), 64)
		}
		if v.High != nil {
			tick.High, _ = strconv.ParseFloat(v.High.(string), 64)
		}
		if v.Last != nil {
			tick.Last, _ = strconv.ParseFloat(v.Last.(string), 64)
		}
		if v.Low != nil {
			tick.Low, _ = strconv.ParseFloat(v.Low.(string), 64)
		}
		if v.Volume != nil {
			tick.Volume, _ = strconv.ParseFloat(v.Volume.(string), 64)
		}
		result[key] = tick
	}
	return result, nil
}

// GetOrderBook returns the order book from LakeBTC
func (l *LakeBTC) GetOrderBook(currency string) (Orderbook, error) {
	type Response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("%s/%s?symbol=%s", l.APIUrl, lakeBTCOrderbook, common.StringToLower(currency))
	resp := Response{}
	err := l.SendHTTPRequest(path, &resp)
	if err != nil {
		return Orderbook{}, err
	}
	orderbook := Orderbook{}

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, OrderbookStructure{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Error(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Error(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, OrderbookStructure{price, amount})
	}
	return orderbook, nil
}

// GetTradeHistory returns the trade history for a given currency pair
func (l *LakeBTC) GetTradeHistory(currency string) ([]TradeHistory, error) {
	path := fmt.Sprintf("%s/%s?symbol=%s", l.APIUrl, lakeBTCTrades, common.StringToLower(currency))
	var resp []TradeHistory

	return resp, l.SendHTTPRequest(path, &resp)
}

// GetAccountInformation returns your current account information
func (l *LakeBTC) GetAccountInformation() (AccountInfo, error) {
	resp := AccountInfo{}

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetAccountInfo, "", &resp)
}

// Trade executes an order on the exchange and returns trade inforamtion or an
// error
func (l *LakeBTC) Trade(isBuyOrder bool, amount, price float64, currency string) (Trade, error) {
	resp := Trade{}
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency

	if isBuyOrder {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCBuyOrder, params, &resp); err != nil {
			return resp, err
		}
	} else {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCSellOrder, params, &resp); err != nil {
			return resp, err
		}
	}

	if resp.Result != "order received" {
		return resp, fmt.Errorf("unexpected result: %s", resp.Result)
	}

	return resp, nil
}

// GetOpenOrders returns all open orders associated with your account
func (l *LakeBTC) GetOpenOrders() ([]OpenOrders, error) {
	var orders []OpenOrders

	return orders, l.SendAuthenticatedHTTPRequest(lakeBTCOpenOrders, "", &orders)
}

// GetOrders returns your orders
func (l *LakeBTC) GetOrders(orders []int64) ([]Orders, error) {
	var ordersStr []string
	for _, x := range orders {
		ordersStr = append(ordersStr, strconv.FormatInt(x, 10))
	}

	var resp []Orders
	return resp,
		l.SendAuthenticatedHTTPRequest(lakeBTCGetOrders, common.JoinStrings(ordersStr, ","), &resp)
}

// CancelExistingOrder cancels an order by ID number and returns an error
func (l *LakeBTC) CancelExistingOrder(orderID int64) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(lakeBTCCancelOrder, params, &resp)
	if err != nil {
		return err
	}

	if !resp.Result {
		return errors.New("unable to cancel order")
	}
	return nil
}

// CancelExistingOrders cancels an order by ID number and returns an error
func (l *LakeBTC) CancelExistingOrders(orderIDs []string) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := common.JoinStrings(orderIDs, ",")
	err := l.SendAuthenticatedHTTPRequest(lakeBTCCancelOrder, params, &resp)
	if err != nil {
		return err
	}

	if !resp.Result {
		return fmt.Errorf("unable to cancel order(s): %v", orderIDs)
	}
	return nil
}

// GetTrades returns trades associated with your account by timestamp
func (l *LakeBTC) GetTrades(timestamp int64) ([]AuthenticatedTradeHistory, error) {
	params := ""
	if timestamp != 0 {
		params = strconv.FormatInt(timestamp, 10)
	}

	var trades []AuthenticatedTradeHistory
	return trades, l.SendAuthenticatedHTTPRequest(lakeBTCGetTrades, params, &trades)
}

// GetExternalAccounts returns your external accounts WARNING: Only for BTC!
func (l *LakeBTC) GetExternalAccounts() ([]ExternalAccounts, error) {
	var resp []ExternalAccounts

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetExternalAccounts, "", &resp)
}

// CreateWithdraw allows your to withdraw to external account WARNING: Only for
// BTC!
func (l *LakeBTC) CreateWithdraw(amount float64, accountID string) (Withdraw, error) {
	resp := Withdraw{}
	params := strconv.FormatFloat(amount, 'f', -1, 64) + ",btc," + accountID

	err := l.SendAuthenticatedHTTPRequest(lakeBTCCreateWithdraw, params, &resp)
	if err != nil {
		return Withdraw{}, err
	}
	if len(resp.Error) > 0 {
		return resp, errors.New(resp.Error)
	}

	return resp, nil
}

// SendHTTPRequest sends an unauthenticated http request
func (l *LakeBTC) SendHTTPRequest(path string, result interface{}) error {
	return l.SendPayload(http.MethodGet, path, nil, nil, result, false, false, l.Verbose, l.HTTPDebugging)
}

// SendAuthenticatedHTTPRequest sends an autheticated HTTP request to a LakeBTC
func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	n := l.Requester.GetNonce(true).String()

	req := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=1&method=%s&params=%s", n, l.APIKey, method, params)
	hmac := common.GetHMAC(common.HashSHA1, []byte(req), []byte(l.APISecret))

	if l.Verbose {
		log.Debugf("Sending POST request to %s calling method %s with params %s\n", l.APIUrl, method, req)
	}

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["id"] = 1
	postData["params"] = common.SplitStrings(params, ",")

	data, err := common.JSONEncode(postData)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = l.Nonce.String()
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(l.APIKey+":"+common.HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/json-rpc"

	return l.SendPayload(http.MethodPost, l.APIUrl, headers, strings.NewReader(string(data)), result, true, true, l.Verbose, l.HTTPDebugging)
}

// GetFee returns an estimate of fee based on type of transaction
func (l *LakeBTC) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.CyptocurrencyDepositFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		// fees for withdrawals are dynamic. They cannot be calculated in
		// advance as they are manually performed via the website, it can only
		// be determined when submitting the request
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
	return 0.002 * price * amount
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	if isMaker {
		// TODO: Volume based fee calculation
		fee = 0.0015
	} else {
		fee = 0.002
	}

	return fee * amount * purchasePrice
}

func getCryptocurrencyWithdrawalFee(c currency.Code) (fee float64) {
	if c == currency.BTC {
		fee = 0.001
	}
	return fee
}
