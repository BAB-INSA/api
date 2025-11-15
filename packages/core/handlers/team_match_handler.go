package handlers

import (
	"core/models"
	"core/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TeamMatchHandler struct {
	teamMatchService *services.TeamMatchService
}

func NewTeamMatchHandler(db *gorm.DB) *TeamMatchHandler {
	return &TeamMatchHandler{
		teamMatchService: services.NewTeamMatchService(db),
	}
}

// CreateTeamMatch creates a new team match
// @Summary Create a new team match
// @Description Create a new match between two teams
// @Tags team-matches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param match body models.CreateTeamMatchRequest true "Team match data"
// @Success 201 {object} models.TeamMatch
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches [post]
func (h *TeamMatchHandler) CreateTeamMatch(c *gin.Context) {
	var req models.CreateTeamMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	match, err := h.teamMatchService.CreateTeamMatch(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, match)
}

// GetTeamMatches gets team matches with filters
// @Summary Get team matches
// @Description Get team matches with optional filters for team, player, status, and date range
// @Tags team-matches
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param per_page query int false "Items per page (default: 10, max: 100)"
// @Param team_id query int false "Filter by team ID"
// @Param player_id query int false "Filter by player ID"
// @Param status query string false "Filter by status" Enums(pending, confirmed, rejected, cancelled)
// @Param date_from query string false "Filter from date (YYYY-MM-DD format)"
// @Param date_to query string false "Filter to date (YYYY-MM-DD format)"
// @Success 200 {object} models.PaginatedTeamMatchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches [get]
func (h *TeamMatchHandler) GetTeamMatches(c *gin.Context) {
	// Parse query parameters
	page := 1
	perPage := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if perPageParam := c.Query("per_page"); perPageParam != "" {
		if pp, err := strconv.Atoi(perPageParam); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	filters := services.TeamMatchFilters{
		Page:    page,
		PerPage: perPage,
	}

	// Parse team_id filter
	if teamIDParam := c.Query("team_id"); teamIDParam != "" {
		if teamID, err := strconv.ParseUint(teamIDParam, 10, 32); err == nil {
			teamIDUint := uint(teamID)
			filters.TeamID = &teamIDUint
		}
	}

	// Parse player_id filter
	if playerIDParam := c.Query("player_id"); playerIDParam != "" {
		if playerID, err := strconv.ParseUint(playerIDParam, 10, 32); err == nil {
			playerIDUint := uint(playerID)
			filters.PlayerID = &playerIDUint
		}
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	// Parse date filters
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if df, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filters.DateFrom = &df
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if dt, err := time.Parse("2006-01-02", dateTo); err == nil {
			filters.DateTo = &dt
		}
	}

	result, err := h.teamMatchService.GetTeamMatches(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRecentTeamMatches gets recent team matches
// @Summary Get recent team matches
// @Description Get the N most recent team matches ordered by creation date (newest first)
// @Tags team-matches
// @Produce json
// @Param limit query int false "Number of matches to retrieve (default: 10, max: 100)"
// @Success 200 {object} map[string][]models.TeamMatch
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches/recent [get]
func (h *TeamMatchHandler) GetRecentTeamMatches(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	matches, err := h.teamMatchService.GetRecentTeamMatches(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": matches})
}

// UpdateTeamMatchStatus updates team match status
// @Summary Update team match status
// @Description Update the status and/or winner of a pending team match
// @Tags team-matches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Team Match ID"
// @Param update body models.UpdateTeamMatchStatusRequest true "Status update data"
// @Success 200 {object} models.TeamMatch
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches/{id} [patch]
func (h *TeamMatchHandler) UpdateTeamMatchStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team match ID"})
		return
	}

	var req models.UpdateTeamMatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	match, err := h.teamMatchService.UpdateTeamMatchStatus(uint(id), req)
	if err != nil {
		if err.Error() == "team match not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "team match is not pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, match)
}

// RejectTeamMatch rejects a team match
// @Summary Reject team match
// @Description Reject a pending team match
// @Tags team-matches
// @Security BearerAuth
// @Produce json
// @Param id path int true "Team Match ID"
// @Success 200 {object} models.TeamMatch
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches/{id}/reject [patch]
func (h *TeamMatchHandler) RejectTeamMatch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team match ID"})
		return
	}

	match, err := h.teamMatchService.RejectTeamMatch(uint(id))
	if err != nil {
		if err.Error() == "team match not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "team match is not pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, match)
}

// CancelTeamMatch cancels a team match
// @Summary Cancel team match
// @Description Cancel a team match (admin only)
// @Tags team-matches
// @Security BearerAuth
// @Produce json
// @Param id path int true "Team Match ID"
// @Success 200 {object} models.TeamMatch
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /team-matches/{id}/cancel [patch]
func (h *TeamMatchHandler) CancelTeamMatch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team match ID"})
		return
	}

	match, err := h.teamMatchService.CancelTeamMatch(uint(id))
	if err != nil {
		if err.Error() == "team match not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, match)
}
