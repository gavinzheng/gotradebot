package zb

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var z ZB
var wsSetupRan bool

func TestSetDefaults(t *testing.T) {
	z.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	zbConfig, err := cfg.GetExchangeConfig("ZB")
	if err != nil {
		t.Error("Test Failed - ZB Setup() init error")
	}
	zbConfig.AuthenticatedAPISupport = true
	zbConfig.AuthenticatedWebsocketAPISupport = true
	zbConfig.APIKey = apiKey
	zbConfig.APISecret = apiSecret

	z.Setup(&zbConfig)
}

func setupWsAuth(t *testing.T) {
	if wsSetupRan {
		return
	}
	z.SetDefaults()
	TestSetup(t)
	if !z.Websocket.IsEnabled() && !z.AuthenticatedWebsocketAPISupport || !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	z.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         z.Name,
		URL:                  zbWebsocketAPI,
		Verbose:              z.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := z.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	z.Websocket.DataHandler = make(chan interface{}, 11)
	z.Websocket.TrafficAlert = make(chan struct{}, 11)
	go z.WsHandleData()
	wsSetupRan = true
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if z.APIKey == "" || z.APISecret == "" {
		t.Skip()
	}

	arg := SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Type:   SpotNewOrderRequestParamsTypeSell,
		Amount: 0.01,
		Price:  10246.1,
	}
	orderid, err := z.SpotNewOrder(arg)
	if err != nil {
		t.Errorf("Test failed - ZB SpotNewOrder: %s", err)
	} else {
		fmt.Println(orderid)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if z.APIKey == "" || z.APISecret == "" {
		t.Skip()
	}

	err := z.CancelExistingOrder(20180629145864850, "btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB CancelExistingOrder: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := z.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := z.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := z.GetTickers()
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := z.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := z.GetMarkets()
	if err != nil {
		t.Errorf("Test failed - ZB GetMarkets: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	arg := KlinesRequestParams{
		Symbol: "btc_usdt",
		Type:   TimeIntervalFiveMinutes,
		Size:   10,
	}
	_, err := z.GetSpotKline(arg)
	if err != nil {
		t.Errorf("Test failed - ZB GetSpotKline: %s", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	z.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0015), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := z.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	z.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText

	withdrawPermissions := z.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
	}

	_, err := z.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
		OrderSide: exchange.BuyOrderSide,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
	}

	_, err := z.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if z.APIKey != "" && z.APIKey != "Key" &&
		z.APISecret != "" && z.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip(fmt.Sprintf("ApiKey: %s. Can place orders: %v",
			z.APIKey,
			canManipulateRealOrders))
	}
	var pair = currency.Pair{
		Delimiter: "_",
		Base:      currency.QTUM,
		Quote:     currency.USDT,
	}

	response, err := z.SubmitOrder(pair,
		exchange.BuyOrderSide,
		exchange.MarketOrderType,
		1,
		10,
		"hi")

	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := z.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := z.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || apiSecret != "" {
		_, err := z.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	} else {
		_, err := z.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := z.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
		FeeAmount:   1,
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := z.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := z.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := z.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if apiKey != "" || apiSecret != "" {
		_, err := z.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error PLEASE MAKE SURE YOU CREATE DEPOSIT ADDRESSES VIA ZB.COM",
				err)
		}
	} else {
		_, err := z.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error")
		}
	}
}

// TestZBInvalidJSON ZB sends poorly formed JSON. this tests the JSON fixer
// Then JSON decode it to test if successful
func TestZBInvalidJSON(t *testing.T) {
	json := `{"success":true,"code":1000,"channel":"getSubUserList","message":"[{"isOpenApi":false,"memo":"Memo","userName":"hello@imgoodthanksandyou.com@good","userId":1337,"isFreez":false}]","no":"0"}`
	fixedJSON := z.wsFixInvalidJSON([]byte(json))
	var response WsGetSubUserListResponse
	err := common.JSONDecode(fixedJSON, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Message[0].UserID != 1337 {
		t.Fatal("Expected extracted JSON USERID to equal 1337")
	}

	json = `{"success":true,"code":1000,"channel":"createSubUserKey","message":"{"apiKey":"thisisnotareallykeyyousillybilly","apiSecret":"lol"}","no":"123"}`
	fixedJSON = z.wsFixInvalidJSON([]byte(json))
	var response2 WsRequestResponse
	err = common.JSONDecode(fixedJSON, &response2)
	if err != nil {
		t.Error(err)
	}
}

// TestWsTransferFunds ws test
func TestWsTransferFunds(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsDoTransferFunds(currency.BTC,
		0.0001,
		"username1",
		"username2",
	)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsCreateSuUserKey ws test
func TestWsCreateSuUserKey(t *testing.T) {
	setupWsAuth(t)
	subUsers, err := z.wsGetSubUserList()
	if err != nil {
		t.Fatal(err)
	}
	userID := subUsers.Message[0].UserID
	_, err = z.wsCreateSubUserKey(true, true, true, true, "subu", fmt.Sprintf("%v", userID))
	if err != nil {
		t.Fatal(err)
	}
}

// TestGetSubUserList ws test
func TestGetSubUserList(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsGetSubUserList()
	if err != nil {
		t.Fatal(err)
	}
}

// TestAddSubUser ws test
func TestAddSubUser(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsAddSubUser("1", "123456789101112aA!")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsSubmitOrder ws test
func TestWsSubmitOrder(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsSubmitOrder(currency.NewPairWithDelimiter(currency.LTC.String(), currency.BTC.String(), "").Lower(), 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsCancelOrder ws test
func TestWsCancelOrder(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsCancelOrder(currency.NewPairWithDelimiter(currency.LTC.String(), currency.BTC.String(), "").Lower(), 1234)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetAccountInfo ws test
func TestWsGetAccountInfo(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsGetAccountInfoRequest()
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetOrder ws test
func TestWsGetOrder(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsGetOrder(currency.NewPairWithDelimiter(currency.LTC.String(), currency.BTC.String(), "").Lower(), 1234)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetOrders ws test
func TestWsGetOrders(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsGetOrders(currency.NewPairWithDelimiter(currency.LTC.String(), currency.BTC.String(), "").Lower(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetOrdersIgnoreTradeType ws test
func TestWsGetOrdersIgnoreTradeType(t *testing.T) {
	setupWsAuth(t)
	_, err := z.wsGetOrdersIgnoreTradeType(currency.NewPairWithDelimiter(currency.LTC.String(), currency.BTC.String(), "").Lower(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}
