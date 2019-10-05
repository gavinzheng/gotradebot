package okex

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
)

const (
	okExAuthRate     = 600
	okExUnauthRate   = 600
	okExAPIPath      = "api/"
	okExAPIURL       = "https://www.okex.com/" + okExAPIPath
	okExAPIVersion   = "/v3/"
	okExExchangeName = "OKEX"
	// OkExWebsocketURL WebsocketURL
	OkExWebsocketURL = "wss://real.okex.com:10442/ws/v3"
	// API subsections
	okGroupFuturesSubsection = "futures"
	okGroupSwapSubsection    = "swap"
	okGroupETTSubsection     = "ett"
	// Futures based endpoints
	okGroupFuturePosition = "position"
	okGroupFutureLeverage = "leverage"
	okGroupFutureOrder    = "order"
	okGroupFutureHolds    = "holds"
	okGroupIndices        = "index"
	okGroupRate           = "rate"
	okGroupEsimtatedPrice = "estimated_price"
	okGroupOpenInterest   = "open_interest"
	// Perpetual swap based endpoints
	okGroupSettings              = "settings"
	okGroupDepth                 = "depth"
	okGroupFundingTime           = "funding_time"
	okGroupHistoricalFundingRate = "historical_funding_rate"
	// ETT endpoints
	okGroupConstituents = "constituents"
	okGroupDefinePrice  = "define-price"
)

// OKEX bases all account, spot and margin methods off okgroup implementation
type OKEX struct {
	okgroup.OKGroup
}

// SetDefaults method assignes the default values for OKEX
func (o *OKEX) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = okExExchangeName
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okExAuthRate),
		request.NewRateLimit(time.Second, okExUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = okExAPIURL
	o.APIUrl = okExAPIURL
	o.AssetTypes = []string{ticker.Spot}
	o.Websocket = wshandler.New()
	o.APIVersion = okExAPIVersion
	o.WebsocketURL = OkExWebsocketURL
	o.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketMessageCorrelationSupported
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout

}

// GetFuturesPostions Get the information of all holding positions in futures trading.
// Due to high energy consumption, you are advised to capture data with the "Futures Account of a Currency" API instead.
func (o *OKEX) GetFuturesPostions() (resp okgroup.GetFuturesPositionsResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupFuturePosition, nil, &resp, true)
}

// GetFuturesPostionsForCurrency Get the information of holding positions of a contract.
func (o *OKEX) GetFuturesPostionsForCurrency(instrumentID string) (resp okgroup.GetFuturesPositionsForCurrencyResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", instrumentID, okGroupFuturePosition)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesAccountOfAllCurrencies Get the futures account info of all token.
// Due to high energy consumption, you are advised to capture data with the "Futures Account of a Currency" API instead.
func (o *OKEX) GetFuturesAccountOfAllCurrencies() (resp okgroup.FuturesAccountForAllCurrenciesResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okgroup.OKGroupAccounts, nil, &resp, true)
}

// GetFuturesAccountOfACurrency Get the futures account info of a token.
func (o *OKEX) GetFuturesAccountOfACurrency(instrumentID string) (resp okgroup.FuturesCurrencyData, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupAccounts, instrumentID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesLeverage Get the leverage of the futures account
func (o *OKEX) GetFuturesLeverage(instrumentID string) (resp okgroup.GetFuturesLeverageResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, instrumentID, okGroupFutureLeverage)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// SetFuturesLeverage Adjusting the leverage for futures account。
// Cross margin request requirements:  {"leverage":"10"}
// Fixed margin request requirements: {"instrument_id":"BTC-USD-180213","direction":"long","leverage":"10"}
func (o *OKEX) SetFuturesLeverage(request okgroup.SetFuturesLeverageRequest) (resp okgroup.SetFuturesLeverageResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, request.Currency, okGroupFutureLeverage)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// GetFuturesBillDetails Shows the account’s historical coin in flow and out flow.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetFuturesBillDetails(request okgroup.GetSpotBillDetailsForCurrencyRequest) (resp []okgroup.GetSpotBillDetailsForCurrencyResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupAccounts, request.Currency, okgroup.OKGroupLedger, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// PlaceFuturesOrder OKEx futures trading only supports limit orders.
// You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold in the order lifecycle.
// The assets and amount on hold depends on the order's specific type and parameters.
func (o *OKEX) PlaceFuturesOrder(request okgroup.PlaceFuturesOrderRequest) (resp okgroup.PlaceFuturesOrderResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, okGroupFutureOrder, request, &resp, true)
}

// PlaceFuturesOrderBatch Batch contract placing order operation.
func (o *OKEX) PlaceFuturesOrderBatch(request okgroup.PlaceFuturesOrderBatchRequest) (resp okgroup.PlaceFuturesOrderBatchResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, okgroup.OKGroupOrders, request, &resp, true)
}

// CancelFuturesOrder Cancelling an unfilled order.
func (o *OKEX) CancelFuturesOrder(request okgroup.CancelFuturesOrderRequest) (resp okgroup.CancelFuturesOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupCancelOrder, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// CancelFuturesOrderBatch With best effort, cancel all open orders.
func (o *OKEX) CancelFuturesOrderBatch(request okgroup.CancelMultipleSpotOrdersRequest) (resp okgroup.CancelMultipleSpotOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupCancelBatchOrders, request.InstrumentID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// GetFuturesOrderList List your orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetFuturesOrderList(request okgroup.GetFuturesOrdersListRequest) (resp okgroup.GetFuturesOrderListResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v%v", okgroup.OKGroupOrders, request.InstrumentID, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesOrderDetails Get order details by order ID.
func (o *OKEX) GetFuturesOrderDetails(request okgroup.GetFuturesOrderDetailsRequest) (resp okgroup.GetFuturesOrderDetailsResponseData, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupOrders, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesTransactionDetails  Get details of the recent filled orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetFuturesTransactionDetails(request okgroup.GetFuturesTransactionDetailsRequest) (resp []okgroup.GetFuturesTransactionDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", okgroup.OKGroupGetSpotTransactionDetails, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesContractInformation Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
func (o *OKEX) GetFuturesContractInformation() (resp []okgroup.GetFuturesContractInformationResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okgroup.OKGroupInstruments, nil, &resp, false)
}

// GetFuturesOrderBook List all contracts. This request does not support pagination. The full list will be returned for a request.
func (o *OKEX) GetFuturesOrderBook(request okgroup.GetFuturesOrderBookRequest) (resp okgroup.GetFuturesOrderBookResponse, err error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupGetSpotOrderBook, okgroup.FormatParameters(request))

	type tempOB struct {
		Bids      [][]string `json:"bids"`
		Asks      [][]string `json:"asks"`
		Timestamp time.Time  `json:"timestamp"`
	}

	var tmpOB tempOB
	err = o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &tmpOB, true)
	if err != nil {
		return resp, err
	}

	processOB := func(ob [][]string) ([]okgroup.FuturesOrderbookItem, error) {
		var processedOB []okgroup.FuturesOrderbookItem
		for x := range ob {
			price, convErr := strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, convErr
			}

			size, convErr := strconv.ParseInt(ob[x][1], 10, 64)
			if err != nil {
				return nil, convErr
			}

			liqOrders, convErr := strconv.ParseInt(ob[x][2], 10, 64)
			if err != nil {
				return nil, convErr
			}

			numOrders, convErr := strconv.ParseInt(ob[x][3], 10, 64)
			if err != nil {
				return nil, convErr
			}

			processedOB = append(processedOB, okgroup.FuturesOrderbookItem{
				Price:                 price,
				Size:                  size,
				ForceLiquidatedOrders: liqOrders,
				NumberOrders:          numOrders,
			})
		}
		return processedOB, nil
	}

	resp.Bids, err = processOB(tmpOB.Bids)
	if err != nil {
		return
	}

	resp.Asks, err = processOB(tmpOB.Asks)
	if err != nil {
		return
	}

	resp.Timestamp = tmpOB.Timestamp
	return resp, nil
}

// GetAllFuturesTokenInfo Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all contracts.
func (o *OKEX) GetAllFuturesTokenInfo() (resp []okgroup.GetFuturesTokenInfoResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupInstruments, okgroup.OKGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesTokenInfoForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a contract.
func (o *OKEX) GetFuturesTokenInfoForCurrency(instrumentID string) (resp okgroup.GetFuturesTokenInfoResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesFilledOrder Get the recent 300 transactions of all contracts. Pagination is not supported here.
// The whole book will be returned for one request. Websocket is recommended here.
func (o *OKEX) GetFuturesFilledOrder(request okgroup.GetFuturesFilledOrderRequest) (resp []okgroup.GetFuturesFilledOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupTrades, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
func (o *OKEX) GetFuturesMarketData(request okgroup.GetFuturesMarketDateRequest) (resp okgroup.GetFuturesMarketDataResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupGetSpotMarketData, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesHoldAmount Get the number of futures with hold.
func (o *OKEX) GetFuturesHoldAmount(instrumentID string) (resp okgroup.GetFuturesHoldAmountResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, instrumentID, okGroupFutureHolds)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesIndices Get Indices of tokens. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesIndices(instrumentID string) (resp okgroup.GetFuturesIndicesResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupIndices)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesExchangeRates Get the fiat exchange rates. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesExchangeRates() (resp okgroup.GetFuturesExchangeRatesResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupRate, nil, &resp, false)
}

// GetFuturesEstimatedDeliveryPrice the estimated delivery price. It is available 3 hours before delivery.
// This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesEstimatedDeliveryPrice(instrumentID string) (resp okgroup.GetFuturesEstimatedDeliveryPriceResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupEsimtatedPrice)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesOpenInterests Get the open interest of a contract. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesOpenInterests(instrumentID string) (resp okgroup.GetFuturesOpenInterestsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupOpenInterest)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesCurrentPriceLimit The maximum buying price and the minimum selling price of the contract.
// This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesCurrentPriceLimit(instrumentID string) (resp okgroup.GetFuturesCurrentPriceLimitResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupPriceLimit)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesCurrentMarkPrice The maximum buying price and the minimum selling price of the contract.
// This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesCurrentMarkPrice(instrumentID string) (resp okgroup.GetFuturesCurrentMarkPriceResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupMarkPrice)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesForceLiquidatedOrders Get force liquidated orders. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesForceLiquidatedOrders(request okgroup.GetFuturesForceLiquidatedOrdersRequest) (resp []okgroup.GetFuturesForceLiquidatedOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupLiquidation, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesTagPrice Get the tag price. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetFuturesTagPrice(instrumentID string) (resp okgroup.GetFuturesTagPriceResponse, _ error) {
	// OKEX documentation is missing for this endpoint. Guessing "tag_price" for the URL results in 404
	return okgroup.GetFuturesTagPriceResponse{}, common.ErrNotYetImplemented
}

// GetSwapPostions Get the information of all holding positions in swap trading.
// Due to high energy consumption, you are advised to capture data with the "Swap Account of a Currency" API instead.
func (o *OKEX) GetSwapPostions() (resp []okgroup.GetSwapPostionsResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, okGroupFuturePosition, nil, &resp, true)
}

// GetSwapPostionsForContract Get the information of holding positions of a contract.
func (o *OKEX) GetSwapPostionsForContract(instrumentID string) (resp okgroup.GetSwapPostionsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", instrumentID, okGroupFuturePosition)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// GetSwapAccountOfAllCurrency Get the perpetual swap account info of a token.
// Margin ratio set as 10,000 when users have no open position.
func (o *OKEX) GetSwapAccountOfAllCurrency() (resp okgroup.GetSwapAccountOfAllCurrencyResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, okgroup.OKGroupAccounts, nil, &resp, true)
}

// GetSwapAccountSettingsOfAContract Get leverage level and margin mode of a contract.
func (o *OKEX) GetSwapAccountSettingsOfAContract(instrumentID string) (resp okgroup.GetSwapAccountSettingsOfAContractResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, instrumentID, okGroupSettings)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// SetSwapLeverageLevelOfAContract Setting the leverage level of a contract
// TODO this returns invalid parameters, but matches spec. Unsure how to fix
func (o *OKEX) SetSwapLeverageLevelOfAContract(request okgroup.SetSwapLeverageLevelOfAContractRequest) (resp okgroup.SetSwapLeverageLevelOfAContractResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, request.InstrumentID, okGroupFutureLeverage)
	request.InstrumentID = ""
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupSwapSubsection, requestURL, request, &resp, true)
}

// GetSwapBillDetails Shows the account’s historical coin in flow and out flow.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetSwapBillDetails(request okgroup.GetSpotBillDetailsForCurrencyRequest) (resp []okgroup.GetSwapBillDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupAccounts, request.Currency, okgroup.OKGroupLedger, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// PlaceSwapOrder OKEx perpetual swap trading only supports limit orders，USD as quote currency for orders.
// You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold in the order lifecycle.
// The assets and amount on hold depends on the order's specific type and parameters.
func (o *OKEX) PlaceSwapOrder(request okgroup.PlaceSwapOrderRequest) (resp okgroup.PlaceSwapOrderResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupSwapSubsection, okGroupFutureOrder, request, &resp, true)
}

// PlaceMultipleSwapOrders Batch contract placing order operation.
func (o *OKEX) PlaceMultipleSwapOrders(request okgroup.PlaceMultipleSwapOrdersRequest) (resp okgroup.PlaceMultipleSwapOrdersResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupSwapSubsection, okgroup.OKGroupOrders, request, &resp, true)
}

// CancelSwapOrder Cancelling an unfilled order
func (o *OKEX) CancelSwapOrder(request okgroup.CancelSwapOrderRequest) (resp okgroup.CancelSwapOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupCancelOrder, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// CancelMultipleSwapOrders With best effort, cancel all open orders.
func (o *OKEX) CancelMultipleSwapOrders(request okgroup.CancelMultipleSwapOrdersRequest) (resp okgroup.CancelMultipleSwapOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupCancelBatchOrders, request.InstrumentID)
	request.InstrumentID = ""
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupSwapSubsection, requestURL, request, &resp, true)
}

// GetSwapOrderList List your orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetSwapOrderList(request okgroup.GetSwapOrderListRequest) (resp okgroup.GetSwapOrderListResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v%v", okgroup.OKGroupOrders, request.InstrumentID, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// GetSwapOrderDetails Get order details by order ID.
func (o *OKEX) GetSwapOrderDetails(request okgroup.GetSwapOrderDetailsRequest) (resp okgroup.GetSwapOrderListResponseData, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupOrders, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// GetSwapTransactionDetails Get details of the recent filled orders
func (o *OKEX) GetSwapTransactionDetails(request okgroup.GetSwapTransactionDetailsRequest) (resp []okgroup.GetSwapTransactionDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", okgroup.OKGroupGetSpotTransactionDetails, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// GetSwapContractInformation Get market data.
func (o *OKEX) GetSwapContractInformation() (resp []okgroup.GetSwapContractInformationResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, okgroup.OKGroupInstruments, nil, &resp, false)
}

// GetSwapOrderBook Get the charts of the trading pairs.
func (o *OKEX) GetSwapOrderBook(request okgroup.GetSwapOrderBookRequest) (resp okgroup.GetSwapOrderBookResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okGroupDepth, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetAllSwapTokensInformation Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all contracts.
func (o *OKEX) GetAllSwapTokensInformation() (resp []okgroup.GetAllSwapTokensInformationResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupInstruments, okgroup.OKGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapTokensInformationForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all contracts.
func (o *OKEX) GetSwapTokensInformationForCurrency(instrumentID string) (resp okgroup.GetAllSwapTokensInformationResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapFilledOrdersData Get details of the recent filled orders
func (o *OKEX) GetSwapFilledOrdersData(request *okgroup.GetSwapFilledOrdersDataRequest) (resp []okgroup.GetSwapFilledOrdersDataResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupTrades, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapMarketData Get the charts of the trading pairs.
func (o *OKEX) GetSwapMarketData(request okgroup.GetSwapMarketDataRequest) (resp []okgroup.GetSwapMarketDataResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupGetSpotMarketData, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapIndices Get Indices of tokens.
func (o *OKEX) GetSwapIndices(instrumentID string) (resp okgroup.GetSwapIndecesResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupIndices)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapExchangeRates Get the fiat exchange rates.
func (o *OKEX) GetSwapExchangeRates() (resp okgroup.GetSwapExchangeRatesResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, okGroupRate, nil, &resp, false)
}

// GetSwapOpenInterest Get the open interest of a contract.
func (o *OKEX) GetSwapOpenInterest(instrumentID string) (resp okgroup.GetSwapExchangeRatesResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupOpenInterest)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapCurrentPriceLimits Get the open interest of a contract.
func (o *OKEX) GetSwapCurrentPriceLimits(instrumentID string) (resp okgroup.GetSwapCurrentPriceLimitsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupPriceLimit)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapForceLiquidatedOrders Get force liquidated orders.
func (o *OKEX) GetSwapForceLiquidatedOrders(request okgroup.GetSwapForceLiquidatedOrdersRequest) (resp []okgroup.GetSwapForceLiquidatedOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okgroup.OKGroupLiquidation, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapOnHoldAmountForOpenOrders Get On Hold Amount for Open Orders.
func (o *OKEX) GetSwapOnHoldAmountForOpenOrders(instrumentID string) (resp okgroup.GetSwapOnHoldAmountForOpenOrdersResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, instrumentID, okGroupFutureHolds)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, true)
}

// GetSwapNextSettlementTime Get the time of next settlement.
func (o *OKEX) GetSwapNextSettlementTime(instrumentID string) (resp okgroup.GetSwapNextSettlementTimeResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okGroupFundingTime)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapMarkPrice Get the time of next settlement.
func (o *OKEX) GetSwapMarkPrice(instrumentID string) (resp okgroup.GetSwapMarkPriceResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupInstruments, instrumentID, okgroup.OKGroupMarkPrice)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetSwapFundingRateHistory Get Funding Rate History.
func (o *OKEX) GetSwapFundingRateHistory(request okgroup.GetSwapFundingRateHistoryRequest) (resp []okgroup.GetSwapFundingRateHistoryResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", okgroup.OKGroupInstruments, request.InstrumentID, okGroupHistoricalFundingRate, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupSwapSubsection, requestURL, nil, &resp, false)
}

// GetETT List the assets in ETT account. Get information such as balance, amount on hold/ available.
func (o *OKEX) GetETT() (resp []okgroup.GetETTResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, okgroup.OKGroupAccounts, nil, &resp, true)
}

// GetETTAccountInformationForCurrency Getting the balance, amount available/on hold of a token in ETT account.
func (o *OKEX) GetETTAccountInformationForCurrency(currency string) (resp okgroup.GetETTResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupAccounts, currency)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, true)
}

// GetETTBillsDetails Bills details. All paginated requests return the latest information (newest)
// as the first page sorted by newest (in chronological time) first
func (o *OKEX) GetETTBillsDetails(currency string) (resp []okgroup.GetETTBillsDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okgroup.OKGroupAccounts, currency, okgroup.OKGroupLedger)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, true)
}

// PlaceETTOrder You can place subscription or redemption orders under ETT trading.
// You can place an order only if you have enough funds. Once your order is placed,
// the amount will be put on hold in the order lifecycle.
// The assets and amount on hold depends on the order's specific type and parameters.
func (o *OKEX) PlaceETTOrder(request *okgroup.PlaceETTOrderRequest) (resp okgroup.PlaceETTOrderResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupETTSubsection, okgroup.OKGroupOrders, nil, &resp, true)
}

// CancelETTOrder Cancel an unfilled order.
func (o *OKEX) CancelETTOrder(orderID string) (resp okgroup.PlaceETTOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupOrders, orderID)
	return resp, o.SendHTTPRequest(http.MethodDelete, okGroupETTSubsection, requestURL, nil, &resp, true)
}

// GetETTOrderList List your orders. Cursor pagination is used. All paginated requests return the latest information
// (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKEX) GetETTOrderList(request okgroup.GetETTOrderListRequest) (resp []okgroup.GetETTOrderListResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", okgroup.OKGroupOrders, okgroup.FormatParameters(request))
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, true)
}

// GetETTOrderDetails Get order details by order ID.
func (o *OKEX) GetETTOrderDetails(orderID string) (resp okgroup.GetETTOrderListResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okgroup.OKGroupOrders, orderID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, true)
}

// GetETTConstituents Get ETT Constituents.This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetETTConstituents(ett string) (resp okgroup.GetETTConstituentsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okGroupConstituents, ett)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, false)
}

// GetETTSettlementPriceHistory Get ETT settlement price history. This is a public endpoint, no identity verification is needed.
func (o *OKEX) GetETTSettlementPriceHistory(ett string) (resp []okgroup.GetETTSettlementPriceHistoryResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okGroupDefinePrice, ett)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupETTSubsection, requestURL, nil, &resp, false)
}
