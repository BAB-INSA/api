package handlers

import (
	"core/models"
	"core/services"
	"errors"
	"net/http"
	"strconv"

	authMiddleware "auth/middleware"
	authModels "auth/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MatchHandler struct {
	matchService *services.MatchService
	db           *gorm.DB
}

func NewMatchHandler(matchService *services.MatchService, db *gorm.DB) *MatchHandler {
	return &MatchHandler{
		matchService: matchService,
		db:           db,
	}
}

// GetRecentMatches retrieves the N most recent matches
// @Summary Get recent matches
// @Description Get the N most recent matches ordered by creation date (newest first)
// @Tags matches
// @Produce json
// @Param limit query int false "Number of matches to retrieve (default: 10, max: 100)"
// @Success 200 {array} models.Match
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /matches/recent [get]
func (h *MatchHandler) GetRecentMatches(c *gin.Context) {
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

	matches, err := h.matchService.GetRecentMatches(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve recent matches",
		})
		return
	}

	c.JSON(http.StatusOK, matches)
}

// CreateMatch creates a new match
// @Summary Create a new match
// @Description Create a new match between two players with automatic ELO calculation and stats update
// @Tags matches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param match body models.CreateMatchRequest true "Match data"
// @Success 201 {object} models.Match
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /matches [post]
func (h *MatchHandler) CreateMatch(c *gin.Context) {
	// Get authenticated user ID
	userID, exists := authMiddleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	var req models.CreateMatchRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Authorization check: user must be admin OR one of the players
	if err := h.checkMatchAuthorization(c, userID, req.Player1ID, req.Player2ID); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only create matches for yourself or you must be an admin",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authorization check failed",
			})
		}
		return
	}

	match, err := h.matchService.CreateMatch(req)
	if err != nil {
		if err.Error() == "player1 not found" || err.Error() == "player2 not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		
		if err.Error() == "player1 and player2 must be different" || 
		   err.Error() == "winner must be either player1 or player2" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create match",
		})
		return
	}

	c.JSON(http.StatusCreated, match)
}

// UpdateMatchStatus confirms or rejects a pending match
// @Summary Update match status (confirm/reject)
// @Description Update the status of a pending match. Only player2 or admin can confirm/reject.
// @Tags matches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Match ID"
// @Param status body models.UpdateMatchStatusRequest true "New status"
// @Success 200 {object} models.Match
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /matches/{id} [patch]
func (h *MatchHandler) UpdateMatchStatus(c *gin.Context) {
	// Get authenticated user ID
	userID, exists := authMiddleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	// Get match ID from URL
	matchIDStr := c.Param("id")
	matchID, err := strconv.ParseUint(matchIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid match ID",
		})
		return
	}

	var req models.UpdateMatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Authorization check: user must be player2 or admin
	if err := h.checkMatchStatusUpdateAuthorization(c, userID, uint(matchID)); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Only player2 or admin can confirm/reject matches",
			})
		} else if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authorization check failed",
			})
		}
		return
	}

	// Update match status
	match, err := h.matchService.UpdateMatchStatus(uint(matchID), req)
	if err != nil {
		if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
			return
		}
		if err.Error() == "match is not pending" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Match is not pending",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update match status",
		})
		return
	}

	c.JSON(http.StatusOK, match)
}

// checkMatchAuthorization vérifie si l'utilisateur a le droit de créer ce match
func (h *MatchHandler) checkMatchAuthorization(c *gin.Context, userID, player1ID, player2ID uint) error {
	// Check if user is one of the players (user_id = player_id)
	if userID == player1ID || userID == player2ID {
		return nil
	}

	// Check if user is admin
	var user authModels.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}

	if user.HasRole(authModels.RoleAdmin) {
		return nil
	}

	return errors.New("unauthorized")
}

// checkMatchStatusUpdateAuthorization vérifie si l'utilisateur peut mettre à jour le statut du match
func (h *MatchHandler) checkMatchStatusUpdateAuthorization(c *gin.Context, userID, matchID uint) error {
	// Get the match to check player2
	var match models.Match
	if err := h.db.First(&match, matchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("match not found")
		}
		return err
	}

	// Check if user is player2 (user_id = player_id)
	if userID == match.Player2ID {
		return nil
	}

	// Check if user is admin
	var user authModels.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}

	if user.HasRole(authModels.RoleAdmin) {
		return nil
	}

	return errors.New("unauthorized")
}

// ConfirmMatch confirms a match (only accessible to player2 or admin)
// @Summary Confirm a match
// @Description Confirm a pending match. Only player2 or admin can confirm.
// @Tags matches
// @Security BearerAuth
// @Produce json
// @Param id path int true "Match ID"
// @Success 200 {object} models.Match
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /matches/{id}/confirm [put]
func (h *MatchHandler) ConfirmMatch(c *gin.Context) {
	// Get authenticated user ID
	userID, exists := authMiddleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	// Get match ID from URL
	matchIDStr := c.Param("id")
	matchID, err := strconv.ParseUint(matchIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid match ID",
		})
		return
	}

	// Authorization check: user must be player2 or admin
	if err := h.checkMatchStatusUpdateAuthorization(c, userID, uint(matchID)); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Only player2 or admin can confirm matches",
			})
		} else if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authorization check failed",
			})
		}
		return
	}

	// Confirm match
	match, err := h.matchService.ConfirmMatch(uint(matchID))
	if err != nil {
		if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
			return
		}
		if err.Error() == "match is not pending" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Match is not pending",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to confirm match",
		})
		return
	}

	c.JSON(http.StatusOK, match)
}