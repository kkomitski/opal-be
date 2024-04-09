package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kkomitski/exchange/server"
)

const Endpoint = "http://localhost:3004"

type Client struct {
	*http.Client
	cancelledOrders int64
}

func NewClient() *Client {
	return &Client{
		Client: http.DefaultClient,
	}
}

type PlaceOrderParams struct {
	UserID int64
	Bid bool
	Price float64 // only needed for LIMIT orders
	Size float64
}

// Shows the *LOWEST* price someone is willing to pay to *SELL* an asset for
func (c *Client) GetBestAsk() (float64, error){
	e := Endpoint + "/book/ETH/ask"

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}

	priceResp := &server.PriceResponse{}
	if err := json.NewDecoder(resp.Body).Decode(priceResp); err != nil {
		return 0, err
	}

	return priceResp.Price, nil
}

// Shows the *HIGHEST* price someone is willing to pay to *BUY* an asset
func (c *Client) GetBestBid() (float64, error){
	e := Endpoint + "/book/ETH/bid"

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}

	priceResp := &server.PriceResponse{}
	if err := json.NewDecoder(resp.Body).Decode(priceResp); err != nil {
		return 0, err
	}

	return priceResp.Price, nil
}

func (c *Client) GetOrders(userID int64) (*server.GetOrdersResponse, error) {
	e := Endpoint + "/orders/" + strconv.FormatInt(userID, 10)

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	orders := &server.GetOrdersResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, err
	}

	// DEBUGGING
	// Uncomment to print out the orders for the user id
	// str := fmt.Sprintf("CLIENT: Orders in exchange for User [%v]: \n- %+v \n",userID, orders)
	// fmt.Println(utils.PrintColor("yellow", str))
	// for i := 0; i < len(orders); i++ {
	// 	str = fmt.Sprintf("- %+v", orders[i])
	// 	fmt.Println(utils.PrintColor("yellow", str))
	// }

	return orders, nil
}

func (c *Client) PlaceMarketOrder(p *PlaceOrderParams) (*server.PlaceOrderResponse, error) {
	params := &server.PlaceOrderRequest{
		UserID: p.UserID,
		Type: server.MarketOrder,
		Bid: p.Bid,
		Size: p.Size,
		Market: server.MarketETH,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	e := Endpoint + "/order"

	req, err := http.NewRequest("POST", e, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	placeOrderResponse := &server.PlaceOrderResponse{}

	if err := json.NewDecoder(resp.Body).Decode(placeOrderResponse); err != nil {
		return nil, err
	}

	// fmt.Printf("\nreq: \n%+v\n\n", req)
	// fmt.Printf("res: \n%+v\n\n", resp)

	return placeOrderResponse, nil
}

func (c *Client) PlaceLimitOrder(p *PlaceOrderParams) (*server.PlaceOrderResponse, error) {
	params := &server.PlaceOrderRequest{
		UserID: p.UserID,
		Type: server.LimitOrder,
		Bid: p.Bid,
		Size: p.Size,
		Price: p.Price,
		Market: server.MarketETH,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	e := Endpoint + "/order"

	req, err := http.NewRequest("POST", e, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	placeOrderResponse := &server.PlaceOrderResponse{}

	if err := json.NewDecoder(resp.Body).Decode(placeOrderResponse); err != nil {
		return nil, err
	}

	return placeOrderResponse, nil
}

func (c *Client) CancelOrder(orderID int64) error {
	e := Endpoint + "/order/" + strconv.FormatInt(orderID, 10)

	req, err := http.NewRequest(http.MethodDelete, e, nil)
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if err != nil {
		return err
	}

	c.cancelledOrders++
	fmt.Printf("\n\n Total cancelled orders: %v \n\n", c.cancelledOrders)
	return nil
}

func (c *Client) GetTrades(market string) (*server.GetTradesResponse , error) {
	e := Endpoint + "/trades/" + market

	req, err := http.NewRequest(http.MethodGet, e, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	trades := &server.GetTradesResponse{}
	if err := json.NewDecoder(resp.Body).Decode(trades); err != nil {
		return nil, err
	}

	return trades, nil
}	