package notifications

import (
	"crypto/tls"
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
	subject := "Focus admin invite"
	body := fmt.Sprintf("You are invited to Focus admin panel.\n\nOpen this link to accept the invitation:\n%s\n", inviteURL)
	msg := []byte("From: " + m.from + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		body)
	return m.sendMail(email, msg)
}

func (m *SMTPInviteMailer) sendMail(to string, body []byte) error {
	addr := m.host + ":" + m.port
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	if m.port == "465" {
		tlsConfig := &tls.Config{ServerName: m.host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, m.host)
		if err != nil {
			return err
		}
		defer client.Close()
		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return err
			}
		}
		if err = client.Mail(m.from); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		if err != nil {
			return err
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, body)
}
