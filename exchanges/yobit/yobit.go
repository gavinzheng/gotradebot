package yobit

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	apiPublicURL                  = "https://yobit.net/api"
	apiPrivateURL                 = "https://yobit.net/tapi"
	apiPublicVersion              = "3"
	publicInfo                    = "info"
	publicTicker                  = "ticker"
	publicDepth                   = "depth"
	publicTrades                  = "trades"
	privateAccountInfo            = "getInfo"
	privateTrade                  = "Trade"
	privateActiveOrders           = "ActiveOrders"
	privateOrderInfo              = "OrderInfo"
	privateCancelOrder            = "CancelOrder"
	privateTradeHistory           = "TradeHistory"
	privateGetDepositAddress      = "GetDepositAddress"
	privateWithdrawCoinsToAddress = "WithdrawCoinsToAddress"
	privateCreateCoupon           = "CreateYobicode"
	privateRedeemCoupon           = "RedeemYobicode"

	yobitAuthRate   = 0
	yobitUnauthRate = 0
)

// Yobit is the overarching type across the Yobit package
type Yobit struct {
	exchange.Base
	Ticker map[string]Ticker
}

// SetDefaults sets current default value for Yobit
func (y *Yobit) SetDefaults() {
	y.Name = "Yobit"
	y.Enabled = true
	y.Fee = 0.2
	y.Verbose = false
	y.RESTPollingDelay = 10
	y.AuthenticatedAPISupport = true
	y.Ticker = make(map[string]Ticker)
	y.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.WithdrawFiatViaWebsiteOnly
	y.RequestCurrencyPairFormat.Delimiter = "_"
	y.RequestCurrencyPairFormat.Uppercase = false
	y.RequestCurrencyPairFormat.Separator = "-"
	y.ConfigCurrencyPairFormat.Delimiter = "_"
	y.ConfigCurrencyPairFormat.Uppercase = true
	y.AssetTypes = []string{ticker.Spot}
	y.SupportsAutoPairUpdating = false
	y.SupportsRESTTickerBatching = true
	y.Requester = request.New(y.Name,
		request.NewRateLimit(time.Second, yobitAuthRate),
		request.NewRateLimit(time.Second, yobitUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	y.APIUrlDefault = apiPublicURL
	y.APIUrl = y.APIUrlDefault
	y.APIUrlSecondaryDefault = apiPrivateURL
	y.APIUrlSecondary = y.APIUrlSecondaryDefault
	y.Websocket = wshandler.New()
}

// Setup sets exchange configuration parameters for Yobit
func (y *Yobit) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		y.SetEnabled(false)
	} else {
		y.Enabled = true
		y.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		y.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		y.RESTPollingDelay = exch.RESTPollingDelay
		y.Verbose = exch.Verbose
		y.HTTPDebugging = exch.HTTPDebugging
		y.Websocket.SetWsStatusAndConnection(exch.Websocket)
		y.BaseCurrencies = exch.BaseCurrencies
		y.AvailablePairs = exch.AvailablePairs
		y.EnabledPairs = exch.EnabledPairs
		y.SetHTTPClientTimeout(exch.HTTPTimeout)
		y.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		err := y.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetInfo returns the Yobit info
func (y *Yobit) GetInfo() (Info, error) {
	resp := Info{}
	path := fmt.Sprintf("%s/%s/%s/", y.APIUrl, apiPublicVersion, publicInfo)

	return resp, y.SendHTTPRequest(path, &resp)
}

// GetTicker returns a ticker for a specific currency
func (y *Yobit) GetTicker(symbol string) (map[string]Ticker, error) {
	type Response struct {
		Data map[string]Ticker
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", y.APIUrl, apiPublicVersion, publicTicker, symbol)

	return response.Data, y.SendHTTPRequest(path, &response.Data)
}

// GetDepth returns the depth for a specific currency
func (y *Yobit) GetDepth(symbol string) (Orderbook, error) {
	type Response struct {
		Data map[string]Orderbook
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", y.APIUrl, apiPublicVersion, publicDepth, symbol)

	return response.Data[symbol],
		y.SendHTTPRequest(path, &response.Data)
}

// GetTrades returns the trades for a specific currency
func (y *Yobit) GetTrades(symbol string) ([]Trades, error) {
	type Response struct {
		Data map[string][]Trades
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", y.APIUrl, apiPublicVersion, publicTrades, symbol)

	return response.Data[symbol], y.SendHTTPRequest(path, &response.Data)
}

// GetAccountInformation returns a users account info
func (y *Yobit) GetAccountInformation() (AccountInfo, error) {
	result := AccountInfo{}

	err := y.SendAuthenticatedHTTPRequest(privateAccountInfo, url.Values{}, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// Trade places an order and returns the order ID if successful or an error
func (y *Yobit) Trade(pair, orderType string, amount, price float64) (int64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	result := Trade{}

	err := y.SendAuthenticatedHTTPRequest(privateTrade, req, &result)
	if err != nil {
		return int64(result.OrderID), err
	}
	if result.Error != "" {
		return int64(result.OrderID), errors.New(result.Error)
	}
	return int64(result.OrderID), nil
}

// GetOpenOrders returns the active orders for a specific currency
func (y *Yobit) GetOpenOrders(pair string) (map[string]ActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	result := map[string]ActiveOrders{}

	return result, y.SendAuthenticatedHTTPRequest(privateActiveOrders, req, &result)
}

// GetOrderInformation returns the order info for a specific order ID
func (y *Yobit) GetOrderInformation(orderID int64) (map[string]OrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(orderID, 10))

	result := map[string]OrderInfo{}

	return result, y.SendAuthenticatedHTTPRequest(privateOrderInfo, req, &result)
}

// CancelExistingOrder cancels an order for a specific order ID
func (y *Yobit) CancelExistingOrder(orderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(orderID, 10))

	result := CancelOrder{}

	err := y.SendAuthenticatedHTTPRequest(privateCancelOrder, req, &result)
	if err != nil {
		return false, err
	}
	if result.Error != "" {
		return false, errors.New(result.Error)
	}
	return true, nil
}

// GetTradeHistory returns the trade history
func (y *Yobit) GetTradeHistory(tidFrom, count, tidEnd, since, end int64, order, pair string) (map[string]TradeHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(tidFrom, 10))
	req.Add("count", strconv.FormatInt(count, 10))
	req.Add("from_id", strconv.FormatInt(tidFrom, 10))
	req.Add("end_id", strconv.FormatInt(tidEnd, 10))
	req.Add("order", order)
	req.Add("since", strconv.FormatInt(since, 10))
	req.Add("end", strconv.FormatInt(end, 10))
	req.Add("pair", pair)

	result := TradeHistoryResponse{}

	err := y.SendAuthenticatedHTTPRequest(privateTradeHistory, req, &result)
	if err != nil {
		return nil, err
	}
	if result.Success == 0 {
		return nil, errors.New(result.Error)
	}

	return result.Data, nil
}

// GetCryptoDepositAddress returns the deposit address for a specific currency
func (y *Yobit) GetCryptoDepositAddress(coin string) (DepositAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)

	result := DepositAddress{}

	err := y.SendAuthenticatedHTTPRequest(privateGetDepositAddress, req, &result)
	if err != nil {
		return result, err
	}
	if result.Success != 1 {
		return result, fmt.Errorf("%s", result.Error)
	}
	return result, nil
}

// WithdrawCoinsToAddress initiates a withdrawal to a specified address
func (y *Yobit) WithdrawCoinsToAddress(coin string, amount float64, address string) (WithdrawCoinsToAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	result := WithdrawCoinsToAddress{}

	err := y.SendAuthenticatedHTTPRequest(privateWithdrawCoinsToAddress, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// CreateCoupon creates an exchange coupon for a sepcific currency
func (y *Yobit) CreateCoupon(currency string, amount float64) (CreateCoupon, error) {
	req := url.Values{}
	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result CreateCoupon

	err := y.SendAuthenticatedHTTPRequest(privateCreateCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// RedeemCoupon redeems an exchange coupon
func (y *Yobit) RedeemCoupon(coupon string) (RedeemCoupon, error) {
	req := url.Values{}
	req.Add("coupon", coupon)

	result := RedeemCoupon{}

	err := y.SendAuthenticatedHTTPRequest(privateRedeemCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (y *Yobit) SendHTTPRequest(path string, result interface{}) error {
	return y.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		y.Verbose,
		y.HTTPDebugging)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to Yobit
func (y *Yobit) SendAuthenticatedHTTPRequest(path string, params url.Values, result interface{}) (err error) {
	if !y.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			y.Name)
	}

	if params == nil {
		params = url.Values{}
	}

	n := y.Requester.GetNonce(false).String()

	params.Set("nonce", n)
	params.Set("method", path)

	encoded := params.Encode()
	hmac := common.GetHMAC(common.HashSHA512,
		[]byte(encoded),
		[]byte(y.APISecret))

	if y.Verbose {
		log.Debugf("Sending POST request to %s calling path %s with params %s\n",
			apiPrivateURL,
			path,
			encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = y.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	return y.SendPayload(http.MethodPost,
		apiPrivateURL,
		headers,
		strings.NewReader(encoded),
		result,
		true,
		true,
		y.Verbose,
		y.HTTPDebugging)
}

// GetFee returns an estimate of fee based on type of transaction
func (y *Yobit) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency,
			feeBuilder.BankTransactionType)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount,
			feeBuilder.BankTransactionType)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(price, amount float64) (fee float64) {
	return 0.002 * price * amount
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64, bankTransactionType exchange.InternationalBankTransactionType) float64 {
	var fee float64

	switch bankTransactionType {
	case exchange.PerfectMoney:
		if c == currency.USD {
			fee = 0.02 * amount
		}
	case exchange.Payeer:
		switch c {
		case currency.USD:
			fee = 0.03 * amount
		case currency.RUR:
			fee = 0.006 * amount
		}
	case exchange.AdvCash:
		switch c {
		case currency.USD:
			fee = 0.04 * amount
		case currency.RUR:
			fee = 0.03 * amount
		}
	case exchange.Qiwi:
		if c == currency.RUR {
			fee = 0.04 * amount
		}
	case exchange.Capitalist:
		if c == currency.USD {
			fee = 0.06 * amount
		}
	}

	return fee
}

// getInternationalBankDepositFee; No real fees for yobit deposits, but want to be explicit on what each payment type supports
func getInternationalBankDepositFee(c currency.Code, bankTransactionType exchange.InternationalBankTransactionType) float64 {
	var fee float64
	switch bankTransactionType {
	case exchange.PerfectMoney:
		if c == currency.USD {
			fee = 0
		}
	case exchange.Payeer:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	case exchange.AdvCash:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	case exchange.Qiwi:
		if c == currency.RUR {
			fee = 0
		}
	case exchange.Capitalist:
		switch c {
		case currency.USD:
			fee = 0
		case currency.RUR:
			fee = 0
		}
	}

	return fee
}
