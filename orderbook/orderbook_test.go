package orderbook

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {
	if !reflect.DeepEqual(a,b){
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimit(t *testing.T) {
	ob := NewOrderbook()
	l := NewLimit(10_000)

	buyOrderA := ob.NewOrder(true, 5, 0)
	buyOrderB := ob.NewOrder(true, 8, 0)
	buyOrderC := ob.NewOrder(true, 10, 0)

	l.AddOrder(buyOrderA)
	fmt.Println(l)
	l.AddOrder(buyOrderB)
	fmt.Println(l)
	l.AddOrder(buyOrderC)

	l.DeleteOrder(buyOrderB)

	fmt.Printf("hey %v", l)
}

func TestOrderbook(t *testing.T) {
	ob := NewOrderbook()

	buyOrder := ob.NewOrder(true, 10, 0)
	buyOrderA := ob.NewOrder(true, 2000, 0)

	ob.PlaceLimitOrder(18_000, buyOrder)
	ob.PlaceLimitOrder(16_000, buyOrderA)

	fmt.Println("ob", ob)

	// fmt.Println("\n Total Bid Orders:", ob.Bids[0].Orders)
	for i := 0; i < len(ob.bids); i++ {
		fmt.Printf("\n Bid Price %v: %+v", i, ob.bids[i].Orders)
	}
}

func TestPlaceLimitOrder(t *testing.T){
	ob := NewOrderbook()

	sellOrder := ob.NewOrder(false, 10, 0)
	sellOrderA := ob.NewOrder(false, 2000, 0)

	ob.PlaceLimitOrder(10_000, sellOrder)
	ob.PlaceLimitOrder(12_000, sellOrderA)

	fmt.Println("Total Ask volume:", ob.AskTotalVolume())
	fmt.Println("Total Bid volume:", ob.BidTotalVolume())

	assert(t, len(ob.Orders), 2)
	assert(t, ob.Orders[sellOrder.ID], sellOrder)
	assert(t, ob.Orders[sellOrderA.ID], sellOrderA)
	assert(t, len(ob.asks), 2)
}

func TestPlaceMarketOrder(t *testing.T){
	ob := NewOrderbook()

	// sellOrder := NewOrder(false, 20)
	sellOrderA := ob.NewOrder(false, 20, 0)
	
	// ob.PlaceLimitOrder(10_000, sellOrder)
	ob.PlaceLimitOrder(10_000, sellOrderA)

	// buyOrder := NewOrder(true, 10)
	buyOrderA := ob.NewOrder(true, 10, 0)

	// ob.PlaceMarketOrder(buyOrder)
	matches := ob.PlaceMarketOrder(buyOrderA)

	assert(t, len(matches), 1)
	assert(t, len(ob.asks), 1)
	assert(t, ob.AskTotalVolume(), 10.0)
	assert(t, matches[0].Ask, sellOrderA)
	assert(t, matches[0].Bid, buyOrderA)
	assert(t, buyOrderA.IsFilled(), true)
	
	fmt.Printf("%+v", matches)
}

func TestPlaceMarketOrderMultifill(t *testing.T){
	ob := NewOrderbook()

	buyOrderA := ob.NewOrder(true, 5, 0)
	buyOrderB := ob.NewOrder(true, 8, 0)
	buyOrderC := ob.NewOrder(true, 10, 0)
	buyOrderD := ob.NewOrder(true, 1, 0)
	
	ob.PlaceLimitOrder(5_000, buyOrderC)
	ob.PlaceLimitOrder(10_000, buyOrderA)
	ob.PlaceLimitOrder(9_000, buyOrderB)
	ob.PlaceLimitOrder(5_000, buyOrderD)

	assert(t, ob.BidTotalVolume(), 24.00)

	sellOrder := ob.NewOrder(false, 20, 0)
	matches := ob.PlaceMarketOrder(sellOrder)

	assert(t, ob.BidTotalVolume(), 4.00)
	assert(t, len(matches), 3)
	assert(t, len(ob.bids), 1)

	fmt.Println("\n\n bid limits:", ob.BidLimits)
	fmt.Println("\n top limit:", ob.Bids()[0].Price)
	fmt.Println("\n bottom limit:", ob.Bids()[len(ob.Bids()) - 1].Price)
}

func TestCancelOrderBid(t *testing.T){
	ob := NewOrderbook()

	buyOrder := ob.NewOrder(true, 4, 22)
	price := 10_000.0

	ob.PlaceLimitOrder(price, buyOrder)

	assert(t, ob.BidTotalVolume(), 4.0)
	assert(t, len(ob.bids), 1)
	
	ob.CancelOrder(buyOrder)
	assert(t, ob.BidTotalVolume(), 0.0)
	assert(t, len(ob.bids), 0)

	_, ok := ob.Orders[buyOrder.ID]
	assert(t, ok, false)

	_, ok = ob.BidLimits[price]
	assert(t, ok, false)
}

func TestCancelOrderAsk(t *testing.T){
	ob := NewOrderbook()

	sellOrder := ob.NewOrder(false, 4, 11)
	price := 10_000.0

	ob.PlaceLimitOrder(price, sellOrder)

	assert(t, ob.AskTotalVolume(), 4.0)
	assert(t, len(ob.asks), 1)
	
	ob.CancelOrder(sellOrder)
	assert(t, ob.AskTotalVolume(), 0.0)
	assert(t, len(ob.asks), 0)

	_, ok := ob.Orders[sellOrder.ID]
	assert(t, ok, false)

	_, ok = ob.AskLimits[price]
	assert(t, ok, false)
}

func TestLastMarketTrades(t *testing.T){
	ob := NewOrderbook()
	price := 10_000.0

	sellOrder := ob.NewOrder(false, 10, 0)
	ob.PlaceLimitOrder(price, sellOrder)

	marketOrder := ob.NewOrder(true, 10, 0)

	matches := ob.PlaceMarketOrder(marketOrder)
	assert(t, len(matches), 1)
	match := matches[0]

	assert(t, len(ob.Trades), 1)

	trade := ob.Trades[0]
	assert(t, trade.Price, price)
	assert(t, trade.Bid, marketOrder.Bid)
	assert(t, trade.Size, match.SizeFilled)
}