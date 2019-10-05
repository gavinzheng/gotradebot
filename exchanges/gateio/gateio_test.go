package gateio

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g Gateio
var wsSetupRan bool

func TestSetDefaults(t *testing.T) {
	g.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	gateioConfig, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		t.Error("Test Failed - GateIO Setup() init error")
	}
	gateioConfig.AuthenticatedWebsocketAPISupport = true
	gateioConfig.AuthenticatedAPISupport = true
	gateioConfig.APIKey = apiKey
	gateioConfig.APISecret = apiSecret

	g.Setup(&gateioConfig)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Errorf("Test failed - Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo()
	if err != nil {
		t.Errorf("Test failed - Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestParamsTypeSell,
	})
	if err != nil {
		t.Errorf("Test failed - Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.CancelExistingOrder(917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio CancelExistingOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.GetBalances()
	if err != nil {
		t.Errorf("Test failed - Gateio GetBalances: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetTickers()
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := g.GetSpotKline(KlinesRequestParams{
		Symbol:   "btc_usdt",
		GroupSec: TimeIntervalFiveMinutes, // 5 minutes or less
		HourSize: 1,                       // 1 hour data
	})

	if err != nil {
		t.Errorf("Test failed - Gateio GetSpotKline: %s", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_"),
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	g.GetFeeByType(feeBuilder)
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
	g.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()
	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	g.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText

	withdrawPermissions := g.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := g.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	currPair := currency.NewPair(currency.LTC, currency.BTC)
	currPair.Delimiter = "_"
	getOrdersRequest.Currencies = []currency.Pair{currPair}

	_, err := g.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if g.APIKey != "" && g.APIKey != "Key" &&
		g.APISecret != "" && g.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	var p = currency.Pair{
		Delimiter: "_",
		Base:      currency.LTC,
		Quote:     currency.BTC,
	}
	response, err := g.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 10, "1234234")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := g.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := g.CancelAllOrders(orderCancellation)

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
	if apiSecret == "" || apiKey == "" {
		_, err := g.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	} else {
		_, err := g.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := g.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.LTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := g.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := g.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := g.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := g.GetDepositAddress(currency.ETC, "")
		if err != nil {
			t.Error("Test Fail - GetDepositAddress error", err)
		}
	} else {
		_, err := g.GetDepositAddress(currency.ETC, "")
		if err == nil {
			t.Error("Test Fail - GetDepositAddress error cannot be nil")
		}
	}
}
func TestGetOrderInfo(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	if !areTestAPIKeysSet() {
		t.Skip("no API keys set skipping test")
	}

	_, err := g.GetOrderInfo("917591554")
	if err != nil {
		if err.Error() != "no order found with id 917591554" && err.Error() != "failed to get open orders" {
			t.Fatalf("GetOrderInfo() returned an error skipping test: %v", err)
		}
	}
}

// TestWsGetBalance dials websocket, sends balance request.
func TestWsGetBalance(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)
	if !g.Websocket.IsEnabled() && !g.AuthenticatedWebsocketAPISupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.WsHandleData()
	g.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	g.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	resp, err := g.wsServerSignIn()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Result.Status != "success" {
		t.Fatal("Unsuccessful login")
	}
	_, err = g.wsGetBalance([]string{"EOS", "BTC"})
	if err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderInfo dials websocket, sends order info request.
func TestWsGetOrderInfo(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)
	if !g.Websocket.IsEnabled() && !g.AuthenticatedWebsocketAPISupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.WsHandleData()
	g.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	g.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	resp, err := g.wsServerSignIn()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Result.Status != "success" {
		t.Fatal("Unsuccessful login")
	}
	_, err = g.wsGetOrderInfo("EOS_USDT", 0, 10)
	if err != nil {
		t.Error(err)
	}
}

func setupWSTestAuth(t *testing.T) {
	if wsSetupRan {
		return
	}
	g.SetDefaults()
	TestSetup(t)
	if !g.Websocket.IsEnabled() && !g.AuthenticatedWebsocketAPISupport {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.WsHandleData()
	g.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	g.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	wsSetupRan = true
}

// TestWsSubscribe dials websocket, sends a subscribe request.
func TestWsSubscribe(t *testing.T) {
	setupWSTestAuth(t)
	err := g.Subscribe(wshandler.WebsocketChannelSubscription{
		Channel:  "ticker.subscribe",
		Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsUnsubscribe dials websocket, sends an unsubscribe request.
func TestWsUnsubscribe(t *testing.T) {
	setupWSTestAuth(t)
	err := g.Unsubscribe(wshandler.WebsocketChannelSubscription{
		Channel:  "ticker.subscribe",
		Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
	})
	if err != nil {
		t.Error(err)
	}
}
