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
	"github.com/sirupsen/logrus"
)

type Storage interface {
	News(ctx context.Context, id int64) (entity.News, error)
}

type Server struct {
	storage Storage
	natsURL string
	subSubj string

	connection   *nats.Conn
	subscription *nats.Subscription

	log *logrus.Entry
}

func NewServer(s Storage, natsURL, subSubj string) *Server {
	return &Server{
		storage: s,
		natsURL: natsURL,
		subSubj: subSubj,
		log:     logrus.WithField("subsystem", "nats_server"),
	}
}

func (s *Server) Start() error {
	conn, err := nats.Connect(s.natsURL, nats.DrainTimeout(5*time.Second))
	if err != nil {
		return errors.New("failed to connect to nats: " + err.Error())
	}

	sub, err := conn.Subscribe(s.subSubj, s.msgHandler)
	if err != nil {
		conn.Close()
		return errors.New("failed to subscribe: " + err.Error())
	}

	s.connection = conn
	s.subscription = sub

	return nil
}

func (s *Server) Stop() {
	err := s.subscription.Unsubscribe()
	if err != nil {
		s.log.WithError(err).Error("failed to unsubscribe")
	}

	s.connection.Close()
}

func (s *Server) msgHandler(msg *nats.Msg) {
	var req pb.GetNewsRequest

	err := proto.Unmarshal(msg.Data, &req)
	if err != nil {
		s.log.WithError(err).Error("failed to unmarshal request")
		s.respondWithError(msg, http.StatusBadRequest,
			"failed to unmarshal request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	news, err := s.storage.News(ctx, req.Id)
	if err != nil {
		if err == entity.ErrNewsNotFound {
			s.respondWithError(msg, http.StatusNotFound, err.Error())
			return
		}
		s.log.WithError(err).Error("failed to get news from storage")
		s.respondWithError(msg, http.StatusInternalServerError,
			"internal server error")
		return
	}

	res := &pb.GetNewsResponse{
		News: &pb.News{
			Id:     news.ID,
			Header: news.Header,
			Date:   news.Date.UTC().Format("2006-01-02"),
		},
	}

	resBytes, err := proto.Marshal(res)
	if err != nil {
		s.log.WithError(err).Error("failed to marshal response")
		s.respondWithError(msg, http.StatusInternalServerError,
			"internal server error")
		return
	}

	err = msg.Respond(resBytes)
	if err != nil {
		s.log.WithError(err).Error("failed to respond: " + err.Error())
	}
}

func (s *Server) respondWithError(msg *nats.Msg, errCode int, errMsg string) {
	res := &pb.GetNewsResponse{
		Error: &pb.Error{
			Code:    int64(errCode),
			Message: "failed to unmarshal request: " + errMsg,
		},
	}

	resBytes, err := proto.Marshal(res)
	if err != nil {
		s.log.WithError(err).Error(
			"failed to marshal error response")
		return
	}

	err = msg.Respond(resBytes)
	if err != nil {
		s.log.WithError(err).Error("failed to respond to message")
	}
}
