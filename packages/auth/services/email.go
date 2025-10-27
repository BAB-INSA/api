package services

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/go-mail/mail/v2"
)

// EmailService interface pour l'envoi d'emails
type EmailService interface {
	SendPasswordResetEmail(to, resetURL string) error
}

// LogEmailService implémentation qui log les emails (pour développement)
type LogEmailService struct{}

// NewLogEmailService crée une nouvelle instance du service email de développement
func NewLogEmailService() *LogEmailService {
	return &LogEmailService{}
}

// SendPasswordResetEmail envoie un email de reset de mot de passe (version dev qui log)
func (s *LogEmailService) SendPasswordResetEmail(to, resetURL string) error {
	subject := "Réinitialisation de votre mot de passe"
	body := fmt.Sprintf(`Bonjour,

Vous avez demandé la réinitialisation de votre mot de passe.
Cliquez sur le lien suivant pour créer un nouveau mot de passe :

%s

Ce lien est valide pendant 2 heures.

Si vous n'avez pas fait cette demande, ignorez ce message.

Cordialement,
L'équipe`, resetURL)

	log.Printf("=== EMAIL SENT ===")
	log.Printf("To: %s", to)
	log.Printf("Subject: %s", subject)
	log.Printf("Body: %s", body)
	log.Printf("=================")
	return nil
}

// SMTPEmailService pour l'envoi réel d'emails via SMTP
type SMTPEmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPEmailService crée une nouvelle instance du service email SMTP
func NewSMTPEmailService() (*SMTPEmailService, error) {
	mailDSN := os.Getenv("MAIL_DSN")
	if mailDSN == "" {
		return nil, fmt.Errorf("MAIL_DSN environment variable is required")
	}

	u, err := url.Parse(mailDSN)
	if err != nil {
		return nil, fmt.Errorf("invalid MAIL_DSN format: %v", err)
	}

	port := 25 // Port par défaut
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port in MAIL_DSN: %v", err)
		}
	}

	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	from := "noreply@example.com" // Email par défaut
	if envSender := os.Getenv("MAILER_ENVELOPE_SENDER"); envSender != "" {
		from = envSender
	} else if username != "" {
		from = username
	}

	return &SMTPEmailService{
		host:     u.Hostname(),
		port:     port,
		username: username,
		password: password,
		from:     from,
	}, nil
}

// SendPasswordResetEmail envoie un email de reset de mot de passe via SMTP
func (s *SMTPEmailService) SendPasswordResetEmail(to, resetURL string) error {
	subject := "Réinitialisation de votre mot de passe"
	body := fmt.Sprintf(`Bonjour,

Vous avez demandé la réinitialisation de votre mot de passe.
Cliquez sur le lien suivant pour créer un nouveau mot de passe :

%s

Ce lien est valide pendant 2 heures.

Si vous n'avez pas fait cette demande, ignorez ce message.

Cordialement,
L'équipe`, resetURL)

	m := mail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := mail.NewDialer(s.host, s.port, s.username, s.password)

	// Pour les serveurs locaux comme Mailpit, désactiver TLS
	if s.host == "localhost" || s.host == "127.0.0.1" {
		d.TLSConfig = nil
	}

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}

	log.Printf("Email sent successfully to %s via SMTP (%s:%d)", to, s.host, s.port)
	return nil
}

// NewEmailService crée le service email approprié selon la configuration
func NewEmailService() EmailService {
	// Essayer d'abord le service SMTP
	if smtpService, err := NewSMTPEmailService(); err == nil {
		return smtpService
	}

	// Fallback vers le service de log pour le développement
	log.Println("MAIL_DSN not configured, using log email service")
	return NewLogEmailService()
}
