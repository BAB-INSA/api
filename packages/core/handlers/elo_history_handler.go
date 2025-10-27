package handlers

import (
	"core/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EloHistoryHandler struct {
	eloHistoryService *services.EloHistoryService
}

func NewEloHistoryHandler(eloHistoryService *services.EloHistoryService) *EloHistoryHandler {
	return &EloHistoryHandler{
		eloHistoryService: eloHistoryService,
	}
}

// GetRecentEloChanges retrieves recent ELO changes for all players
// @Summary Get recent ELO changes
// @Description Get recent ELO changes for all players ordered by date (newest first)
// @Tags elo-history
// @Produce json
// @Param limit query int false "Number of ELO changes to retrieve (default: 10, max: 100)"
// @Success 200 {array} models.EloHistory
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /elo-history/recent [get]
func (h *EloHistoryHandler) GetRecentEloChanges(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit parameter",
		})
		return
	}

	// Cap the limit to prevent excessive queries
	if limit > 100 {
		limit = 100
	}

	eloChanges, err := h.eloHistoryService.GetRecentEloChanges(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve recent ELO changes",
		})
		return
	}

	c.JSON(http.StatusOK, eloChanges)
}