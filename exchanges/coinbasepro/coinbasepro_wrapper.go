package coinbasepro

import (
	"errors"
	"fmt"
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

// Start starts the coinbasepro go routine
func (c *CoinbasePro) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run() {
	if c.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket.IsEnabled()), coinbaseproWebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	exchangeProducts, err := c.GetProducts()
	if err != nil {
		log.Errorf("%s Failed to get available products.\n", c.GetName())
	} else {
		var currencies []string
		for _, x := range exchangeProducts {
			if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
				currencies = append(currencies, x.ID[0:3]+x.ID[4:])
			}
		}

		var newCurrencies currency.Pairs
		for _, p := range currencies {
			newCurrencies = append(newCurrencies,
				currency.NewPairFromString(p))
		}

		err = c.UpdateCurrencies(newCurrencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", c.GetName())
		}
	}
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = c.GetName()
	accountBalance, err := c.GetAccounts()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := c.GetTicker(exchange.FormatExchangeCurrency(c.Name, p).String())
	if err != nil {
		return ticker.Price{}, err
	}

	stats, err := c.GetStats(exchange.FormatExchangeCurrency(c.Name, p).String())

	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = tick.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low

	err = ticker.ProcessTicker(c.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (c *CoinbasePro) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (c *CoinbasePro) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := c.GetOrderbook(exchange.FormatExchangeCurrency(c.Name, p).String(), 2)
	if err != nil {
		return orderBook, err
	}

	obNew := orderbookNew.(OrderbookL1L2)

	for x := range obNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	for x := range obNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: obNew.Asks[x].Amount, Price: obNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = c.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *CoinbasePro) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *CoinbasePro) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var response string
	var err error

	switch orderType {
	case exchange.MarketOrderType:
		response, err = c.PlaceMarginOrder("",
			amount,
			amount,
			side.ToString(),
			p.String(),
			"")
	case exchange.LimitOrderType:
		response, err = c.PlaceLimitOrder("",
			price,
			amount,
			side.ToString(),
			"",
			"",
			p.String(),
			"",
			false)
	default:
		err = errors.New("order type not supported")
	}

	if response != "" {
		submitOrderResponse.OrderID = response
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(order *exchange.OrderCancellation) error {
	return c.CancelExistingOrder(order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders("")
	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (c *CoinbasePro) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	resp, err := c.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
	return resp.ID, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	paymentMethods, err := c.GetPayMethods()
	if err != nil {
		return "", err
	}

	selectedWithdrawalMethod := PaymentMethod{}
	for i := range paymentMethods {
		if withdrawRequest.BankName == paymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return "", fmt.Errorf("could not find payment method '%v'. Check the name via the website and try again", withdrawRequest.BankName)
	}

	resp, err := c.WithdrawViaPaymentMethod(withdrawRequest.Amount, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return c.WithdrawFiatFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *CoinbasePro) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (c.APIKey == "" || c.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var respOrders []GeneralizedOrderResponse
	for i := range getOrdersRequest.Currencies {
		resp, err := c.GetOrders([]string{"open", "pending", "active"},
			exchange.FormatExchangeCurrency(c.Name, getOrdersRequest.Currencies[i]).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range respOrders {
		currency := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.ConfigCurrencyPairFormat.Delimiter)
		orderSide := exchange.OrderSide(strings.ToUpper(respOrders[i].Side))
		orderType := exchange.OrderType(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name, "GetActiveOrders", respOrders[i].ID, respOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			OrderType:      orderType,
			OrderDate:      orderDate,
			OrderSide:      orderSide,
			CurrencyPair:   currency,
			Exchange:       c.Name,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var respOrders []GeneralizedOrderResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := c.GetOrders([]string{"done", "settled"},
			exchange.FormatExchangeCurrency(c.Name, currency).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range respOrders {
		currency := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.ConfigCurrencyPairFormat.Delimiter)
		orderSide := exchange.OrderSide(strings.ToUpper(respOrders[i].Side))
		orderType := exchange.OrderType(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name, "GetActiveOrders", respOrders[i].ID, respOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			OrderType:      orderType,
			OrderDate:      orderDate,
			OrderSide:      orderSide,
			CurrencyPair:   currency,
			Exchange:       c.Name,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *CoinbasePro) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *CoinbasePro) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (c *CoinbasePro) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return c.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *CoinbasePro) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
