package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)


func (s *Service) InitHandlers() {
	s.GET("/api/stats", s.severAPIStats)
	s.GET("/api/test", s.serverTest)
}

func (s *Service) severAPIStats(c *gin.Context) {
	stats, err := s.Monitor.GodEyeStatistics(nil)
	if err != nil {
		c.JSON(400, err)
		return
	}

	c.JSON(200, stats)
	return
}

func (s *Service) serverTest(c *gin.Context) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		c.JSON(400, err)
	}

	c.JSON(200, metrics)
	return
}