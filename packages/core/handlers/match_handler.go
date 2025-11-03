package handlers

import (
	"core/models"
	"core/services"
	"errors"
	"net/http"
	"strconv"
	"time"

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

// GetMatches retrieves matches with pagination and filters
// @Summary Get matches with pagination and filters
// @Description Get matches with optional filters for player, status, and date range
// @Tags matches
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param per_page query int false "Items per page (default: 10, max: 100)" default(10)
// @Param player_id query int false "Filter by player ID (matches where player is player1 or player2)"
// @Param status query string false "Filter by match status" Enums(pending,confirmed,rejected)
// @Param date_from query string false "Filter from date (YYYY-MM-DD format)"
// @Param date_to query string false "Filter to date (YYYY-MM-DD format)"
// @Success 200 {object} models.PaginatedMatchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /matches [get]
func (h *MatchHandler) GetMatches(c *gin.Context) {
	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	perPageStr := c.DefaultQuery("per_page", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}

	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid per_page parameter"})
		return
	}

	// Limit per_page to maximum 100
	if perPage > 100 {
		perPage = 100
	}

	// Build filters
	filters := services.MatchFilters{
		Page:    page,
		PerPage: perPage,
	}

	// Parse player_id filter
	if playerIDStr := c.Query("player_id"); playerIDStr != "" {
		playerID, err := strconv.ParseUint(playerIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player_id parameter"})
			return
		}
		playerIDUint := uint(playerID)
		filters.PlayerID = &playerIDUint
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		// Validate status
		if status != "pending" && status != "confirmed" && status != "rejected" && status != "cancelled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be one of: pending, confirmed, rejected, cancelled"})
			return
		}
		filters.Status = &status
	}

	// Parse date_from filter
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_from format. Use YYYY-MM-DD"})
			return
		}
		filters.DateFrom = &dateFrom
	}

	// Parse date_to filter
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse("2006-01-02", dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_to format. Use YYYY-MM-DD"})
			return
		}
		filters.DateTo = &dateTo
	}

	// Get matches from service
	result, err := h.matchService.GetMatches(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve matches"})
		return
	}

	c.JSON(http.StatusOK, result)
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

// UpdateMatchStatus updates match status and/or winner
// @Summary Update match status and/or winner (PATCH)
// @Description Update the status and/or winner of a pending match. All fields are optional. Only player2 or admin can update.
// @Tags matches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Match ID"
// @Param update body models.UpdateMatchStatusRequest true "Optional status and/or winner update"
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

	// Validate that at least one field is provided
	if req.Status == nil && req.WinnerID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one field (status or winner_id) must be provided",
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
		if err.Error() == "winner must be either player1 or player2" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Winner must be either player1 or player2",
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

// RejectMatch rejects a match (only accessible to player2 or admin)
// @Summary Reject a match
// @Description Reject a pending match. Only player2 or admin can reject.
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
// @Router /matches/{id}/reject [patch]
func (h *MatchHandler) RejectMatch(c *gin.Context) {
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
				"error": "Only player2 or admin can reject matches",
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

	// Reject match
	match, err := h.matchService.RejectMatch(uint(matchID))
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
			"error": "Failed to reject match",
		})
		return
	}

	c.JSON(http.StatusOK, match)
}

// CancelMatch cancels a match (only accessible to admin)
// @Summary Cancel a match
// @Description Cancel a match by setting its status to cancelled. Only admin can cancel matches.
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
// @Router /matches/{id}/cancel [patch]
func (h *MatchHandler) CancelMatch(c *gin.Context) {
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

	// Authorization check: user must be admin
	if err := h.checkAdminAuthorization(userID); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Only admin can cancel matches",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authorization check failed",
			})
		}
		return
	}

	// Cancel match
	match, err := h.matchService.CancelMatch(uint(matchID))
	if err != nil {
		if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel match",
		})
		return
	}

	c.JSON(http.StatusOK, match)
}

// checkAdminAuthorization vérifie si l'utilisateur est admin
func (h *MatchHandler) checkAdminAuthorization(userID uint) error {
	var user authModels.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return err
	}

	if user.HasRole(authModels.RoleAdmin) {
		return nil
	}

	return errors.New("unauthorized")
}

// DeleteMatch deletes a match by setting status to deleted (only accessible to admin)
// @Summary Delete a match
// @Description Delete a match by setting its status to deleted. Only admin can delete matches.
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
// @Router /matches/{id} [delete]
func (h *MatchHandler) DeleteMatch(c *gin.Context) {
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

	// Authorization check: user must be admin
	if err := h.checkAdminAuthorization(userID); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Only admin can delete matches",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authorization check failed",
			})
		}
		return
	}

	// Delete match
	match, err := h.matchService.DeleteMatch(uint(matchID))
	if err != nil {
		if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Match not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete match",
		})
		return
	}

	c.JSON(http.StatusOK, match)
}
