// TODO: Create copies of all orders to save in permanent storage
// Two separate stores: one for pending orders, one for completed orders
// Pending order storage for recreating the matching engine in case of panic/crash
// One for admin
package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kkomitski/exchange/orderbook"
	"github.com/kkomitski/exchange/utils"
	"github.com/labstack/echo/v4"
)

const exchangePrivateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

const (
	MarketETH Market = "ETH"

	MarketOrder OrderType = "MARKET"
	LimitOrder OrderType = "LIMIT"
)

type ( 
	OrderType string
	Market string

	PlaceOrderRequest struct {
		UserID int64
		Type OrderType // Limit or market
		Bid bool
		Size float64
		Price float64
		Market Market
	}

	Order struct {
		UserID int64
		ID    int64
		Price float64
		Size float64
		Bid bool
		Timestamp int64
	}

	OrderBookuserOrders struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks []*Order
		Bids []*Order
	}
	
	MatchedOrder struct {
		UserID int64
		Price float64
		Size float64
		ID int64
	}	
)

func StartServer() {
	e := echo.New()

	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("HTTP://127.0.0.1:8545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchange(exchangePrivateKey, client)
	if err != nil {
		log.Fatal(err)
	}

	pk1 := "6d9c82e581d012ea2723a50a93bfdccff997e7ea2320ad54adb46f9dfd8450b4"
	addr1 := "0x50FbCFa41279530064ed8Cb18fAbE99C77982F57"
	user1 := NewUser(pk1, addr1, 11)

	pk2 := "2e217ecde538ec3810a1ed4aa812bd8df0b1f1ac7ab7d8b79e9f976f12922d59"
	addr2 := "0x6D60CAcfdac815fcC6873A400F2DeB80C61D0BDB"
	user2 := NewUser(pk2, addr2, 22)

	pk3 := "01b7ba57f8ca8e547fa37b02f7019bda63ded578b050d848cfd26145ef95aa16"
	addr3 := "0xdAD83EB015197B5AFF313D3F2dA9bA14d39bA88D"
	user3 := NewUser(pk3, addr3, 33)

	ex.Users[user1.ID] = user1
	ex.Users[user2.ID] = user2
	ex.Users[user3.ID] = user3

	balance1 := user1.GetBalance(client, user1.Address)
	fmt.Printf("Balance: %+v\n", balance1)
	_ = balance1

	balance2 := user2.GetBalance(client, user2.Address)
	fmt.Printf("Balance: %+v\n", balance2)
	_ = balance2

	balance3 := user3.GetBalance(client, user3.Address)
	fmt.Printf("Balance: %+v\n", balance3)
	_ = balance3

	e.GET("/book/:market", ex.handleGetBook)
	e.GET("/book/:market/bid", ex.handleGetBestBid)
	e.GET("/book/:market/ask", ex.handleGetBestAsk)
	e.GET("/orders/:userID", ex.handleGetOrders)

	e.GET("/orderbook", ex.getOrderBook)
	e.GET("/balance", ex.getBalance)
	
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	fmt.Printf("%+v", client)

	e.Logger.Fatal(e.Start(":3004"))
}

type User struct {
	ID int64
	PrivateKey *ecdsa.PrivateKey
	Address  common.Address
}

func NewUser(privKey string, addr string, id int64) *User {
	address := common.HexToAddress(addr)

	pk, err := crypto.HexToECDSA(privKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID: id,
		PrivateKey: pk,
		Address: address,
	}
}

func (u *User) GetBalance(client *ethclient.Client, address common.Address) *big.Int {
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		log.Fatal(err)
	}

	return balance
}

func httpErrorHandler(err error, c echo.Context){
	fmt.Println(err)
}


/*
The format for the Exchange Orders is the following:

users: {
	UserID: {
		OrderID: { order },
		OrderID: { order },
		OrderID: { order },
	}
}
*/
type UserOrders struct {
	mu sync.RWMutex
	orderMap map[int64]map[int64]*orderbook.Order
}

type Exchange struct {
	Client *ethclient.Client
	Users map[int64]*User
	// Orders map[int64]map[int64]*orderbook.Order // Orders maps a user ID to a list of his orders
	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook

	// mu sync.RWMutex
	UserOrders // TODO: Maybe attach to the User struct instead of the exchange
}

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketETH] = orderbook.NewOrderbook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatal(err)
	}

	return &Exchange{
		Client: client,
		Users: make(map[int64]*User),
		// Orders: make(map[int64]map[int64]*orderbook.Order),
		UserOrders:     UserOrders{orderMap: make(map[int64]map[int64]*orderbook.Order)},
		PrivateKey: pk,
		orderbooks: orderbooks,
	}, nil
}

type GetOrdersResponse struct {
	Asks []*orderbook.Order
	Bids []*orderbook.Order
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userIDStr := c.Param("userID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return err
	}

	var userOrders []*orderbook.Order

	for _, val := range ex.orderMap[userID] {
		userOrders = append(userOrders, val)
	}

	ordersResp := &GetOrdersResponse{
		Asks: []*orderbook.Order{},
		Bids: []*orderbook.Order{},
	}

	for i := 0; i < len(userOrders); i++ {
		if userOrders[i].Bid {
			ordersResp.Bids = append(ordersResp.Bids, userOrders[i])
		} else {
			ordersResp.Asks = append(ordersResp.Asks, userOrders[i])
		}
	}

	return c.JSON(http.StatusOK, ordersResp)
	// return c.JSON(http.StatusOK, userOrders)
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match){
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)

	// matchedOrders := make([]*MatchedOrder, len(matches))

	// isBid := false

	// if order.Bid {
	// 	isBid = true
	// }

	totalSizeFilled := 0.0
	sumPrice := 0.0

	// Create a prices set
	pricesMap := make(map[float64]bool)

	for i := 0; i < len(matches); i++ {
	// for i := 0; i < len(matchedOrders); i++ {
		// id := matches[i].Bid.ID 
		// if isBid {
		// 	id = matches[i].Ask.ID
		// }
		// matchedOrders[i] = &MatchedOrder{
		// 	UserID: order.UserID,
		// 	ID: id,
		// 	Size: matches[i].SizeFilled,
		// 	Price: matches[i].Price,
		// }

		totalSizeFilled += matches[i].SizeFilled

		sumPrice += matches[i].Price

		// Add the price to the set
		pricesMap[matches[i].Price] = true
	}

	// Create an array from the SET
	var prices []float64
	for price := range pricesMap {
		prices = append(prices, price)
	}

	avgPrice := sumPrice / float64(len(matches))

	strOut := fmt.Sprintf("\nSERVER: Filled MARKET order: \n- UID: %v | Order ID: %d | Bid: %v | Size: %.2f | AvgPrice: %v | Prices: %v \n ", order.UserID, order.ID, order.Bid, totalSizeFilled, avgPrice, prices)

	fmt.Println(utils.PrintColor("blue", strOut))

	// return matches, matchedOrders
	return matches
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	ex.UserOrders.mu.Lock()
	defer ex.UserOrders.mu.Unlock()

	// Adds the order to a map at the exchange orders using the order ID as the key
	if ex.orderMap[order.UserID] == nil {
    ex.orderMap[order.UserID] = make(map[int64]*orderbook.Order)
	}


	ex.orderMap[order.UserID][order.ID] = order


	return nil
}

type PlaceOrderResponse struct {
	OrderID int64
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderuserOrders PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderuserOrders); err != nil {
		return err
	}

	market := Market(placeOrderuserOrders.Market)
	order := orderbook.NewOrder(placeOrderuserOrders.Bid, placeOrderuserOrders.Size, placeOrderuserOrders.UserID)

	// Limit orders
	if placeOrderuserOrders.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderuserOrders.Price, order); err != nil {
			return err
		}
	}
	
	// Market orders
	if placeOrderuserOrders.Type == MarketOrder {
		matches:= ex.handlePlaceMarketOrder(market, order)

		if err := ex.handleMatches(matches); err != nil {
			return err
		}
	}

	resp := &PlaceOrderResponse{
		OrderID: order.ID,
	}

	return c.JSON(200, resp)
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("User not found: %d", match.Ask.UserID)
		}

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("User not found: %d", match.Bid.UserID)
		}
		
		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		// TODO: Implement this - will be used to charge fees
		// exchangePubKey := ex.PrivateKey.Public()
		// publicKeyECDSA, ok := exchangePubKey.(*ecdsa.PublicKey)
		// if !ok {
		// 	return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		// }

		// TODO: Fix this typing issue
		amount := big.NewInt(int64(match.SizeFilled))

		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount, match.Ask.UserID, match.Bid.UserID)

		// Clear the Exchange order store if order is empty
		if match.Ask.Size == 0.0 {
			orderID := match.Ask.ID
			userID := match.Ask.UserID
			delete(ex.orderMap[userID], orderID)
		}

		if match.Bid.Size == 0.0 {
			orderID := match.Bid.ID
			userID := match.Bid.UserID
			delete(ex.orderMap[userID], orderID)
		}
	}

	return nil
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))

	ob, ok := ex.orderbooks[market]

	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookuserOrders := OrderBookuserOrders{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),

		Asks: []*Order{},
		Bids: []*Order{},
	}

	for _, limit := range ob.Asks(){
		for _, order := range limit.Orders {
			o := Order{
				UserID: order.UserID,
				ID: order.ID,
				Price: order.Limit.Price,
				Size: order.Size,
				Bid: order.Bid,
				Timestamp: order.Timestamp,
			}

			orderbookuserOrders.Asks = append(orderbookuserOrders.Asks, &o)
		}
	}

	for _, limit := range ob.Bids(){
		for _, order := range limit.Orders {
			o := Order{
				UserID: order.UserID,
				ID: order.ID,
				Price: order.Limit.Price,
				Size: order.Size,
				Bid: order.Bid,
				Timestamp: order.Timestamp,
			}

			orderbookuserOrders.Bids = append(orderbookuserOrders.Bids, &o)
		}
	}

	return c.JSON(http.StatusOK, orderbookuserOrders)
}

type CancelOrderRequest struct {
	Bid bool
	ID int64
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	ob := ex.orderbooks[MarketETH]
	order := ob.Orders[int64(id)]

	ob.CancelOrder(order)

	// fmt.Println("Order cancelled ")

	return c.JSON(200, map[string]any{ "msg": "order deleted" })
}

func (ex *Exchange) getOrderBook(c echo.Context) error {
	ob := ex.orderbooks[MarketETH]

	// return c.JSON(200, map[string]any{ jsonuserOrders})
	return c.JSON(200, ob.Orders)
}

func (ex *Exchange) getBalance(c echo.Context) error {
	balance1 := ex.Users[11].GetBalance(ex.Client, ex.Users[11].Address)
	fmt.Printf("Balance: %+v\n", balance1)

	balance2 := ex.Users[22].GetBalance(ex.Client, ex.Users[22].Address)
	fmt.Printf("Balance: %+v\n", balance2)

	balance3 := ex.Users[33].GetBalance(ex.Client, ex.Users[33].Address)
	fmt.Printf("Balance: %+v\n", balance3)

	return c.JSON(200, map[string]any{"1": balance1.String(), "2": balance2.String(), "3": balance3.String()})
}

// TODO: Change userOrders structure to use negative and positive ids for asks and bids]
func transferETH(client *ethclient.Client, fromPrivKey *ecdsa.PrivateKey, to common.Address, amount *big.Int, fromUserID int64, toUserID int64) error {
	ctx := context.Background()

	publicKey := fromPrivKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasLimit := uint64(21000) // in units

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

	chainID := big.NewInt(1337)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivKey)
	if err != nil {
		return err
	}

	client.SendTransaction(ctx, signedTx)

	fromBalance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
	
	toBalance, err := client.BalanceAt(ctx, to, nil)
	if err != nil {
		log.Fatal(err)
	}


	fbOut := fmt.Sprintf("Transferred %v \n- From: User [%v] | balance %+v \n- To:   User [%v] | balance %+v \n", amount, fromUserID, fromBalance, toUserID, toBalance)
	fmt.Println(utils.PrintColor("blue", fbOut))

	return client.SendTransaction(ctx, signedTx)
}

type PriceResponse struct {
	Price float64
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	market := Market(c.Param("market"))

	ob := ex.orderbooks[market]

	if len(ob.Bids()) == 0{ 
		return fmt.Errorf(utils.PrintColor("Blue", "SERVER: No bids to show!"))
	}
	bestBid := ob.Bids()[0]

	// DEBUGGING
	// str := fmt.Sprintf("SERVER: Best bid: %v", ob.Bids())
	// fmt.Println(utils.PrintColor("red", str))

	return c.JSON(http.StatusOK, bestBid)
}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {
	market := Market(c.Param("market"))

	ob := ex.orderbooks[market]

	if len(ob.Asks()) == 0{ 
		return fmt.Errorf(utils.PrintColor("Blue", "SERVER: No asks to show!"))
	}

	// DEBUGGING
	// str := fmt.Sprintf("SERVER: Best ask: %v", ob.Asks())
	// fmt.Println(utils.PrintColor("red", str))

	bestAsk := ob.Asks()[0]

	return c.JSON(http.StatusOK, bestAsk)
}