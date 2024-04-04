package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kkomitski/exchange/orderbook"
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

	OrderBookData struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks []*Order
		Bids []*Order
	}
	
	MatchedOrder struct {
		Price float64
		Size float64
		ID int64
	}	
)

func main() {
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

	pk1 := "dae942a9c73dc12d651d16599b8152908dc97a1fc2a87d02d003925c46ef7bcd"
	user1 := NewUser(pk1, 11)
	pk2 := "be14349a1085afff69328b026f4060027efec31e9b0d2db0e72bad2d507d6746"
	user2 := NewUser(pk2, 22)
	pk3 := "c79bdd25d28eac1b89ad8fe0322acf4c5049562bf81c695aa9d304fd69311c84"
	user3 := NewUser(pk3, 33)

	ex.Users[user1.ID] = user1
	ex.Users[user2.ID] = user2
	ex.Users[user3.ID] = user3

	e.GET("/book/:market", ex.handleGetBook)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)
	e.GET("/orderbook", ex.getOrderBook)

	fmt.Printf("%+v", client)

	// ctx := context.Background()

	// privateKey, err := crypto.HexToECDSA("dae942a9c73dc12d651d16599b8152908dc97a1fc2a87d02d003925c46ef7bcd")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// publicKey := privateKey.Public()
	// publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	// if !ok {
	// 	log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	// }

	// fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// nonce, err := client.PendingNonceAt(ctx, fromAddress)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// value := big.NewInt(1000000000000000000) // in wei (1 eth)
	// gasLimit := uint64(21000) // in units

	// gasPrice, err := client.SuggestGasPrice(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// toAddress := common.HexToAddress("0x1dF62f291b2E969fB0849d99D9Ce41e2F137006e")

	// tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

	// chainID := big.NewInt(1337)

	// signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = client.SendTransaction(ctx, signedTx)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// balance, err := client.BalanceAt(ctx, toAddress, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// _ = balance

	e.Logger.Fatal(e.Start(":3004"))
}

type User struct {
	ID int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(privKey string, id int64) *User {
	// pkStr := "dae942a9c73dc12d651d16599b8152908dc97a1fc2a87d02d003925c46ef7bcd"
	// pk, _ := crypto.HexToECDSA(privKey)

	// user := &User{
	// 	ID: id,
	// 	PrivateKey: pk,
	// }

	pk, err := crypto.HexToECDSA(privKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID: id,
		PrivateKey: pk,
	}
}

func httpErrorHandler(err error, c echo.Context){
	fmt.Println(err)
}

type Exchange struct {
	Client *ethclient.Client
	Users map[int64]*User
	orders map[int64]int64

	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook
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
		orders: make(map[int64]int64),

		PrivateKey: pk,
		orderbooks: orderbooks,
	}, nil
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder){
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)

	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false

	if order.Bid {
		isBid = true
	}

	for i := 0; i < len(matchedOrders); i++ {
		id := matches[i].Bid.ID 
		if isBid {
			id = matches[i].Ask.ID
		}
		matchedOrders[i] = &MatchedOrder{
			ID: id,
			Size: matches[i].SizeFilled,
			Price: matches[i].Price,
		}
	}

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	return nil
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	if placeOrderData.Type == LimitOrder {
		ex.handlePlaceLimitOrder(market, placeOrderData.Price, order)

		return c.JSON(200, map[string]any{"msg": "limit order placed"})
	}
	
	if placeOrderData.Type == MarketOrder {
		matches, matchedOrders := ex.handlePlaceMarketOrder(market, order)

		if err := ex.handleMatches(matches); err != nil {
			return err
		}

		return c.JSON(200, map[string]any{"matches": matchedOrders})
	}
	
	return c.JSON(http.StatusBadRequest, map[string]any{"msg": "no such order type"})
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fmt.Printf("Match: %+v\n", match)
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

		amount := big.NewInt(int64(match.SizeFilled))

		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}

	return nil
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))

	ob, ok := ex.orderbooks[market]

	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderBookData{
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

			orderbookData.Asks = append(orderbookData.Asks, &o)
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

			orderbookData.Bids = append(orderbookData.Bids, &o)
		}
	}

	return c.JSON(http.StatusOK, orderbookData)
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

	return c.JSON(200, map[string]any{ "msg": "order deleted" })
}

// func (ex *Exchange) handleMatches(matches []orderbook.Match) error {

// }

// type Bids struct {
// 	Price float64
// 	Size float64

// }

func (ex *Exchange) getOrderBook(c echo.Context) error {
	ob := ex.orderbooks[MarketETH]
	fmt.Println("here")
	fmt.Printf("Orderbook: %+v\n", ob.Orders)

	jsonData, err := json.Marshal(ob.Orders)
	if err != nil {
		log.Fatal(err)
	}

	return c.JSON(200, map[string]any{ "ob": jsonData})
}

// TODO: Change data structure to use negative and positive ids for asks and bids