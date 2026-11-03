package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type SMTP struct {
	host       string
	port       string
	username   string
	password   string
	from       string
	requireTLS bool
}

func NewSMTP(host, port, username, password, from string, requireTLS bool) *SMTP {
	return &SMTP{
		host: host, port: port, username: username, password: password,
		from: from, requireTLS: requireTLS,
	}
}

func (client *SMTP) Send(to, subject, body string) error {
	fromAddress, err := mail.ParseAddress(client.from)
	if err != nil || strings.ContainsAny(client.from, "\r\n") {
		return fmt.Errorf("invalid SMTP sender address")
	}
	toAddress, err := mail.ParseAddress(to)
	if err != nil || toAddress.Address != to || strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid SMTP recipient address")
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid email subject")
	}
	address := net.JoinHostPort(client.host, client.port)
	connection, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	server, err := smtp.NewClient(connection, client.host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer server.Close()
	if supported, _ := server.Extension("STARTTLS"); supported {
		if err := server.StartTLS(&tls.Config{ServerName: client.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	} else if client.requireTLS {
		return fmt.Errorf("SMTP server does not advertise required STARTTLS")
	}
	if client.username != "" {
		if err := server.Auth(smtp.PlainAuth("", client.username, client.password, client.host)); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := server.Mail(fromAddress.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := server.Rcpt(toAddress.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := server.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	message := "From: " + fromAddress.String() + "\r\nTo: " + toAddress.String() + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n"
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := server.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}
