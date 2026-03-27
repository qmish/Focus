package notifications

import (
	"fmt"
	"net/smtp"
	"strings"
)

type InviteMailer interface {
	SendInvite(email, inviteURL string) error
}

type SMTPInviteMailer struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPInviteMailer(host, port, user, pass, from string) *SMTPInviteMailer {
	return &SMTPInviteMailer{
		host: strings.TrimSpace(host),
		port: strings.TrimSpace(port),
		user: strings.TrimSpace(user),
		pass: pass,
		from: strings.TrimSpace(from),
	}
}

func (m *SMTPInviteMailer) SendInvite(email, inviteURL string) error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	subject := "Focus admin invite"
	body := fmt.Sprintf("You are invited to Focus admin panel.\n\nOpen this link to accept the invitation:\n%s\n", inviteURL)
	msg := []byte("To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		body)
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(addr, auth, m.from, []string{email}, msg)
}
