package handlers

import (
	"core/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	statsService *services.StatsService
}

func NewStatsHandler(statsService *services.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetStats retrieves general statistics
// @Summary Get general statistics
// @Description Get general statistics including total number of players and matches
// @Tags stats
// @Produce json
// @Success 200 {object} models.Stats
// @Failure 500 {object} map[string]string
// @Router /stats [get]
func (h *StatsHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
