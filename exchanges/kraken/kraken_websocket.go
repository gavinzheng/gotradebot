package kraken

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// List of all websocket channels to subscribe to
const (
	krakenWSURL              = "wss://ws.kraken.com"
	krakenWSSandboxURL       = "wss://sandbox.kraken.com"
	krakenWSSupportedVersion = "0.2.0"
	// If a checksum fails, then resubscribing to the channel fails, fatal after these attempts
	krakenWsResubscribeFailureLimit   = 3
	krakenWsResubscribeDelayInSeconds = 3
	// WS endpoints
	krakenWsHeartbeat          = "heartbeat"
	krakenWsPing               = "ping"
	krakenWsPong               = "pong"
	krakenWsSystemStatus       = "systemStatus"
	krakenWsSubscribe          = "subscribe"
	krakenWsSubscriptionStatus = "subscriptionStatus"
	krakenWsUnsubscribe        = "unsubscribe"
	krakenWsTicker             = "ticker"
	krakenWsOHLC               = "ohlc"
	krakenWsTrade              = "trade"
	krakenWsSpread             = "spread"
	krakenWsOrderbook          = "book"
	// Only supported asset type
	orderbookBufferLimit = 3
	krakenWsRateLimit    = 50
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var orderbookMutex sync.Mutex
var subscriptionChannelPair []WebsocketChannelData

// krakenOrderBooks TODO THIS IS A TEMPORARY SOLUTION UNTIL ENGINE BRANCH IS MERGED
// WS orderbook data can only rely on WS orderbook data
// Currently REST and WS runs simultaneously, dirtying the data
var krakenOrderBooks map[int64]orderbook.Base

// orderbookBuffer Stores orderbook updates per channel
var orderbookBuffer map[int64][]orderbook.Base
var subscribeToDefaultChannels = true

// Channels require a topic and a currency
// Format [[ticker,but-t4u],[orderbook,nce-btt]]
var defaultSubscribedChannels = []string{krakenWsTicker, krakenWsTrade, krakenWsOrderbook, krakenWsOHLC, krakenWsSpread}

// WsConnect initiates a websocket connection
func (k *Kraken) WsConnect() error {
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := k.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go k.WsHandleData()
	go k.wsPingHandler()
	if subscribeToDefaultChannels {
		k.GenerateDefaultSubscriptions()
	}

	return nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsPingHandler() {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()
	ticker := time.NewTicker(time.Second * 27)
	defer ticker.Stop()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		case <-ticker.C:
			pingEvent := WebsocketBaseEventRequest{Event: krakenWsPing}
			if k.Verbose {
				log.Debugf("%v sending ping",
					k.Name)
			}
			err := k.WebsocketConn.SendMessage(pingEvent)
			if err != nil {
				k.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData handles the read data from the websocket connection
func (k *Kraken) WsHandleData() {
	k.Websocket.Wg.Add(1)
	defer func() {
		k.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		default:
			resp, err := k.WebsocketConn.ReadMessage()
			if err != nil {
				k.Websocket.DataHandler <- fmt.Errorf("%v WsHandleData: %v",
					k.Name,
					err)
				return
			}
			k.Websocket.TrafficAlert <- struct{}{}
			// event response handling
			var eventResponse WebsocketEventResponse
			err = common.JSONDecode(resp.Raw, &eventResponse)
			if err == nil && eventResponse.Event != "" {
				k.WsHandleEventResponse(&eventResponse, resp.Raw)
				continue
			}
			// Data response handling
			var dataResponse WebsocketDataResponse
			err = common.JSONDecode(resp.Raw, &dataResponse)
			if err == nil && dataResponse[0].(float64) >= 0 {
				k.WsHandleDataResponse(dataResponse)
				continue
			}
			continue
		}
	}
}

// WsHandleDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleDataResponse(response WebsocketDataResponse) {
	channelID := int64(response[0].(float64))
	channelData := getSubscriptionChannelData(channelID)
	switch channelData.Subscription {
	case krakenWsTicker:
		if k.Verbose {
			log.Debugf("%v Websocket ticker data received",
				k.Name)
		}
		k.wsProcessTickers(&channelData, response[1])
	case krakenWsOHLC:
		if k.Verbose {
			log.Debugf("%v Websocket OHLC data received",
				k.Name)
		}
		k.wsProcessCandles(&channelData, response[1])
	case krakenWsOrderbook:
		if k.Verbose {
			log.Debugf("%v Websocket Orderbook data received",
				k.Name)
		}
		k.wsProcessOrderBook(&channelData, response[1])
	case krakenWsSpread:
		if k.Verbose {
			log.Debugf("%v Websocket Spread data received",
				k.Name)
		}
		k.wsProcessSpread(&channelData, response[1])
	case krakenWsTrade:
		if k.Verbose {
			log.Debugf("%v Websocket Trade data received",
				k.Name)
		}
		k.wsProcessTrades(&channelData, response[1])
	default:
		log.Errorf("%v Unidentified websocket data received: %v",
			k.Name,
			response)
	}
}

// WsHandleEventResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleEventResponse(response *WebsocketEventResponse, rawResponse []byte) {
	switch response.Event {
	case krakenWsHeartbeat:
		if k.Verbose {
			log.Debugf("%v Websocket heartbeat data received",
				k.Name)
		}
	case krakenWsPong:
		if k.Verbose {
			log.Debugf("%v Websocket pong data received",
				k.Name)
		}
	case krakenWsSystemStatus:
		if k.Verbose {
			log.Debugf("%v Websocket status data received",
				k.Name)
		}
		if response.Status != "online" {
			k.Websocket.DataHandler <- fmt.Errorf("%v Websocket status '%v'",
				k.Name, response.Status)
		}
		if response.WebsocketStatusResponse.Version != krakenWSSupportedVersion {
			log.Warnf("%v New version of Websocket API released. Was %v Now %v",
				k.Name, krakenWSSupportedVersion, response.WebsocketStatusResponse.Version)
		}
	case krakenWsSubscriptionStatus:
		k.WebsocketConn.AddResponseWithID(response.RequestID, rawResponse)
		if response.Status != "subscribed" {
			k.Websocket.DataHandler <- fmt.Errorf("%v %v %v", k.Name, response.RequestID, response.WebsocketErrorResponse.ErrorMessage)
			return
		}
		addNewSubscriptionChannelData(response)
	default:
		log.Errorf("%v Unidentified websocket data received: %v", k.Name, response)
	}
}

// addNewSubscriptionChannelData stores channel ids, pairs and subscription types to an array
// allowing correlation between subscriptions and returned data
func addNewSubscriptionChannelData(response *WebsocketEventResponse) {
	for i := range subscriptionChannelPair {
		if response.ChannelID != subscriptionChannelPair[i].ChannelID {
			continue
		}
		// kill the stale orderbooks due to resubscribing
		if orderbookBuffer == nil {
			orderbookBuffer = make(map[int64][]orderbook.Base)
		}
		orderbookBuffer[response.ChannelID] = []orderbook.Base{}
		if krakenOrderBooks == nil {
			krakenOrderBooks = make(map[int64]orderbook.Base)
		}
		krakenOrderBooks[response.ChannelID] = orderbook.Base{}
		return
	}

	// We change the / to - to maintain compatibility with REST/config
	pair := currency.NewPairWithDelimiter(response.Pair.Base.String(), response.Pair.Quote.String(), "-")
	subscriptionChannelPair = append(subscriptionChannelPair, WebsocketChannelData{
		Subscription: response.Subscription.Name,
		Pair:         pair,
		ChannelID:    response.ChannelID,
	})
}

// getSubscriptionChannelData retrieves WebsocketChannelData based on response ID
func getSubscriptionChannelData(id int64) WebsocketChannelData {
	for i := range subscriptionChannelPair {
		if id == subscriptionChannelPair[i].ChannelID {
			return subscriptionChannelPair[i]
		}
	}
	return WebsocketChannelData{}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(channelData *WebsocketChannelData, data interface{}) {
	tickerData := data.(map[string]interface{})
	closeData := tickerData["c"].([]interface{})
	openData := tickerData["o"].([]interface{})
	lowData := tickerData["l"].([]interface{})
	highData := tickerData["h"].([]interface{})
	volumeData := tickerData["v"].([]interface{})
	closePrice, _ := strconv.ParseFloat(closeData[0].(string), 64)
	openPrice, _ := strconv.ParseFloat(openData[0].(string), 64)
	highPrice, _ := strconv.ParseFloat(highData[0].(string), 64)
	lowPrice, _ := strconv.ParseFloat(lowData[0].(string), 64)
	quantity, _ := strconv.ParseFloat(volumeData[0].(string), 64)

	k.Websocket.DataHandler <- wshandler.TickerData{
		Timestamp:  time.Now(),
		Exchange:   k.Name,
		AssetType:  orderbook.Spot,
		Pair:       channelData.Pair,
		ClosePrice: closePrice,
		OpenPrice:  openPrice,
		HighPrice:  highPrice,
		LowPrice:   lowPrice,
		Quantity:   quantity,
	}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(channelData *WebsocketChannelData, data interface{}) {
	spreadData := data.([]interface{})
	bestBid := spreadData[0].(string)
	bestAsk := spreadData[1].(string)
	timeData, _ := strconv.ParseFloat(spreadData[2].(string), 64)
	bidVolume := spreadData[3].(string)
	askVolume := spreadData[4].(string)
	sec, dec := math.Modf(timeData)
	spreadTimestamp := time.Unix(int64(sec), int64(dec*(1e9)))
	if k.Verbose {
		log.Debugf("%v Spread data for '%v' received. Best bid: '%v' Best ask: '%v' Time: '%v', Bid volume '%v', Ask volume '%v'",
			k.Name,
			channelData.Pair,
			bestBid,
			bestAsk,
			spreadTimestamp,
			bidVolume,
			askVolume)
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(channelData *WebsocketChannelData, data interface{}) {
	tradeData := data.([]interface{})
	for i := range tradeData {
		trade := tradeData[i].([]interface{})
		timeData, _ := strconv.ParseInt(trade[2].(string), 10, 64)
		timeUnix := time.Unix(timeData, 0)
		price, _ := strconv.ParseFloat(trade[0].(string), 64)
		amount, _ := strconv.ParseFloat(trade[1].(string), 64)

		k.Websocket.DataHandler <- wshandler.TradeData{
			AssetType:    orderbook.Spot,
			CurrencyPair: channelData.Pair,
			EventTime:    time.Now().Unix(),
			Exchange:     k.Name,
			Price:        price,
			Amount:       amount,
			Timestamp:    timeUnix,
			Side:         trade[3].(string),
		}
	}
}

// wsProcessOrderBook determines if the orderbook data is partial or update
// Then sends to appropriate fun
func (k *Kraken) wsProcessOrderBook(channelData *WebsocketChannelData, data interface{}) {
	obData := data.(map[string]interface{})
	if _, ok := obData["as"]; ok {
		k.wsProcessOrderBookPartial(channelData, obData)
	} else {
		_, asksExist := obData["a"]
		_, bidsExist := obData["b"]
		if asksExist || bidsExist {
			k.wsRequestMtx.Lock()
			defer k.wsRequestMtx.Unlock()
			k.wsProcessOrderBookBuffer(channelData, obData)
			if len(orderbookBuffer[channelData.ChannelID]) >= orderbookBufferLimit {
				err := k.wsProcessOrderBookUpdate(channelData)
				if err != nil {
					subscriptionToRemove := wshandler.WebsocketChannelSubscription{
						Channel:  krakenWsOrderbook,
						Currency: channelData.Pair,
					}
					k.Websocket.ResubscribeToChannel(subscriptionToRemove)
				}
			}
		}
	}
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(channelData *WebsocketChannelData, obData map[string]interface{}) {
	ob := orderbook.Base{
		Pair:      channelData.Pair,
		AssetType: orderbook.Spot,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is timestamped per entry
	// Using the highest last update time, we can attempt to respect both within a reasonable degree
	var highestLastUpdate time.Time
	askData := obData["as"].([]interface{})
	for i := range askData {
		asks := askData[i].([]interface{})
		price, _ := strconv.ParseFloat(asks[0].(string), 64)
		amount, _ := strconv.ParseFloat(asks[1].(string), 64)
		ob.Asks = append(ob.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeData, _ := strconv.ParseFloat(asks[2].(string), 64)
		sec, dec := math.Modf(timeData)
		askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	bidData := obData["bs"].([]interface{})
	for i := range bidData {
		bids := bidData[i].([]interface{})
		price, _ := strconv.ParseFloat(bids[0].(string), 64)
		amount, _ := strconv.ParseFloat(bids[1].(string), 64)
		ob.Bids = append(ob.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeData, _ := strconv.ParseFloat(bids[2].(string), 64)
		sec, dec := math.Modf(timeData)
		bidUpdateTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(bidUpdateTime) {
			highestLastUpdate = bidUpdateTime
		}
	}

	ob.LastUpdated = highestLastUpdate
	err := k.Websocket.Orderbook.LoadSnapshot(&ob, k.Name, true)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	k.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: k.Name,
		Asset:    orderbook.Spot,
		Pair:     channelData.Pair,
	}

	if krakenOrderBooks == nil {
		krakenOrderBooks = make(map[int64]orderbook.Base)
	}
	krakenOrderBooks[channelData.ChannelID] = ob
}

func (k *Kraken) wsProcessOrderBookBuffer(channelData *WebsocketChannelData, obData map[string]interface{}) {
	ob := orderbook.Base{
		AssetType:    orderbook.Spot,
		ExchangeName: k.Name,
		Pair:         channelData.Pair,
	}

	var highestLastUpdate time.Time
	// Ask data is not always sent
	if _, ok := obData["a"]; ok {
		askData := obData["a"].([]interface{})
		for i := range askData {
			asks := askData[i].([]interface{})
			price, _ := strconv.ParseFloat(asks[0].(string), 64)
			amount, _ := strconv.ParseFloat(asks[1].(string), 64)
			ob.Asks = append(ob.Asks, orderbook.Item{
				Amount: amount,
				Price:  price,
			})

			timeData, _ := strconv.ParseFloat(asks[2].(string), 64)
			sec, dec := math.Modf(timeData)
			askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
			if highestLastUpdate.Before(askUpdatedTime) {
				highestLastUpdate = askUpdatedTime
			}
		}
	}
	// Bid data is not always sent
	if _, ok := obData["b"]; ok {
		bidData := obData["b"].([]interface{})
		for i := range bidData {
			bids := bidData[i].([]interface{})
			price, _ := strconv.ParseFloat(bids[0].(string), 64)
			amount, _ := strconv.ParseFloat(bids[1].(string), 64)
			ob.Bids = append(ob.Bids, orderbook.Item{
				Amount: amount,
				Price:  price,
			})
			timeData, _ := strconv.ParseFloat(bids[2].(string), 64)
			sec, dec := math.Modf(timeData)
			bidUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
			if highestLastUpdate.Before(bidUpdatedTime) {
				highestLastUpdate = bidUpdatedTime
			}
		}
	}
	ob.LastUpdated = highestLastUpdate
	if orderbookBuffer == nil {
		orderbookBuffer = make(map[int64][]orderbook.Base)
	}
	orderbookBuffer[channelData.ChannelID] = append(orderbookBuffer[channelData.ChannelID], ob)
	if k.Verbose {
		log.Debugf("%v Adding orderbook to buffer for channel %v. Lastupdated: %v. %v / %v",
			k.Name,
			channelData.ChannelID,
			ob.LastUpdated,
			len(orderbookBuffer[channelData.ChannelID]),
			orderbookBufferLimit)
	}
}

// wsProcessOrderBookUpdate updates an orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookUpdate(channelData *WebsocketChannelData) error {
	if k.Verbose {
		log.Debugf("%v Current orderbook 'LastUpdated': %v",
			k.Name,
			krakenOrderBooks[channelData.ChannelID].LastUpdated)
	}
	lowestLastUpdated := orderbookBuffer[channelData.ChannelID][0].LastUpdated
	if k.Verbose {
		log.Debugf("%v Sorting orderbook. Earliest 'LastUpdated' entry: %v",
			k.Name,
			lowestLastUpdated)
	}
	sort.Slice(orderbookBuffer[channelData.ChannelID], func(i, j int) bool {
		return orderbookBuffer[channelData.ChannelID][i].LastUpdated.Before(orderbookBuffer[channelData.ChannelID][j].LastUpdated)
	})

	lowestLastUpdated = orderbookBuffer[channelData.ChannelID][0].LastUpdated
	if k.Verbose {
		log.Debugf("%v Sorted orderbook. Earliest 'LastUpdated' entry: %v",
			k.Name,
			lowestLastUpdated)
	}
	// The earliest update has to be after the previously stored orderbook
	if krakenOrderBooks[channelData.ChannelID].LastUpdated.After(lowestLastUpdated) {
		err := fmt.Errorf("%v orderbook update out of order. Existing: %v, Attempted: %v",
			k.Name,
			krakenOrderBooks[channelData.ChannelID].LastUpdated,
			lowestLastUpdated)
		k.Websocket.DataHandler <- err
		return err
	}

	k.updateChannelOrderbookEntries(channelData)
	highestLastUpdate := orderbookBuffer[channelData.ChannelID][len(orderbookBuffer[channelData.ChannelID])-1].LastUpdated
	if k.Verbose {
		log.Debugf("%v Saving orderbook. Lastupdated: %v",
			k.Name,
			highestLastUpdate)
	}

	ob := krakenOrderBooks[channelData.ChannelID]
	ob.LastUpdated = highestLastUpdate
	err := k.Websocket.Orderbook.LoadSnapshot(&ob, k.Name, true)
	if err != nil {
		k.Websocket.DataHandler <- err
		return err
	}

	k.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: k.Name,
		Asset:    orderbook.Spot,
		Pair:     channelData.Pair,
	}
	// Reset the buffer
	orderbookBuffer[channelData.ChannelID] = []orderbook.Base{}
	return nil
}

func (k *Kraken) updateChannelOrderbookEntries(channelData *WebsocketChannelData) {
	for i := 0; i < len(orderbookBuffer[channelData.ChannelID]); i++ {
		for j := 0; j < len(orderbookBuffer[channelData.ChannelID][i].Asks); j++ {
			k.updateChannelOrderbookAsks(i, j, channelData)
		}
		for j := 0; j < len(orderbookBuffer[channelData.ChannelID][i].Bids); j++ {
			k.updateChannelOrderbookBids(i, j, channelData)
		}
	}
}

func (k *Kraken) updateChannelOrderbookAsks(i, j int, channelData *WebsocketChannelData) {
	askFound := k.updateChannelOrderbookAsk(i, j, channelData)
	if !askFound {
		if k.Verbose {
			log.Debugf("%v Adding Ask for channel %v. Price %v. Amount %v",
				k.Name,
				channelData.ChannelID,
				orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
				orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount)
		}
		ob := krakenOrderBooks[channelData.ChannelID]
		ob.Asks = append(ob.Asks, orderbookBuffer[channelData.ChannelID][i].Asks[j])
		krakenOrderBooks[channelData.ChannelID] = ob
	}
}

func (k *Kraken) updateChannelOrderbookAsk(i, j int, channelData *WebsocketChannelData) bool {
	askFound := false
	for l := 0; l < len(krakenOrderBooks[channelData.ChannelID].Asks); l++ {
		if krakenOrderBooks[channelData.ChannelID].Asks[l].Price == orderbookBuffer[channelData.ChannelID][i].Asks[j].Price {
			askFound = true
			if orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount == 0 {
				// Remove existing entry
				if k.Verbose {
					log.Debugf("%v Removing Ask for channel %v. Price %v. Old amount %v. Buffer %v",
						k.Name,
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
						krakenOrderBooks[channelData.ChannelID].Asks[l].Amount, i)
				}
				ob := krakenOrderBooks[channelData.ChannelID]
				ob.Asks = append(ob.Asks[:l], ob.Asks[l+1:]...)
				krakenOrderBooks[channelData.ChannelID] = ob
				l--
			} else if krakenOrderBooks[channelData.ChannelID].Asks[l].Amount != orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount {
				if k.Verbose {
					log.Debugf("%v Updating Ask for channel %v. Price %v. Old amount %v, New Amount %v",
						k.Name,
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
						krakenOrderBooks[channelData.ChannelID].Asks[l].Amount,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount)
				}
				krakenOrderBooks[channelData.ChannelID].Asks[l].Amount = orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount
			}
			return askFound
		}
	}
	return askFound
}

func (k *Kraken) updateChannelOrderbookBids(i, j int, channelData *WebsocketChannelData) {
	bidFound := k.updateChannelOrderbookBid(i, j, channelData)
	if !bidFound {
		if k.Verbose {
			log.Debugf("%v Adding Bid for channel %v. Price %v. Amount %v",
				k.Name,
				channelData.ChannelID,
				orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
				orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount)
		}
		ob := krakenOrderBooks[channelData.ChannelID]
		ob.Bids = append(ob.Bids, orderbookBuffer[channelData.ChannelID][i].Bids[j])
		krakenOrderBooks[channelData.ChannelID] = ob
	}
}

func (k *Kraken) updateChannelOrderbookBid(i, j int, channelData *WebsocketChannelData) bool {
	bidFound := false
	for l := 0; l < len(krakenOrderBooks[channelData.ChannelID].Bids); l++ {
		if krakenOrderBooks[channelData.ChannelID].Bids[l].Price == orderbookBuffer[channelData.ChannelID][i].Bids[j].Price {
			bidFound = true
			if orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount == 0 {
				// Remove existing entry
				if k.Verbose {
					log.Debugf("%v Removing Bid for channel %v. Price %v. Old amount %v. Buffer %v",
						k.Name,
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
						krakenOrderBooks[channelData.ChannelID].Bids[l].Amount, i)
				}
				ob := krakenOrderBooks[channelData.ChannelID]
				ob.Bids = append(ob.Bids[:l], ob.Bids[l+1:]...)
				krakenOrderBooks[channelData.ChannelID] = ob
				l--
			} else if krakenOrderBooks[channelData.ChannelID].Bids[l].Amount != orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount {
				if k.Verbose {
					log.Debugf("%v Updating Bid for channel %v. Price %v. Old amount %v, New Amount %v",
						k.Name,
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
						krakenOrderBooks[channelData.ChannelID].Bids[l].Amount,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount)
				}
				krakenOrderBooks[channelData.ChannelID].Bids[l].Amount = orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount
			}
			return bidFound
		}
	}
	return bidFound
}

// wsProcessCandles converts candle data and sends it to the data handler
func (k *Kraken) wsProcessCandles(channelData *WebsocketChannelData, data interface{}) {
	candleData := data.([]interface{})
	startTimeData, _ := strconv.ParseInt(candleData[0].(string), 10, 64)
	startTimeUnix := time.Unix(startTimeData, 0)
	endTimeData, _ := strconv.ParseInt(candleData[1].(string), 10, 64)
	endTimeUnix := time.Unix(endTimeData, 0)
	openPrice, _ := strconv.ParseFloat(candleData[2].(string), 64)
	highPrice, _ := strconv.ParseFloat(candleData[3].(string), 64)
	lowPrice, _ := strconv.ParseFloat(candleData[4].(string), 64)
	closePrice, _ := strconv.ParseFloat(candleData[5].(string), 64)
	volume, _ := strconv.ParseFloat(candleData[7].(string), 64)

	k.Websocket.DataHandler <- wshandler.KlineData{
		AssetType: orderbook.Spot,
		Pair:      channelData.Pair,
		Timestamp: time.Now(),
		Exchange:  k.Name,
		StartTime: startTimeUnix,
		CloseTime: endTimeUnix,
		// Candles are sent every 60 seconds
		Interval:   "60",
		HighPrice:  highPrice,
		LowPrice:   lowPrice,
		OpenPrice:  openPrice,
		ClosePrice: closePrice,
		Volume:     volume,
	}
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (k *Kraken) GenerateDefaultSubscriptions() {
	enabledCurrencies := k.GetEnabledCurrencies()
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range defaultSubscribedChannels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = "/"
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  defaultSubscribedChannels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	k.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsSubscribe,
		Pairs: []string{channelToSubscribe.Currency.String()},
		Subscription: WebsocketSubscriptionData{
			Name: channelToSubscribe.Channel,
		},
		RequestID: k.WebsocketConn.GenerateMessageID(true),
	}
	_, err := k.WebsocketConn.SendMessageReturnResponse(resp.RequestID, resp)
	return err
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsUnsubscribe,
		Pairs: []string{channelToSubscribe.Currency.String()},
		Subscription: WebsocketSubscriptionData{
			Name: channelToSubscribe.Channel,
		},
		RequestID: k.WebsocketConn.GenerateMessageID(true),
	}
	_, err := k.WebsocketConn.SendMessageReturnResponse(resp.RequestID, resp)
	return err
}
