package orderbook

import (
	"fmt"
	"sort"
)

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
