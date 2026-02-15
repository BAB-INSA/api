package core

import (
	"core/cron"
	"core/handlers"
	"core/services"
	"log"

	authMiddleware "auth/middleware"
	authModels "auth/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	PlayerHandler         *handlers.PlayerHandler
	PlayerService         *services.PlayerService
	MatchHandler          *handlers.MatchHandler
	MatchService          *services.MatchService
	TeamHandler           *handlers.TeamHandler
	TeamService           *services.TeamService
	TeamMatchHandler      *handlers.TeamMatchHandler
	TeamMatchService      *services.TeamMatchService
	TournamentHandler     *handlers.TournamentHandler
	TournamentService     *services.TournamentService
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
	teamService := services.NewTeamService(db)
	playerHandler := handlers.NewPlayerHandler(playerService, teamService)

	matchService := services.NewMatchService(db)
	matchHandler := handlers.NewMatchHandler(matchService, db)

	teamHandler := handlers.NewTeamHandler(db)

	teamMatchService := services.NewTeamMatchService(db)
	teamMatchHandler := handlers.NewTeamMatchHandler(db)

	tournamentService := services.NewTournamentService(db)
	tournamentHandler := handlers.NewTournamentHandler(db)

	eloHistoryService := services.NewEloHistoryService(db)
	eloHistoryHandler := handlers.NewEloHistoryHandler(eloHistoryService)

	statsService := services.NewStatsService(db)
	statsHandler := handlers.NewStatsHandler(statsService)

	// Initialize auto-validation service and scheduler
	autoValidationService := services.NewAutoValidationService(db, matchService, teamMatchService)
	scheduler := cron.NewScheduler(autoValidationService)

	return &Module{
		PlayerHandler:         playerHandler,
		PlayerService:         playerService,
		MatchHandler:          matchHandler,
		MatchService:          matchService,
		TeamHandler:           teamHandler,
		TeamService:           teamService,
		TeamMatchHandler:      teamMatchHandler,
		TeamMatchService:      teamMatchService,
		TournamentHandler:     tournamentHandler,
		TournamentService:     tournamentService,
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
		players.GET("/top", authMiddleware.OptionalJWTMiddleware(), m.PlayerHandler.GetTopPlayers)
		players.GET("/top-teams", authMiddleware.OptionalJWTMiddleware(), m.PlayerHandler.GetTopPlayersByTeamElo)
		players.GET("/:id", m.PlayerHandler.GetPlayer)
		players.GET("/:id/elo-history", m.PlayerHandler.GetEloHistory)
		players.GET("/:id/matches", m.PlayerHandler.GetPlayerMatches)
		players.GET("/:id/teams", m.PlayerHandler.GetPlayerTeams)
	}

	matches := r.Group("/matches")
	{
		matches.GET("", m.MatchHandler.GetMatches)
		matches.GET("/recent", m.MatchHandler.GetRecentMatches)
		matches.POST("", authMiddleware.JWTMiddleware(), m.MatchHandler.CreateMatch)
		matches.PATCH("/:id", authMiddleware.JWTMiddleware(), m.MatchHandler.UpdateMatchStatus)
		matches.PATCH("/:id/reject", authMiddleware.JWTMiddleware(), m.MatchHandler.RejectMatch)
		matches.PATCH("/:id/cancel", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.MatchHandler.CancelMatch)
		matches.DELETE("/:id", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.MatchHandler.DeleteMatch)
	}

	teams := r.Group("/teams")
	{
		teams.GET("", m.TeamHandler.GetAllTeams)
		teams.GET("/:id", m.TeamHandler.GetTeam)
		teams.POST("", authMiddleware.JWTMiddleware(), m.TeamHandler.CreateTeam)
		teams.PUT("/:id", authMiddleware.JWTMiddleware(), m.TeamHandler.UpdateTeam)
		teams.DELETE("/:id", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.TeamHandler.DeleteTeam)
		teams.GET("/players/:playerId", m.TeamHandler.GetTeamsByPlayer)
	}

	teamMatches := r.Group("/team-matches")
	{
		teamMatches.GET("", m.TeamMatchHandler.GetTeamMatches)
		teamMatches.GET("/recent", m.TeamMatchHandler.GetRecentTeamMatches)
		teamMatches.POST("", authMiddleware.JWTMiddleware(), m.TeamMatchHandler.CreateTeamMatch)
		teamMatches.PATCH("/:id", authMiddleware.JWTMiddleware(), m.TeamMatchHandler.UpdateTeamMatchStatus)
		teamMatches.PATCH("/:id/reject", authMiddleware.JWTMiddleware(), m.TeamMatchHandler.RejectTeamMatch)
		teamMatches.PATCH("/:id/cancel", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.TeamMatchHandler.CancelTeamMatch)
	}

	tournaments := r.Group("/tournaments")
	{
		tournaments.GET("", m.TournamentHandler.GetAllTournaments)
		tournaments.GET("/:id", m.TournamentHandler.GetTournament)
		tournaments.GET("/:id/teams", m.TournamentHandler.GetTournamentTeams)
		tournaments.GET("/:id/matches", m.TournamentHandler.GetTournamentMatches)
		tournaments.POST("", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.TournamentHandler.CreateTournament)
		tournaments.PUT("/:id", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.TournamentHandler.UpdateTournament)
		tournaments.POST("/:id/join", authMiddleware.JWTMiddleware(), m.TournamentHandler.JoinTournament)
		tournaments.DELETE("/:id/teams/:teamId", authMiddleware.JWTMiddleware(), m.TournamentHandler.LeaveTournament)
		tournaments.DELETE("/:id", authMiddleware.JWTMiddleware(), authMiddleware.RequireRole(m.db, authModels.RoleAdmin), m.TournamentHandler.DeleteTournament)
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
