package handlers

import (
	"core/models"
	"core/services"
	"net/http"
	"strconv"
	"strings"

	authMiddleware "auth/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TournamentHandler struct {
	tournamentService *services.TournamentService
	db                *gorm.DB
}

func NewTournamentHandler(db *gorm.DB) *TournamentHandler {
	return &TournamentHandler{
		tournamentService: services.NewTournamentService(db),
		db:                db,
	}
}

// CreateTournament creates a new tournament
// @Summary Create a new tournament
// @Description Create a new tournament (admin only)
// @Tags tournaments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param tournament body models.CreateTournamentRequest true "Tournament data"
// @Success 201 {object} models.Tournament
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /tournaments [post]
func (h *TournamentHandler) CreateTournament(c *gin.Context) {
	var req models.CreateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, err := h.tournamentService.CreateTournament(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tournament)
}

// GetTournament gets a tournament by ID
// @Summary Get tournament by ID
// @Description Get tournament information with teams
// @Tags tournaments
// @Produce json
// @Param id path int true "Tournament ID"
// @Success 200 {object} models.TournamentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id} [get]
func (h *TournamentHandler) GetTournament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	tournament, err := h.tournamentService.GetTournamentByID(uint(id))
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, tournament)
}

// GetAllTournaments gets all tournaments with pagination
// @Summary Get all tournaments
// @Description Get all tournaments with optional status filter
// @Tags tournaments
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 10, max: 100)"
// @Param status query string false "Filter by status" Enums(opened, ongoing, finished)
// @Param type query string false "Filter by type" Enums(solo, team)
// @Success 200 {object} models.PaginatedTournamentsResponse
// @Failure 500 {object} map[string]string
// @Router /tournaments [get]
func (h *TournamentHandler) GetAllTournaments(c *gin.Context) {
	page := 1
	pageSize := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if sizeParam := c.Query("pageSize"); sizeParam != "" {
		if ps, err := strconv.Atoi(sizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	var tournamentType *string
	if t := c.Query("type"); t != "" {
		tournamentType = &t
	}

	result, err := h.tournamentService.GetAllTournaments(page, pageSize, status, tournamentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateTournament updates a tournament
// @Summary Update tournament
// @Description Update tournament name or description (admin only)
// @Tags tournaments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Tournament ID"
// @Param tournament body models.UpdateTournamentRequest true "Tournament update data"
// @Success 200 {object} models.Tournament
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id} [put]
func (h *TournamentHandler) UpdateTournament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	var req models.UpdateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tournament, err := h.tournamentService.UpdateTournament(uint(id), req)
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, tournament)
}

// JoinTournament registers a team for a tournament
// @Summary Join tournament
// @Description Register a team for a tournament (must be a team member)
// @Tags tournaments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Tournament ID"
// @Param request body models.JoinTournamentRequest true "Join request"
// @Success 201 {object} models.TournamentTeam
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/join [post]
func (h *TournamentHandler) JoinTournament(c *gin.Context) {
	idParam := c.Param("id")
	tournamentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	var req models.JoinTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := authMiddleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tournamentTeam, err := h.tournamentService.JoinTournament(uint(tournamentID), req.TeamID, userID)
	if err != nil {
		switch {
		case err.Error() == "tournament not found" || err.Error() == "team not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case strings.Contains(err.Error(), "already registered"):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, tournamentTeam)
}

// LeaveTournament removes a team from a tournament
// @Summary Leave tournament
// @Description Remove a team from a tournament (must be a team member)
// @Tags tournaments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Tournament ID"
// @Param teamId path int true "Team ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id}/teams/{teamId} [delete]
func (h *TournamentHandler) LeaveTournament(c *gin.Context) {
	idParam := c.Param("id")
	tournamentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	teamIDParam := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	userID, exists := authMiddleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.tournamentService.LeaveTournament(uint(tournamentID), uint(teamID), userID)
	if err != nil {
		switch err.Error() {
		case "tournament not found", "team not found", "team is not registered in this tournament":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team removed from tournament successfully"})
}

// GetTournamentTeams gets teams registered in a tournament
// @Summary Get tournament teams
// @Description Get paginated list of teams registered in a tournament
// @Tags tournaments
// @Produce json
// @Param id path int true "Tournament ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedTournamentTeamsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tournaments/{id}/teams [get]
func (h *TournamentHandler) GetTournamentTeams(c *gin.Context) {
	idParam := c.Param("id")
	tournamentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	page := 1
	pageSize := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if sizeParam := c.Query("pageSize"); sizeParam != "" {
		if ps, err := strconv.Atoi(sizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	result, err := h.tournamentService.GetTournamentTeams(uint(tournamentID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTournamentMatches gets matches for a tournament
// @Summary Get tournament matches
// @Description Get paginated list of matches in a tournament
// @Tags tournaments
// @Produce json
// @Param id path int true "Tournament ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedTeamMatchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tournaments/{id}/matches [get]
func (h *TournamentHandler) GetTournamentMatches(c *gin.Context) {
	idParam := c.Param("id")
	tournamentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	page := 1
	pageSize := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if sizeParam := c.Query("pageSize"); sizeParam != "" {
		if ps, err := strconv.Atoi(sizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	result, err := h.tournamentService.GetTournamentMatches(uint(tournamentID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteTournament deletes a tournament
// @Summary Delete tournament
// @Description Delete a tournament (admin only)
// @Tags tournaments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Tournament ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tournaments/{id} [delete]
func (h *TournamentHandler) DeleteTournament(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tournament ID"})
		return
	}

	err = h.tournamentService.DeleteTournament(uint(id))
	if err != nil {
		if err.Error() == "tournament not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tournament deleted successfully"})
}
