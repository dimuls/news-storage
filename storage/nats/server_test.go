package nats

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/dimuls/news-storage/entity"
	"github.com/dimuls/news-storage/storage/nats/pb"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

type storageMock struct {
	mock.Mock
}

func (s *storageMock) News(ctx context.Context, id int64) (entity.News, error) {
	args := s.Called(id)
	return args.Get(0).(entity.News), args.Error(1)
}

func initServer(t *testing.T) (*storageMock, *Server) {
	sm := &storageMock{}
	s := NewServer(sm, os.Getenv("TEST_NATS_URL"),
		os.Getenv("TEST_SUBSCRIBE_SUBJECT"))
	err := s.Start()
	if err != nil {
		t.Fatal("failed to start replier: " + err.Error())
	}
	return sm, s
}

func cleanServer(t *testing.T, s *Server) {
	s.Stop()
}

func newGetNewsRequest(t *testing.T, id int64) []byte {
	req := &pb.GetNewsRequest{
		Id: id,
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		t.Fatal("failed to marshal *pb.GetNewsRequest: " + err.Error())
	}

	return reqBytes
}

func decodeGetNewsResponse(t *testing.T, resBytes []byte) *pb.GetNewsResponse {
	var res pb.GetNewsResponse

	err := proto.Unmarshal(resBytes, &res)
	if err != nil {
		t.Fatal("failed to unmarshal *pb.GetNewsResponse: " + err.Error())
	}

	return &res
}

func TestReplier_msgHandler_storageError(t *testing.T) {
	sm, s := initServer(t)
	defer cleanServer(t, s)

	sm.On("News", int64(123)).Return(entity.News{},
		errors.New("error"))

	req := newGetNewsRequest(t, 123)

	resMsg, err := s.connection.Request(s.subSubj, req, 1*time.Second)
	if assert.NoError(t, err) {
		return
	}

	sm.AssertExpectations(t)

	res := decodeGetNewsResponse(t, resMsg.Data)

	assert.Nil(t, res.News)
	assert.Equal(t, http.StatusInternalServerError, res.Error.Code)
	assert.Equal(t, "internal server error", res.Error.Message)
}

func TestReplier_msgHandler_success(t *testing.T) {
	sm, s := initServer(t)
	defer cleanServer(t, s)

	testTime, _ := time.Parse("2006-01-02 15:04:05",
		"2006-01-02 15:04:05")

	wantN := entity.News{
		ID:     123,
		Header: "header",
		Date:   testTime,
	}

	sm.On("News", int64(123)).Return(wantN, nil)

	req := newGetNewsRequest(t, 123)

	resMsg, err := s.connection.Request(s.subSubj, req, 1*time.Second)
	if assert.NoError(t, err) {
		return
	}

	sm.AssertExpectations(t)

	res := decodeGetNewsResponse(t, resMsg.Data)

	assert.Nil(t, res.Error)
	assert.Equal(t, wantN.ID, res.News.Id)
	assert.Equal(t, wantN.Header, res.News.Header)
	assert.Equal(t, wantN.Date.UTC().Format("2016-01-02"), res.News.Date)
}
