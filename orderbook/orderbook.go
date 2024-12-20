package orderbook

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/kkomitski/exchange/utils"
)

type Trade struct {
    Price float64
    Bid bool
    Timestamp int64
    Size float64
}

type Match struct {
    Ask        *Order
    Bid        *Order
    SizeFilled float64
    Price      float64
}

type Order struct {
    ID        int64   `json:"id"`
    UserID    int64   `json:"userId"`
    Size      float64 `json:"size"`
    Bid       bool    `json:"bid"`
    Limit     *Limit  `json:"limit"`
    Timestamp int64   `json:"timestamp"`
}

type Orders []*Order

func (o Orders) Len() int { return len(o) }

func (o Orders) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o Orders) Less(i, j int) bool { return o[i].Timestamp < o[j].Timestamp }

func (o *Order) String() string {
    return fmt.Sprintf("Order{Size: %.2f, Bid: %v, Timestamp: %v}", o.Size, o.Bid, o.Timestamp)
}

func (o *Order) IsFilled() bool {
    return o.Size == 0.0
}

func NewOrder(bid bool, size float64, userID int64) *Order {
    return &Order{
        UserID:    userID,
        ID:        int64(rand.Intn(1_000_000_000)),
        Size:      size,
        Bid:       bid,
        Timestamp: time.Now().UnixNano(),
    }
}

type Limit struct {
    Price       float64 `json:"price"`
    Orders      Orders  `json:"-"`
    TotalVolume float64 `json:"totalVolume"`
}

type LimitJSON struct {
    Price       float64 `json:"price"`
    TotalVolume float64 `json:"totalVolume"`
    Orders      Orders  `json:"orders"`
}

type Limits []*Limit
type ByBestAsk struct{ Limits }

func (a ByBestAsk) Len() int { return len(a.Limits) }

func (a ByBestAsk) Swap(i, j int) { a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i] }

func (a ByBestAsk) Less(i, j int) bool { return a.Limits[i].Price < a.Limits[j].Price }

type ByBestBid struct{ Limits }

func (b ByBestBid) Len() int { return len(b.Limits) }

func (b ByBestBid) Swap(i, j int) { b.Limits[i], b.Limits[j] = b.Limits[j], b.Limits[i] }

func (b ByBestBid) Less(i, j int) bool { return b.Limits[i].Price > b.Limits[j].Price }

func (l *Limit) String() string {
    return fmt.Sprintf("[price: %.2f | volume: %.2f]", l.Price, l.TotalVolume)
}

func (l *Limit) AddOrder(o *Order) {
    o.Limit = l
    l.Orders = append(l.Orders, o)
    l.TotalVolume += o.Size
}

func (l *Limit) DeleteOrder(o *Order) {
    for i := 0; i < len(l.Orders); i++ {
        if l.Orders[i] == o {
            l.Orders[i] = l.Orders[len(l.Orders)-1]
            l.Orders = l.Orders[:len(l.Orders)-1]
        }
    }

    o.Limit = nil
    l.TotalVolume -= o.Size

    sort.Sort(l.Orders)
    // TODO: resort the rest of the orders
}

func (l *Limit) ProcessOrder(o *Order, ob *Orderbook) []Match {
    var (
        matches        []Match
        ordersToDelete []*Order
    )

    // Range over all the orders in the limit
    for _, order := range l.Orders {
        // If the incoming order is empty -> exit early
        if o.IsFilled() {
            break
        }
        
        match := l.fillOrder(order, o)
        matches = append(matches, match)

        l.TotalVolume -= match.SizeFilled

        // If the the Order sitting at this limit is filled -> delete
        if order.IsFilled() {
            ordersToDelete = append(ordersToDelete, order)
        }
    }

    for _, order := range ordersToDelete {
        l.DeleteOrder(order)

        // TODO: Check this 
        if ob.Orders[order.ID].Size == 0.0 {
            delete(ob.Orders, order.ID)
        }
    }

    // if ob.Orders[o.ID].Size == 0.0 {
    //     // fmt.Printf("Orderbook orders %+v", ob.Orders[o.ID])
    //     fmt.Println("here")
    // }

    return matches
}

type PrintParams struct {
    BidUserID int64
    AskUserID int64

    SizeFilled float64
}

func (l *Limit) fillOrder(a, b *Order) Match {
    var (
        bid        *Order
        ask        *Order
        sizeFilled float64
    )

    if a.Bid {
        bid = a
        ask = b
    } else {
        bid = b
        ask = a
    }

    if a.Size >= b.Size {
        a.Size -= b.Size
        sizeFilled = b.Size
        b.Size = 0.0
    } else {
        b.Size -= a.Size
        sizeFilled = a.Size
        a.Size = 0.0
    }

    return Match{
        Bid:        bid,
        Ask:        ask,
        SizeFilled: sizeFilled,
        Price:      l.Price,
    }
}

func NewLimit(price float64) *Limit {
    return &Limit{
        Price:       price,
        Orders:      []*Order{},
        TotalVolume: 0,
    }
}

type Orderbook struct {
    asks []*Limit
    bids []*Limit

    Trades []*Trade

    AskLimits map[float64]*Limit `json:"askLimits"`
    BidLimits map[float64]*Limit `json:"bidLimits"`

    Orders map[int64]*Order `json:"orders"`

    mu sync.RWMutex
}

func NewOrderbook() *Orderbook {
    return &Orderbook{
        asks:      []*Limit{},
        bids:      []*Limit{},

        Trades:    []*Trade{},

        AskLimits: make(map[float64]*Limit),
        BidLimits: make(map[float64]*Limit),
        Orders:    make(map[int64]*Order),
        // mu:       sync.RWMutex{},
    }
}

func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {
    matches := []Match{}

    if o.Bid {
        // Bid order
        if o.Size > ob.AskTotalVolume() {
            panic(fmt.Errorf("not enough volume [size: %.2f] sitting in books for order [size: %.2f]", ob.AskTotalVolume(), o.Size))
        }

        for _, limit := range ob.Asks() {
            limitMatches := limit.ProcessOrder(o, ob)
            // fmt.Println("Limit matches", matches)
            matches = append(matches, limitMatches...)

            if len(limit.Orders) == 0 {
                ob.clearLimit(false, limit)
            }
        }
    } else {
        // Ask order
        if o.Size > ob.BidTotalVolume() {
            panic(fmt.Errorf("not enough volume [size: %.2f] sitting in books for order [size: %.2f]", ob.BidTotalVolume(), o.Size))
        }

        for _, limit := range ob.Bids() {
            limitMatches := limit.ProcessOrder(o, ob)
            // fmt.Println("Limit matches", matches)
            matches = append(matches, limitMatches...)

            if len(limit.Orders) == 0 {
                ob.clearLimit(true, limit)
            }
        }
    }

    fmt.Println(utils.PrintColor("green", "OB: Orders Matched:"))
    for i := 0; i < len(matches); i++ {
        str := fmt.Sprintf("- Bid UID: %v | Ask UID: %v | SizeFilled: %.2f | Price: %.2f", matches[i].Bid.UserID, matches[i].Ask.UserID, matches[i].SizeFilled, matches[i].Price)

        fmt.Println(utils.PrintColor("green", str))
    }

    for _, match := range matches {
        trade := &Trade{
            Price: match.Price,
            Size: match.SizeFilled,
            Timestamp: time.Now().UnixNano(),
            Bid: o.Bid,
        }

        ob.Trades = append(ob.Trades, trade)
    }

    return matches
}

func (ob *Orderbook) PlaceLimitOrder(price float64, o *Order) {
    var limit *Limit

    ob.mu.Lock()
    defer ob.mu.Unlock()

    // fmt.Println("Adding order", o)
    if o.Bid {
        limit = ob.BidLimits[price]
    } else {
        limit = ob.AskLimits[price]
    }

    // fmt.Println("limit", limit)

    if limit == nil {
        limit = NewLimit(price)
        // fmt.Printf("Created new limit at %+v \n\n", limit.Price)

        if o.Bid {
            // fmt.Println("bid")
            ob.bids = append(ob.bids, limit)
            ob.BidLimits[price] = limit
        } else {
            // fmt.Println("ask")
            ob.asks = append(ob.asks, limit)
            ob.AskLimits[price] = limit
        }
    }

    ob.Orders[o.ID] = o
    limit.AddOrder(o)
}

func (ob *Orderbook) clearLimit(bid bool, l *Limit) {
    if bid {
        delete(ob.BidLimits, l.Price)

        for i := 0; i < len(ob.bids); i++ {
            if ob.bids[i] == l {
                ob.bids[i] = ob.bids[len(ob.bids)-1]
                ob.bids = ob.bids[:len(ob.bids)-1]
            }
        }
    } else {
        delete(ob.AskLimits, l.Price)
 
        for i := 0; i < len(ob.asks); i++ {
            if ob.asks[i] == l {
                ob.asks[i] = ob.asks[len(ob.asks)-1]
                ob.asks = ob.asks[:len(ob.asks)-1]
            }
        }
    }

    /**
    *   DEBUGGING
    *   Uncomment this block to see the orderbook being cleared
    */
    // var orderType string
    // if bid {
    //     orderType = "Bid"
    //     str := fmt.Sprintf("OB: Cleared %v limit at price %.2f. \n- Bids Limits: [%v] %+v \n", orderType, l.Price, len(ob.bids), ob.bids)
    //     fmt.Println(utils.PrintColor("green", str))
    // } else {
    //     orderType = "Ask"
    //     str := fmt.Sprintf("OB: Cleared %v limit at price %.2f. \n- Asks Limits: [%v] %+v \n", orderType, l.Price, len(ob.asks), ob.asks)
    //     fmt.Println(utils.PrintColor("green", str))
    // }
}

func (ob *Orderbook) CancelOrder(o *Order) {
    limit := o.Limit
    limit.DeleteOrder(o)

    if len(limit.Orders) == 0 {
        ob.clearLimit(o.Bid, limit)
    }

    delete(ob.Orders, o.ID)
    fmt.Println("\n Cancelled order with id", o.ID)
}

func (ob *Orderbook) BidTotalVolume() float64 {
    totalVolume := 0.0

    for i := 0; i < len(ob.bids); i++ {
        totalVolume += ob.bids[i].TotalVolume
    }

    return totalVolume
}

func (ob *Orderbook) AskTotalVolume() float64 {
    totalVolume := 0.0

    for i := 0; i < len(ob.asks); i++ {
        totalVolume += ob.asks[i].TotalVolume
    }

    return totalVolume
}

func (ob *Orderbook) Asks() []*Limit {
    sort.Sort(ByBestAsk{ob.asks})

    return ob.asks
}

func (ob *Orderbook) Bids() []*Limit {
    sort.Sort(ByBestBid{ob.bids})

    return ob.bids
}