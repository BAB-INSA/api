package handlers

import (
	"core/services"
	"net/http"
	"strconv"

	authMiddleware "auth/middleware"
	"github.com/gin-gonic/gin"
)

type PlayerHandler struct {
	playerService *services.PlayerService
}

func NewPlayerHandler(playerService *services.PlayerService) *PlayerHandler {
	return &PlayerHandler{
		playerService: playerService,
	}
}

// GetPlayer retrieves a player by ID
// @Summary Get player by ID
// @Description Get player information by player ID
// @Tags players
// @Produce json
// @Param id path int true "Player ID"
// @Success 200 {object} models.Player
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /players/{id} [get]
func (h *PlayerHandler) GetPlayer(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	player, err := h.playerService.GetPlayerByID(uint(id))
	if err != nil {
		if err.Error() == "player not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, player)
}

// GetEloHistory retrieves ELO history for a player
// @Summary Get player ELO history
// @Description Get ELO rating history for a specific player
// @Tags players
// @Produce json
// @Param id path int true "Player ID"
// @Success 200 {array} models.EloHistory
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /players/{id}/elo-history [get]
func (h *PlayerHandler) GetEloHistory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	// Check if player exists
	_, err = h.playerService.GetPlayerByID(uint(id))
	if err != nil {
		if err.Error() == "player not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Get ELO history
	eloHistory, err := h.playerService.GetEloHistoryByPlayerID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve ELO history",
		})
		return
	}

	c.JSON(http.StatusOK, eloHistory)
}

// GetTopPlayers retrieves top N players by ELO rating
// @Summary Get top players by ELO rating
// @Description Get top N players ordered by ELO rating (highest first), with option to include current user
// @Tags players
// @Produce json
// @Param limit query int false "Number of players to retrieve (default: 10, max: 100)"
// @Param includeCurrentUser query bool false "Include current user in results even if not in top (default: false)"
// @Success 200 {array} models.Player
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /players/top [get]
func (h *PlayerHandler) GetTopPlayers(c *gin.Context) {
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

	// Vérifier si on doit inclure l'utilisateur connecté
	includeCurrentUserStr := c.DefaultQuery("includeCurrentUser", "false")
	includeCurrentUser, err := strconv.ParseBool(includeCurrentUserStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid includeCurrentUser parameter",
		})
		return
	}

	var currentUserID *uint
	if includeCurrentUser {
		// Récupérer l'ID de l'utilisateur connecté depuis le contexte JWT
		if userID, exists := authMiddleware.GetUserID(c); exists {
			currentUserID = &userID
		}
	}

	players, err := h.playerService.GetTopPlayersByElo(limit, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve top players",
		})
		return
	}

	c.JSON(http.StatusOK, players)
}

// GetPlayerMatches retrieves matches for a specific player with pagination
// @Summary Get matches for a player
// @Description Get matches for a specific player, ordered from newest to oldest, with optional filtering and pagination
// @Tags players
// @Produce json
// @Param id path int true "Player ID"
// @Param wins query string false "Filter for wins only (set to '1')"
// @Param losses query string false "Filter for losses only (set to '1')"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of matches per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedMatchResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /players/{id}/matches [get]
func (h *PlayerHandler) GetPlayerMatches(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	// Check if player exists
	_, err = h.playerService.GetPlayerByID(uint(id))
	if err != nil {
		if err.Error() == "player not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Get filter parameters
	var filter string
	wins := c.Query("wins")
	losses := c.Query("losses")

	if wins == "1" && losses == "1" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot filter for both wins and losses at the same time",
		})
		return
	} else if wins == "1" {
		filter = "wins"
	} else if losses == "1" {
		filter = "losses"
	}

	// Get page parameter
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid page parameter",
		})
		return
	}

	// Get pageSize parameter
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pageSize parameter",
		})
		return
	}

	// Cap the pageSize to prevent excessive queries
	if pageSize > 100 {
		pageSize = 100
	}

	// Get matches
	paginatedResponse, err := h.playerService.GetPlayerMatches(uint(id), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve player matches",
		})
		return
	}

	c.JSON(http.StatusOK, paginatedResponse)
}

// GetAllPlayers retrieves all players with pagination and sorting
// @Summary Get all players
// @Description Get all players with pagination and sorting options
// @Tags players
// @Produce json
// @Param orderBy query string false "Sort field: 'created_at', 'elo_rating', 'username' (default: 'created_at')"
// @Param direction query string false "Sort direction: 'ASC' or 'DESC' (default: 'DESC')"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of players per page (default: 10, max: 100)"
// @Success 200 {object} models.PaginatedPlayersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /players [get]
func (h *PlayerHandler) GetAllPlayers(c *gin.Context) {
	// Get orderBy parameter
	orderBy := c.DefaultQuery("orderBy", "created_at")

	// Get direction parameter
	direction := c.DefaultQuery("direction", "DESC")

	// Get page parameter
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid page parameter",
		})
		return
	}

	// Get pageSize parameter
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pageSize parameter",
		})
		return
	}

	// Cap the pageSize to prevent excessive queries
	if pageSize > 100 {
		pageSize = 100
	}

	// Get players
	paginatedResponse, err := h.playerService.GetAllPlayers(orderBy, direction, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve players",
		})
		return
	}

	c.JSON(http.StatusOK, paginatedResponse)
}
