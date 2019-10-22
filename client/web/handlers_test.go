package web

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dimuls/news-storage/entity"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

type mockStorage struct {
	mock.Mock
}

func (s *mockStorage) News(ctx context.Context, id int64) (entity.News, error) {
	args := s.Called(id)
	return args.Get(0).(entity.News), args.Error(1)
}

func initServer() (*mockStorage, *Server) {
	ms := &mockStorage{}
	s := NewServer(ms, "")
	s.Start()
	return ms, s
}

func TestServer_getNews_badRequest(t *testing.T) {
	ms, s := initServer()
	defer s.Stop()

	req := httptest.NewRequest(http.MethodGet, "/news/asd", nil)
	res := httptest.NewRecorder()

	s.echo.ServeHTTP(res, req)

	ms.AssertExpectations(t)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestServer_getNews_internalError(t *testing.T) {
	ms, s := initServer()
	defer s.Stop()

	ms.On("News", int64(123)).Return(entity.News{},
		errors.New("error"))

	req := httptest.NewRequest(http.MethodGet, "/news/123", nil)
	res := httptest.NewRecorder()

	s.echo.ServeHTTP(res, req)

	ms.AssertExpectations(t)

	assert.Equal(t, http.StatusInternalServerError, res.Code)
}

func TestServer_getNews_notFound(t *testing.T) {
	ms, s := initServer()
	defer s.Stop()

	ms.On("News", int64(123)).Return(entity.News{},
		entity.ErrNewsNotFound)

	req := httptest.NewRequest(http.MethodGet, "/news/123", nil)
	res := httptest.NewRecorder()

	s.echo.ServeHTTP(res, req)

	ms.AssertExpectations(t)

	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestServer_getNews_success(t *testing.T) {
	ms, s := initServer()
	defer s.Stop()

	testTime, _ := time.Parse("2006-01-02", "2006-01-02")

	wantN := entity.News{
		ID:     123,
		Header: "header",
		Date:   testTime,
	}

	ms.On("News", int64(123)).Return(wantN, nil)

	req := httptest.NewRequest(http.MethodGet, "/news/123", nil)
	res := httptest.NewRecorder()

	s.echo.ServeHTTP(res, req)

	ms.AssertExpectations(t)

	if !assert.Equal(t, http.StatusOK, res.Code) {
		return
	}

	var gotN entity.News

	err := json.NewDecoder(res.Body).Decode(&gotN)
	if assert.NoError(t, err) {
		if !assert.True(t, cmp.Equal(wantN, gotN)) {
			t.Log(cmp.Diff(wantN, gotN))
		}
	}
}
