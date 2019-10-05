package itbit

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the ItBit go routine
func (i *ItBit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		i.Run()
		wg.Done()
	}()
}

// Run implements the ItBit wrapper
func (i *ItBit) Run() {
	if i.Verbose {
		log.Debugf("%s polling delay: %ds.\n", i.GetName(), i.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", i.GetName(), len(i.EnabledPairs), i.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (i *ItBit) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := i.GetTicker(exchange.FormatExchangeCurrency(i.Name,
		p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.LastPrice
	tickerPrice.High = tick.High24h
	tickerPrice.Low = tick.Low24h
	tickerPrice.Volume = tick.Volume24h

	err = ticker.ProcessTicker(i.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(i.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (i *ItBit) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(i.GetName(), p, assetType)
	if err != nil {
		return i.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (i *ItBit) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(i.GetName(), p, assetType)
	if err != nil {
		return i.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (i *ItBit) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := i.GetOrderbook(exchange.FormatExchangeCurrency(i.Name,
		p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		var price, amount float64
		price, err = strconv.ParseFloat(data[0], 64)
		if err != nil {
			return orderBook, err
		}
		amount, err = strconv.ParseFloat(data[1], 64)
		if err != nil {
			return orderBook, err
		}
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: amount, Price: price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		var price, amount float64
		price, err = strconv.ParseFloat(data[0], 64)
		if err != nil {
			return orderBook, err
		}
		amount, err = strconv.ParseFloat(data[1], 64)
		if err != nil {
			return orderBook, err
		}
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: amount, Price: price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = i.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(i.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies
func (i *ItBit) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.Exchange = i.GetName()

	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return info, err
	}

	type balance struct {
		TotalValue float64
		Hold       float64
	}

	var amounts = make(map[string]*balance)

	for _, wallet := range wallets {
		for _, cb := range wallet.Balances {
			if _, ok := amounts[cb.Currency]; !ok {
				amounts[cb.Currency] = &balance{}
			}

			amounts[cb.Currency].TotalValue += cb.TotalBalance
			amounts[cb.Currency].Hold += cb.TotalBalance - cb.AvailableBalance
		}
	}

	var fullBalance []exchange.AccountCurrencyInfo

	for key, data := range amounts {
		fullBalance = append(fullBalance, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(key),
			TotalValue:   data.TotalValue,
			Hold:         data.Hold,
		})
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: fullBalance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (i *ItBit) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (i *ItBit) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (i *ItBit) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var wallet string

	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return submitOrderResponse, err
	}

	// Determine what wallet ID to use if there is any actual available currency to make the trade!
	for _, i := range wallets {
		for j := range i.Balances {
			if i.Balances[j].Currency == p.Base.String() &&
				i.Balances[j].AvailableBalance >= amount {
				wallet = i.ID
			}
		}
	}

	if wallet == "" {
		return submitOrderResponse,
			fmt.Errorf("no wallet found with currency: %s with amount >= %v",
				p.Base,
				amount)
	}

	response, err := i.PlaceOrder(wallet,
		side.ToString(),
		orderType.ToString(),
		p.Base.String(),
		amount,
		price,
		p.String(),
		"")

	if response.ID != "" {
		submitOrderResponse.OrderID = response.ID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (i *ItBit) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (i *ItBit) CancelOrder(order *exchange.OrderCancellation) error {
	return i.CancelExistingOrder(order.WalletAddress, order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (i *ItBit) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := i.GetOrders(orderCancellation.WalletAddress, "", "open", 0, 0)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for j := range openOrders {
		err = i.CancelExistingOrder(orderCancellation.WalletAddress, openOrders[j].ID)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[openOrders[j].ID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (i *ItBit) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
// NOTE: This has not been implemented due to the fact you need to generate a
// a specific wallet ID and they restrict the amount of deposit address you can
// request limiting them to 2.
func (i *ItBit) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (i *ItBit) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (i *ItBit) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (i *ItBit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (i.APIKey == "" || i.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return i.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (i *ItBit) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for _, wallet := range wallets {
		resp, err := i.GetOrders(wallet.ID, "", "open", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for j := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[j].Instrument,
			i.ConfigCurrencyPairFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[j].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name, "GetActiveOrders", allOrders[j].ID, allOrders[j].CreatedTime)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:              allOrders[j].ID,
			OrderSide:       side,
			Amount:          allOrders[j].Amount,
			ExecutedAmount:  allOrders[j].AmountFilled,
			RemainingAmount: (allOrders[j].Amount - allOrders[j].AmountFilled),
			Exchange:        i.Name,
			OrderDate:       orderDate,
			CurrencyPair:    symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (i *ItBit) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for _, wallet := range wallets {
		resp, err := i.GetOrders(wallet.ID, "", "", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for j := range allOrders {
		if allOrders[j].Type == "open" {
			continue
		}

		symbol := currency.NewPairDelimiter(allOrders[j].Instrument,
			i.ConfigCurrencyPairFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[j].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name, "GetActiveOrders", allOrders[j].ID, allOrders[j].CreatedTime)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:              allOrders[j].ID,
			OrderSide:       side,
			Amount:          allOrders[j].Amount,
			ExecutedAmount:  allOrders[j].AmountFilled,
			RemainingAmount: (allOrders[j].Amount - allOrders[j].AmountFilled),
			Exchange:        i.Name,
			OrderDate:       orderDate,
			CurrencyPair:    symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (i *ItBit) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (i *ItBit) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (i *ItBit) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (i *ItBit) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
