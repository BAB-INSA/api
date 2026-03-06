package handlers

import (
	"core/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TeamEloHistoryHandler struct {
	eloHistoryService *services.EloHistoryService
}

func NewTeamEloHistoryHandler(eloHistoryService *services.EloHistoryService) *TeamEloHistoryHandler {
	return &TeamEloHistoryHandler{
		eloHistoryService: eloHistoryService,
	}
}

// GetRecentTeamEloChanges retrieves recent team ELO changes for all players
// @Summary Get recent team ELO changes
// @Description Get recent team ELO changes for all players ordered by date (newest first)
// @Tags team-elo-history
// @Produce json
// @Param limit query int false "Number of ELO changes to retrieve (default: 10, max: 100)"
// @Success 200 {array} models.TeamEloHistory
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-elo-history/recent [get]
func (h *TeamEloHistoryHandler) GetRecentTeamEloChanges(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit parameter",
		})
		return
	}

	if limit > 100 {
		limit = 100
	}

	eloChanges, err := h.eloHistoryService.GetRecentTeamEloChanges(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve recent team ELO changes",
		})
		return
	}

	c.JSON(http.StatusOK, eloChanges)
}
