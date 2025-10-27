package cron

import (
	"core/services"
	"log"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron                  *cron.Cron
	autoValidationService *services.AutoValidationService
}

func NewScheduler(autoValidationService *services.AutoValidationService) *Scheduler {
	// Create cron with seconds precision and logging
	c := cron.New(cron.WithSeconds(), cron.WithLogger(cron.VerbosePrintfLogger(log.Default())))
	
	return &Scheduler{
		cron:                  c,
		autoValidationService: autoValidationService,
	}
}

// Start initializes and starts all scheduled jobs
func (s *Scheduler) Start() error {
	log.Println("Starting cron scheduler...")

	// Schedule auto-validation job to run every hour
	// Cron expression: "0 0 * * * *" = at minute 0 of every hour
	_, err := s.cron.AddFunc("0 0 * * * *", s.runAutoValidation)
	if err != nil {
		log.Printf("Error scheduling auto-validation job: %v", err)
		return err
	}

	// You can add more scheduled jobs here in the future
	// Example: cleanup job, statistics calculation, etc.

	s.cron.Start()
	log.Println("Cron scheduler started successfully")
	
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping cron scheduler...")
	s.cron.Stop()
	log.Println("Cron scheduler stopped")
}

// runAutoValidation is the job function that validates expired matches
func (s *Scheduler) runAutoValidation() {
	log.Println("Running auto-validation job...")
	
	// Check how many expired matches we have before processing
	expiredCount, err := s.autoValidationService.GetExpiredMatchesCount()
	if err != nil {
		log.Printf("Error checking expired matches count: %v", err)
		return
	}
	
	if expiredCount == 0 {
		log.Println("No expired matches to validate")
		return
	}
	
	log.Printf("Found %d expired matches to validate", expiredCount)
	
	// Run the validation
	err = s.autoValidationService.ValidateExpiredMatches()
	if err != nil {
		log.Printf("Error during auto-validation: %v", err)
		return
	}
	
	log.Println("Auto-validation job completed successfully")
}

// RunNow manually triggers the auto-validation job (useful for testing)
func (s *Scheduler) RunNow() {
	log.Println("Manually triggering auto-validation job...")
	s.runAutoValidation()
}