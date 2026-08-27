package shared

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"sync"

	"gopkg.in/gomail.v2"
)

//go:embed templates/*.html
var emailTemplates embed.FS

type ResetPasswordData struct {
	To           string
	Name         string
	RedirectLink string
	ExpiresIn    string
}

type emailJob struct {
	to      string
	subject string
	body    string
}

type mailer struct {
	dialer    *gomail.Dialer
	from      string
	templates *template.Template
	queue     chan emailJob
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

type MailerConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Workers  int
	BufSize  int
}
type MailerService interface {
	SendResetPasswordEmail(payload ResetPasswordData) error
	Close()
}

func NewMailer(cfg *MailerConfig) (MailerService, error) {
	tmpl, err := template.ParseFS(emailTemplates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("erro ao compilar templates: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bufSize := cfg.BufSize
	if bufSize <= 0 {
		bufSize = 100
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 2
	}

	m := &mailer{
		dialer:    gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password),
		from:      cfg.From,
		templates: tmpl,
		queue:     make(chan emailJob, bufSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	for i := 0; i < workers; i++ {
		m.wg.Add(1)
		go m.worker(i + 1)
	}

	return m, nil
}
func (m *mailer) SendResetPasswordEmail(payload ResetPasswordData) error {
	var buf bytes.Buffer
	if err := m.templates.ExecuteTemplate(&buf, "reset_password.html", payload); err != nil {
		return fmt.Errorf("falha ao renderizar template de reset: %w", err)
	}

	job := emailJob{
		to:      payload.To,
		subject: "Recuperação de Senha",
		body:    buf.String(),
	}

	select {
	case m.queue <- job:
		return nil
	default:
		return fmt.Errorf("fila de emails cheia, mensagem descartada")
	}
}

func (m *mailer) Close() {
	m.cancel()
	close(m.queue)
	m.wg.Wait()
}

func (m *mailer) deliver(job emailJob) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", job.to)
	msg.SetHeader("Subject", job.subject)
	msg.SetBody("text/html", job.body)

	if err := m.dialer.DialAndSend(msg); err != nil {
		log.Printf("[Mailer Worker] Falha ao enviar email para %s: %v", job.to, err)
		return
	}
	log.Printf("[Mailer Worker] Email enviado para %s", job.to)
}

func (m *mailer) worker(id int) {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			for job := range m.queue {
				m.deliver(job)
			}
			return
		case job, ok := <-m.queue:
			if !ok {
				return
			}
			m.deliver(job)
		}
	}
}
