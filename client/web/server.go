package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dimuls/news-storage/entity"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	News(ctx context.Context, id int64) (entity.News, error)
}

type Server struct {
	storage  Storage
	bindAddr string
	echo     *echo.Echo
	wg       sync.WaitGroup
	log      *logrus.Entry
}

func NewServer(s Storage, bindAddr string) *Server {
	return &Server{
		storage:  s,
		bindAddr: bindAddr,
		log:      logrus.WithField("subsystem", "web_server"),
	}
}

func (s *Server) Start() {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var (
			code = http.StatusInternalServerError
			msg  interface{}
		)

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message
		} else if e.Debug {
			msg = err.Error()
		} else {
			msg = http.StatusText(code)
		}
		if _, ok := msg.(string); !ok {
			msg = fmt.Sprintf("%v", msg)
		}

		// Send response
		if !c.Response().Committed {
			if c.Request().Method == http.MethodHead { // Issue #608
				err = c.NoContent(code)
			} else {
				err = c.String(code, msg.(string))
			}
			if err != nil {
				s.log.WithError(err).Error("failed to error response")
			}
		}
	}

	e.Use(middleware.Recover(), logrusLogger)

	e.GET("/news/:news_id", s.getNews)

	s.echo = e

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			err := s.echo.Start(s.bindAddr)
			if err != nil {
				if err == http.ErrServerClosed {
					s.log.Info("server is closed")
					return
				}
				s.log.WithError(err).Error("failed to start")
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.echo.Shutdown(ctx)
	if err != nil {
		s.log.WithError(err).Error("failed to graceful shutdown")
	}

	s.wg.Wait()
}

func logrusLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		err := next(c)

		stop := time.Now()

		if err != nil {
			c.Error(err)
		}

		req := c.Request()
		res := c.Response()

		p := req.URL.Path
		if p == "" {
			p = "/"
		}

		bytesIn := req.Header.Get(echo.HeaderContentLength)
		if bytesIn == "" {
			bytesIn = "0"
		}

		entry := logrus.WithFields(map[string]interface{}{
			"subsystem":    "web_server",
			"remote_ip":    c.RealIP(),
			"host":         req.Host,
			"query_params": c.QueryParams(),
			"uri":          req.RequestURI,
			"method":       req.Method,
			"path":         p,
			"referer":      req.Referer(),
			"user_agent":   req.UserAgent(),
			"status":       res.Status,
			"latency":      stop.Sub(start).String(),
			"bytes_in":     bytesIn,
			"bytes_out":    strconv.FormatInt(res.Size, 10),
		})

		const msg = "request handled"

		if res.Status >= 500 {
			if err != nil {
				entry = entry.WithError(err)
			}
			entry.Error(msg)
		} else if res.Status >= 400 {
			entry.Warn(msg)
		} else {
			entry.Info(msg)
		}

		return nil
	}
}
