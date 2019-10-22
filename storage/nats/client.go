package nats

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dimuls/news-storage/entity"
	"github.com/dimuls/news-storage/storage/nats/pb"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
)

type Client struct {
	subSubj    string
	connection *nats.Conn
}

func NewClient(natsURL, subSubj string) (*Client, error) {
	conn, err := nats.Connect(natsURL, nats.DrainTimeout(5*time.Second))
	if err != nil {
		return nil, errors.New("failed to connect to nats: " + err.Error())
	}
	return &Client{
		subSubj:    subSubj,
		connection: conn,
	}, nil
}

func (c *Client) Close() {
	c.connection.Close()
}

func (c *Client) News(ctx context.Context, id int64) (entity.News, error) {
	req := &pb.GetNewsRequest{
		Id: id,
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return entity.News{}, errors.New(
			"failed to marshal *pb.GetNewsRequest: " + err.Error())
	}

	resMsg, err := c.connection.RequestWithContext(ctx, c.subSubj, reqBytes)
	if err != nil {
		return entity.News{}, errors.New("failed to request: " + err.Error())
	}

	var res pb.GetNewsResponse

	err = proto.Unmarshal(resMsg.Data, &res)
	if err != nil {
		return entity.News{}, errors.New("failed to unmarshal response: " +
			err.Error())
	}

	if res.Error != nil {
		if res.Error.Code == http.StatusNotFound {
			return entity.News{}, entity.ErrNewsNotFound
		}
		return entity.News{}, errors.New("got response error: " +
			res.Error.Message)
	}

	if res.News == nil {
		return entity.News{}, errors.New("unexpected nil news")
	}

	date, err := time.ParseInLocation("2006-01-02", res.News.Date,
		time.UTC)
	if err != nil {
		return entity.News{}, errors.New("failed to parse date: " + err.Error())
	}

	return entity.News{
		ID:     res.News.Id,
		Header: res.News.Header,
		Date:   date,
	}, nil
}
