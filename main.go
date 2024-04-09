package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kkomitski/exchange/client"
	"github.com/kkomitski/exchange/server"
	"github.com/kkomitski/exchange/utils"
	"github.com/labstack/gommon/log"
)

const (
	maxOrders = 3
)

var (
	tick = 1 * time.Millisecond
	myAsks = make(map[float64]int64)
	myBids = make(map[float64]int64)
)
// BID - desire to BUY
// ASK - desire to SELL

func seedMarket(c *client.Client) error {
	// ASK - desire to SELL
	ask := &client.PlaceOrderParams{
		UserID: 22,
		Bid:    false,
		Price:  11000,
		Size:   1000,
	}

	// BID - desire to BUY
	bid := &client.PlaceOrderParams{
		UserID: 22,
		Bid:    true,
		Price:  10000,
		Size:   1000,
	}

	MakeOrder(c, "LIMIT", ask.UserID, ask.Bid, ask.Price, ask.Size)
	MakeOrder(c, "LIMIT", bid.UserID, bid.Bid, bid.Price, bid.Size)
	MakeOrder(c, "LIMIT", ask.UserID, ask.Bid, ask.Price, ask.Size)
	MakeOrder(c, "LIMIT", bid.UserID, bid.Bid, bid.Price, bid.Size)

	return nil
}

func makeMarketSimple(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		orders, err := c.GetOrders(22)
		if err != nil {
			panic(err)
		}

		bestAsk, err := c.GetBestAsk()
		if err != nil {
			// panic(err)
			fmt.Println(err)
		}
		
		bestBid, err := c.GetBestBid()
		if err != nil {
			// panic(err)
			fmt.Println(err)
		}

		// Get the spread
		spread := math.Abs(bestBid - bestAsk)
		fmt.Println("Spread: ", spread)

		// Create new bid (buy) limits
		if len(orders.Bids) < 3 {
			bidLimit := &client.PlaceOrderParams{
				UserID: 22,
				Bid: true,
				Price: bestBid + 100,
				Size: 1000,
			}

			bidOrderResp, _ := MakeOrder(c, "LIMIT", bidLimit.UserID, bidLimit.Bid, bidLimit.Price, bidLimit.Size)
			myBids[bidLimit.Price] = bidOrderResp.OrderID

			// fmt.Println("Bid order placed: ", bidLimit.Price)
		}

		// Create new ask (sell) limits
		if len(orders.Asks) < 3 {
			askLimit := &client.PlaceOrderParams{
				UserID: 22,
				Bid: false,
				Price: bestAsk - 100,
				Size: 1000,
			}
			askOrderResp, _ := MakeOrder(c, "LIMIT", askLimit.UserID, askLimit.Bid, askLimit.Price, askLimit.Size)
			myAsks[askLimit.Price] = askOrderResp.OrderID

			// fmt.Println("Ask order placed: ", askLimit.Price)
		}

		fmt.Println("Best ask price:", bestAsk)
		fmt.Println("Best bid price:", bestBid)

		<- ticker.C
	}
}

func marketOrderPlacer(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		buy := &client.PlaceOrderParams{
			UserID: 33,
			Bid:    true,
			Size:   1000,
		}

		MakeOrder(c, "MARKET", buy.UserID, buy.Bid, buy.Price, buy.Size)

		sell := &client.PlaceOrderParams{
			UserID: 33,
			Bid:    false,
			Size:   1000,
		}

		MakeOrder(c, "MARKET", sell.UserID, sell.Bid, sell.Price, sell.Size)

		resp, err := c.GetOrders(22)
		if err != nil {
			log.Error(err)
			panic(err)
		}

		fmt.Printf("Response 22 %+v\n\n", resp)

		for i := 0; i < len(resp.Asks); i++ {
			if resp.Asks[i].Size == 0.0 || resp.Asks[i].Size == 0 {
				panic("ZERO SIZE")
			}
		}

		for i := 0; i < len(resp.Bids); i++ {
			if resp.Bids[i].Size == 0.0 || resp.Bids[i].Size == 0 {
				panic("ZERO SIZE")
			}
		}
		
		resp, err = c.GetOrders(11)
		if err != nil {
			log.Error(err)
			panic(err)
		}

		for i := 0; i < len(resp.Asks); i++ {
			if resp.Asks[i].Size == 0.0 || resp.Asks[i].Size == 0 {
				panic("ZERO SIZE")
			}
		}

		for i := 0; i < len(resp.Bids); i++ {
			if resp.Bids[i].Size == 0.0 || resp.Bids[i].Size == 0 {
				panic("ZERO SIZE")
			}
		}

		fmt.Printf("Response 11 %+v", resp)

		<-ticker.C
	}
}

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	// MakeOrder(c, "LIMIT", 22, false, 	1 , 5)
	// MakeOrder(c, "LIMIT", 22, false, 	2 , 5)
	// MakeOrder(c, "LIMIT", 22, false, 	3 , 5)
	// MakeOrder(c, "MARKET", 33, true, 	0, 15)

	// for i := 0; i < 4; i++ {
	// 	MakeOrder(c, "LIMIT", 11, true, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// }
	if err := seedMarket(c); err != nil {
		panic(err)
	}

	go makeMarketSimple(c)

	time.Sleep(1 * time.Second)

	marketOrderPlacer(c)
	

	// if err := seedMarket(c); err != nil {
	// 	panic(err)
	// }
	// if err := seedMarket(c); err != nil {
	// 	panic(err)
	// }

	// for i := 0; i < 4; i++ {
	// 	MakeOrder(c, "LIMIT", 11, true, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// 	MakeOrder(c, "LIMIT", 22, false, 1000 * float64(i + 1), 1000)
	// }

	// MakeOrder(c, "LIMIT", 22, true, 11000, 1000)
	// MakeOrder(c, "LIMIT", 22, true, 9000, 1000)
	// MakeOrder(c, "LIMIT", 11, false, 12000, 1000)

	// sell := &client.PlaceOrderParams{
	// 	UserID: 11,
	// 	Bid:    false,
	// 	Size:   2000,
	// }

	// MakeOrder(c, "MARKET", 11, false, sell.Price, sell.Size)
	// MakeOrder(c, "MARKET", 33, true, sell.Price, sell.Size)

	// fmt.Printf("\n\nResponse 11 %+v", resp)
	
	// go makeMarketSimple(c)

	// time.Sleep(1 * time.Second)
	// marketOrderPlacer(c)

	select{}
}

func TEST_PutLimitOrdersInBooks(c *client.Client, bidOrders int, askOrders int ) {
	for i := 0; i < bidOrders; i++ {
		limitOrderParamsA := &client.PlaceOrderParams{
			UserID: 11,
			Bid:    true,
			Price:  1_000 * (float64(i) + 1),
			Size:   1,
		}

		limitOrderOut := fmt.Sprintf("CLIENT: Placing LIMIT order: \n UID: %v | Bid: %v | Price: %.2f | Size: %.2f \n", limitOrderParamsA.UserID, limitOrderParamsA.Bid, limitOrderParamsA.Price, limitOrderParamsA.Size)
		fmt.Println(utils.PrintColor("yellow", limitOrderOut))
		
		_, err := c.PlaceLimitOrder(limitOrderParamsA)
		if err != nil {
			panic(err)
		}

		// time.Sleep(1 * time.Second)
	}

	for i := 0; i < askOrders; i++ {
		limitOrderParamsA := &client.PlaceOrderParams{
			UserID: 22,
			Bid:    false,
			Price:  1_000 * (float64(i) + 1),
			Size:   1,
		}

		limitOrderOut := fmt.Sprintf("CLIENT: Placing LIMIT order: \n UID: %v | Bid: %v | Price: %.2f | Size: %.2f \n", limitOrderParamsA.UserID, limitOrderParamsA.Bid, limitOrderParamsA.Price, limitOrderParamsA.Size)
		fmt.Println(utils.PrintColor("yellow", limitOrderOut))
		
		_, err := c.PlaceLimitOrder(limitOrderParamsA)
		if err != nil {
			panic(err)
		}

		// time.Sleep(1 * time.Second)
	}
}

func TEST_PutMarketOrdersInBooks(c *client.Client, bidOrders int, askOrders int){
	for i := 0; i < bidOrders; i++ {
		marketOrderParams := &client.PlaceOrderParams{
			UserID: 33,
			Bid:    true,
			Size:   1,
		}
	
		marketOrderOut := fmt.Sprintf("CLIENT: Placing MARKET order: \n UID: %v | Bid: %v | Size: %.2f \n", marketOrderParams.UserID, marketOrderParams.Bid, marketOrderParams.Size)
		fmt.Println(utils.PrintColor("yellow", marketOrderOut))
	
		_, err := c.PlaceMarketOrder(marketOrderParams)
		if err != nil {
			panic(err)
		}
	
		// time.Sleep(1 * time.Second)
	}

	for i := 0; i < askOrders; i++ {
		marketOrderParamsB := &client.PlaceOrderParams{
			UserID: 33,
			Bid:    false,
			Size:   1,
		}

		marketOrderOutB := fmt.Sprintf("CLIENT: Placing MARKET order: \n UID: %v | Bid: %v | Size: %.2f \n", marketOrderParamsB.UserID, marketOrderParamsB.Bid, marketOrderParamsB.Size)
		fmt.Println(utils.PrintColor("yellow", marketOrderOutB))

		_, err := c.PlaceMarketOrder(marketOrderParamsB)
		if err != nil {
			panic(err)
		}

		// time.Sleep(1 * time.Second)
	}
}

func TEST_CancelOrders(c *client.Client, orderIDs []int64) {
	for i := 0; i < len(orderIDs); i++ {
		if err := c.CancelOrder(orderIDs[i]); err != nil {
			fmt.Println("Failed to cancel")
			panic(err)
		}
		fmt.Println("Cancelling =>", orderIDs[i])

		time.Sleep(1 * time.Second)
	}
}

func MakeOrder(c *client.Client, OrderType string, UserID int64, Bid bool, Price float64, Size float64) (*server.PlaceOrderResponse, error) {
	op := &client.PlaceOrderParams{
		UserID: UserID,
		Bid: Bid,
		Price: Price,
		Size: Size,
	}

	if OrderType == "MARKET" || OrderType == "LIMIT" {
		// MARKET ORDER
		if OrderType == "MARKET" {
			resp, err := c.PlaceMarketOrder(op)
			if err != nil {
				log.Errorf("Failed to place market order: %v", err)
				return nil, err
			}

			marketOrderOut := fmt.Sprintf("CLIENT: Placing %v order: \n UID: %v | Bid: %v | Size: %.2f | Price %.2f\n", OrderType, UserID, Bid, Size, Price)
			fmt.Println(utils.PrintColor("yellow", marketOrderOut))

			return resp, nil
		// LIMIT ORDER
		} else if OrderType == "LIMIT" {
			resp, err := c.PlaceLimitOrder(op)
			if err != nil {
				log.Errorf("Failed to place limit order: %v", err)
				return nil, err
			}

			marketOrderOut := fmt.Sprintf("CLIENT: Placing %v order: \n UID: %v | Bid: %v | Size: %.2f | Price %.2f\n", OrderType, UserID, Bid, Size, Price)
			fmt.Println(utils.PrintColor("yellow", marketOrderOut))

			return resp, nil
		} 
	} else {
		panic("Invalid order type")
	}

	return nil, errors.New("failed to place order")
}
/**
*   LIMIT ORDER
*/
// limitOrderParamsA := &client.PlaceOrderParams{
// 	UserID: 11,
// 	Bid:    true,
// 	Price:  1_000 * float64(i + 1),
// 	Size:   1,
// }

// limitOrderOut := fmt.Sprintf("CLIENT: Placing LIMIT order: \n UID: %v | Bid: %v | Price: %.2f | Size: %.2f \n", limitOrderParamsA.UserID, limitOrderParamsA.Bid, limitOrderParamsA.Price, limitOrderParamsA.Size)
// fmt.Println(utils.PrintColor("yellow", limitOrderOut))

// _, err := c.PlaceLimitOrder(limitOrderParamsA)
// if err != nil {
// 	panic(err)
// }


/**
*   MARKET ORDER
*/
// // If bid is true => buy low (only most expensive ask left)
// // If bid is false => sell high (only cheapest bid left)
// marketOrderParams := &client.PlaceOrderParams{
// 	UserID: 33,
// 	Bid:    false,
// 	Size:   2,
// }

// marketOrderOut := fmt.Sprintf("CLIENT: Placing MARKET order: \n UID: %v | Bid: %v | Size: %.2f \n", marketOrderParams.UserID, marketOrderParams.Bid, marketOrderParams.Size)
// fmt.Println(utils.PrintColor("yellow", marketOrderOut))

// _, err := c.PlaceMarketOrder(marketOrderParams)
// if err != nil {
// 	panic(err)
// }