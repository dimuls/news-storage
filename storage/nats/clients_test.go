package nats

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dimuls/news-storage/entity"
	"github.com/dimuls/news-storage/storage/nats/pb"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func initClient(t *testing.T) *Client {
	c, err := NewClient(os.Getenv("TEST_NATS_URL"),
		os.Getenv("TEST_SUBSCRIBE_SUBJECT"))
	if err != nil {
		t.Fatal("failed to create client: " + err.Error())
	}
	return c
}

func decodeGetNewsRequest(t *testing.T, reqBytes []byte) *pb.GetNewsRequest {
	var req pb.GetNewsRequest

	err := proto.Unmarshal(reqBytes, &req)
	if err != nil {
		t.Fatal("failed to unmarshal *pb.GetNewsRequest: " + err.Error())
	}

	return &req
}

func newGetNewsResponse(t *testing.T, res *pb.GetNewsResponse) []byte {
	resBytes, err := proto.Marshal(res)
	if err != nil {
		t.Fatal("failed to marshal *pb.GetNewsResponse: " + err.Error())
	}
	return resBytes
}

func TestClient_News_serverError(t *testing.T) {
	c := initClient(t)
	defer c.Close()

	sub, err := c.connection.SubscribeSync(c.subSubj)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = sub.Unsubscribe()
		if err != nil {
			t.Fatal("failed to unsubscribe: " + err.Error())
		}
	}()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()

		_, err := c.News(ctx, 123)
		if assert.Error(t, err) {
			assert.Equal(t, err.Error(), "got response error: foobar")
		}
	}()

	msg, err := sub.NextMsg(2 * time.Second)
	if !assert.NoError(t, err) {
		return
	}

	req := decodeGetNewsRequest(t, msg.Data)
	assert.Equal(t, int64(123), req.Id)

	err = msg.Respond(newGetNewsResponse(t, &pb.GetNewsResponse{
		Error: &pb.Error{
			Code:    234,
			Message: "foobar",
		},
	}))
	if !assert.NoError(t, err) {
		return
	}

	wg.Wait()
}

func TestClient_News_notFound(t *testing.T) {
	c := initClient(t)
	defer c.Close()

	sub, err := c.connection.SubscribeSync(c.subSubj)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = sub.Unsubscribe()
		if err != nil {
			t.Fatal("failed to unsubscribe: " + err.Error())
		}
	}()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()

		_, err := c.News(ctx, 123)
		assert.Equal(t, entity.ErrNewsNotFound, err)
	}()

	msg, err := sub.NextMsg(2 * time.Second)
	if !assert.NoError(t, err) {
		return
	}

	req := decodeGetNewsRequest(t, msg.Data)
	assert.Equal(t, int64(123), req.Id)

	err = msg.Respond(newGetNewsResponse(t, &pb.GetNewsResponse{
		Error: &pb.Error{
			Code:    http.StatusNotFound,
			Message: "foobar",
		},
	}))
	if !assert.NoError(t, err) {
		return
	}

	wg.Wait()
}

func TestClient_News_success(t *testing.T) {
	c := initClient(t)
	defer c.Close()

	sub, err := c.connection.SubscribeSync(c.subSubj)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = sub.Unsubscribe()
		if err != nil {
			t.Fatal("failed to unsubscribe: " + err.Error())
		}
	}()

	testDate, _ := time.Parse("2006-01-02", "2006-01-02")

	wantN := entity.News{
		ID:     123,
		Header: "header",
		Date:   testDate,
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()

		gotN, err := c.News(ctx, 123)
		if assert.NoError(t, err) {
			assert.True(t, cmp.Equal(wantN, gotN))
		}
	}()

	msg, err := sub.NextMsg(2 * time.Second)
	if !assert.NoError(t, err) {
		return
	}

	req := decodeGetNewsRequest(t, msg.Data)
	assert.Equal(t, int64(123), req.Id)

	err = msg.Respond(newGetNewsResponse(t, &pb.GetNewsResponse{
		News: &pb.News{
			Id:     123,
			Header: "header",
			Date:   testDate.UTC().Format("2006-01-02"),
		},
	}))
	if !assert.NoError(t, err) {
		return
	}

	wg.Wait()
}
