package orderbook

import (
	"fmt"
)


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