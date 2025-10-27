package core

import (
	"core/cron"
	"core/handlers"
	"core/services"
	"log"

	authMiddleware "auth/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	PlayerHandler         *handlers.PlayerHandler
	PlayerService         *services.PlayerService
	MatchHandler          *handlers.MatchHandler
	MatchService          *services.MatchService
	EloHistoryHandler     *handlers.EloHistoryHandler
	EloHistoryService     *services.EloHistoryService
	StatsHandler          *handlers.StatsHandler
	StatsService          *services.StatsService
	AutoValidationService *services.AutoValidationService
	Scheduler             *cron.Scheduler
	db                    *gorm.DB
}

func NewModule(db *gorm.DB) *Module {
	playerService := services.NewPlayerService(db)
	playerHandler := handlers.NewPlayerHandler(playerService)
	
	matchService := services.NewMatchService(db)
	matchHandler := handlers.NewMatchHandler(matchService, db)
	
	eloHistoryService := services.NewEloHistoryService(db)
	eloHistoryHandler := handlers.NewEloHistoryHandler(eloHistoryService)
	
	statsService := services.NewStatsService(db)
	statsHandler := handlers.NewStatsHandler(statsService)

	// Initialize auto-validation service and scheduler
	autoValidationService := services.NewAutoValidationService(db, matchService)
	scheduler := cron.NewScheduler(autoValidationService)

	return &Module{
		PlayerHandler:         playerHandler,
		PlayerService:         playerService,
		MatchHandler:          matchHandler,
		MatchService:          matchService,
		EloHistoryHandler:     eloHistoryHandler,
		EloHistoryService:     eloHistoryService,
		StatsHandler:          statsHandler,
		StatsService:          statsService,
		AutoValidationService: autoValidationService,
		Scheduler:             scheduler,
		db:                    db,
	}
}

func (m *Module) SetupRoutes(r *gin.Engine) {
	players := r.Group("/players")
	{
		players.GET("", m.PlayerHandler.GetAllPlayers)
		players.GET("/top", m.PlayerHandler.GetTopPlayers)
		players.GET("/top-current-streak", m.PlayerHandler.GetTopPlayersByCurrentStreak)
		players.GET("/top-best-streak", m.PlayerHandler.GetTopPlayersByBestStreak)
		players.GET("/:id", m.PlayerHandler.GetPlayer)
		players.GET("/:id/elo-history", m.PlayerHandler.GetEloHistory)
		players.GET("/:id/matches", m.PlayerHandler.GetPlayerMatches)
	}

	matches := r.Group("/matches")
	{
		matches.GET("/recent", m.MatchHandler.GetRecentMatches)
		matches.POST("", authMiddleware.JWTMiddleware(), m.MatchHandler.CreateMatch)
		matches.PATCH("/:id", authMiddleware.JWTMiddleware(), m.MatchHandler.UpdateMatchStatus)
		matches.PUT("/:id/confirm", authMiddleware.JWTMiddleware(), m.MatchHandler.ConfirmMatch)
	}

	eloHistory := r.Group("/elo-history")
	{
		eloHistory.GET("/recent", m.EloHistoryHandler.GetRecentEloChanges)
	}

	r.GET("/stats", m.StatsHandler.GetStats)
}

// StartScheduler starts the cron scheduler for auto-validation
func (m *Module) StartScheduler() error {
	log.Println("Starting core module scheduler...")
	return m.Scheduler.Start()
}

// StopScheduler stops the cron scheduler
func (m *Module) StopScheduler() {
	log.Println("Stopping core module scheduler...")
	m.Scheduler.Stop()
}

// RunAutoValidationNow manually triggers auto-validation (useful for testing)
func (m *Module) RunAutoValidationNow() {
	log.Println("Manually triggering auto-validation...")
	m.Scheduler.RunNow()
}