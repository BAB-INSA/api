package handlers

import (
	"core/models"
	"core/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TeamHandler struct {
	teamService *services.TeamService
}

func NewTeamHandler(db *gorm.DB) *TeamHandler {
	return &TeamHandler{
		teamService: services.NewTeamService(db),
	}
}

// CreateTeam creates a new team
// @Summary Create a new team
// @Description Create a new team with two players
// @Tags teams
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param team body models.CreateTeamRequest true "Team data"
// @Success 201 {object} models.Team
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams [post]
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req models.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.teamService.CreateTeam(req.Player1ID, req.Player2ID, req.Name)
	if err != nil {
		if err.Error() == "team already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, team)
}

// GetTeam gets a team by ID
// @Summary Get team by ID
// @Description Get team information by team ID
// @Tags teams
// @Produce json
// @Param id path int true "Team ID"
// @Success 200 {object} models.Team
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id} [get]
func (h *TeamHandler) GetTeam(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	team, err := h.teamService.GetTeamByID(uint(id))
	if err != nil {
		if err.Error() == "team not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, team)
}

// UpdateTeam updates a team
// @Summary Update team
// @Description Update team name
// @Tags teams
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Team ID"
// @Param team body models.UpdateTeamRequest true "Team update data"
// @Success 200 {object} models.Team
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id} [put]
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var req models.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.teamService.UpdateTeam(uint(id), req.Name)
	if err != nil {
		if err.Error() == "team not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, team)
}

// DeleteTeam deletes a team
// @Summary Delete team
// @Description Delete a team (admin only)
// @Tags teams
// @Security BearerAuth
// @Produce json
// @Param id path int true "Team ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/{id} [delete]
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	err = h.teamService.DeleteTeam(uint(id))
	if err != nil {
		if err.Error() == "team not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

// GetAllTeams gets all teams with pagination
// @Summary Get all teams
// @Description Get all teams with pagination
// @Tags teams
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedTeamsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams [get]
func (h *TeamHandler) GetAllTeams(c *gin.Context) {
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

	result, err := h.teamService.GetAllTeams(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTeamsByPlayer gets teams for a specific player
// @Summary Get teams by player
// @Description Get all teams that include a specific player
// @Tags teams
// @Produce json
// @Param playerId path int true "Player ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedTeamsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /teams/players/{playerId} [get]
func (h *TeamHandler) GetTeamsByPlayer(c *gin.Context) {
	playerIDParam := c.Param("playerId")
	playerID, err := strconv.ParseUint(playerIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player ID"})
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

	result, err := h.teamService.GetTeamsByPlayer(uint(playerID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
