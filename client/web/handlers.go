package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dimuls/news-storage/entity"
	"github.com/labstack/echo"
)

func (s *Server) getNews(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("news_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"failed to parse news_id: "+err.Error())
	}

	news, err := s.storage.News(c.Request().Context(), id)
	if err != nil {
		if err == entity.ErrNewsNotFound {
			return echo.NewHTTPError(http.StatusNotFound, err)
		}
		return errors.New("failed to get news from storage: " + err.Error())
	}

	return c.JSON(http.StatusOK, news)
}
