package service

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type smtpSettingsReader interface {
	GetSettingsByGroup(group string) ([]SettingItem, error)
	DecryptSecretSetting(key string, pin string) (string, error)
}

type SMTPService struct {
	settings smtpSettingsReader
}

type SMTPConfig struct {
	Enabled   bool
	FromEmail string
	Host      string
	Port      string
	Username  string
	Password  string
	Security  string // none | ssl | tls | starttls
}

type SMTPTestResult struct {
	Success bool   `json:"success"`
	To      string `json:"to"`
	Message string `json:"message"`
}

type SMTPSendRequest struct {
	To          []string `json:"to"`
	Subject     string   `json:"subject"`
	PlainBody   string   `json:"plainBody"`
	HTMLBody    string   `json:"htmlBody"`
	Attachments []string `json:"attachments"`
}

func NewSMTPService(settings smtpSettingsReader) *SMTPService {
	return &SMTPService{settings: settings}
}

func (s *SMTPService) TestSendToSelf(pin string) (*SMTPTestResult, error) {
	cfg, err := s.loadConfig(pin)
	if err != nil {
		return nil, err
	}

	subject := "S2QT SMTP 테스트 메일"
	plainBody := strings.TrimSpace(`
안녕하세요.

이 메일은 S2QT SMTP 테스트 메일입니다.
설정된 SMTP 정보로 정상 발송되었는지 확인해 주세요.

감사합니다.
`)
	htmlBody := `
<p>안녕하세요.</p>
<p>이 메일은 <strong>S2QT SMTP 테스트 메일</strong>입니다.</p>
<p>설정된 SMTP 정보로 정상 발송되었는지 확인해 주세요.</p>
<p>감사합니다.</p>
`

	req := SMTPSendRequest{
		To:        []string{cfg.FromEmail},
		Subject:   subject,
		PlainBody: plainBody,
		HTMLBody:  htmlBody,
	}

	if err := s.SendMail(pin, req); err != nil {
		return nil, err
	}

	return &SMTPTestResult{
		Success: true,
		To:      cfg.FromEmail,
		Message: "테스트 메일을 발송했습니다.",
	}, nil
}

func (s *SMTPService) SendMail(pin string, req SMTPSendRequest) error {
	if s == nil || s.settings == nil {
		return fmt.Errorf("smtp service is not initialized")
	}

	cfg, err := s.loadConfig(pin)
	if err != nil {
		return err
	}

	toList := normalizeEmails(req.To)
	if len(toList) == 0 {
		return fmt.Errorf("수신자 이메일이 비어 있습니다")
	}

	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		return fmt.Errorf("메일 제목이 비어 있습니다")
	}

	if strings.TrimSpace(req.PlainBody) == "" && strings.TrimSpace(req.HTMLBody) == "" {
		return fmt.Errorf("메일 본문이 비어 있습니다")
	}

	msg, err := buildMIMEMessage(cfg.FromEmail, toList, subject, req.PlainBody, req.HTMLBody, req.Attachments)
	if err != nil {
		return err
	}

	return sendSMTPMessage(cfg, toList, msg)
}

func (s *SMTPService) loadConfig(pin string) (*SMTPConfig, error) {
	if s == nil || s.settings == nil {
		return nil, fmt.Errorf("smtp service settings reader is nil")
	}

	smtpItems, err := s.settings.GetSettingsByGroup("smtp")
	if err != nil {
		return nil, fmt.Errorf("smtp 설정 조회 실패: %w", err)
	}
	userItems, err := s.settings.GetSettingsByGroup("user")
	if err != nil {
		return nil, fmt.Errorf("user 설정 조회 실패: %w", err)
	}

	smtpMap := smtpSettingMap(smtpItems)
	userMap := smtpSettingMap(userItems)

	fromEmail := stringsTrim(smtpFirstNonEmpty(
		userMap["user.email"],
		smtpMap["smtp.from_email"],
	))
	host := stringsTrim(smtpMap["smtp.host"])
	port := stringsTrim(smtpFirstNonEmpty(smtpMap["smtp.port"], "587"))
	username := stringsTrim(smtpMap["smtp.username"])
	security := strings.ToLower(stringsTrim(smtpFirstNonEmpty(smtpMap["smtp.security"], "tls")))
	enabled := boolFromSMTPSetting(smtpMap["smtp.enabled"])

	password, err := s.settings.DecryptSecretSetting("smtp.password", pin)
	if err != nil {
		return nil, fmt.Errorf("smtp 비밀번호 조회 실패: %w", err)
	}
	password = stringsTrim(password)

	cfg := &SMTPConfig{
		Enabled:   enabled,
		FromEmail: fromEmail,
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		Security:  security,
	}

	if err := validateSMTPConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateSMTPConfig(cfg *SMTPConfig) error {
	if cfg == nil {
		return fmt.Errorf("smtp config is nil")
	}
	if !cfg.Enabled {
		return fmt.Errorf("SMTP 사용이 꺼져 있습니다")
	}
	if cfg.FromEmail == "" {
		return fmt.Errorf("발신 이메일이 비어 있습니다. 기본 정보에서 이메일을 먼저 설정해 주세요")
	}
	if cfg.Host == "" {
		return fmt.Errorf("SMTP 서버가 비어 있습니다")
	}
	if cfg.Port == "" {
		return fmt.Errorf("SMTP 포트가 비어 있습니다")
	}
	if cfg.Username == "" {
		return fmt.Errorf("SMTP 사용자명이 비어 있습니다")
	}
	if cfg.Password == "" {
		return fmt.Errorf("SMTP 앱 비밀번호가 비어 있습니다")
	}

	switch cfg.Security {
	case "none", "ssl", "tls", "starttls":
		return nil
	default:
		return fmt.Errorf("지원하지 않는 SMTP 보안 방식입니다: %s", cfg.Security)
	}
}

func smtpSettingMap(items []SettingItem) map[string]string {
	out := make(map[string]string, len(items))
	for _, item := range items {
		out[item.Key] = item.Value
	}
	return out
}

func boolFromSMTPSetting(v string) bool {
	return strings.EqualFold(stringsTrim(v), "true")
}

func smtpFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if stringsTrim(v) != "" {
			return stringsTrim(v)
		}
	}
	return ""
}

func normalizeEmails(values []string) []string {
	var out []string
	for _, raw := range values {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			email := stringsTrim(p)
			if email != "" {
				out = append(out, email)
			}
		}
	}
	return out
}

func buildMIMEMessage(from string, to []string, subject string, plainBody string, htmlBody string, attachments []string) ([]byte, error) {
	mixedBoundary := fmt.Sprintf("s2qt-mixed-%d", time.Now().UnixNano())
	altBoundary := fmt.Sprintf("s2qt-alt-%d", time.Now().UnixNano())

	var buf bytes.Buffer

	writeHeaderLine(&buf, "From", from)
	writeHeaderLine(&buf, "To", strings.Join(to, ", "))
	writeHeaderLine(&buf, "Subject", mime.QEncoding.Encode("utf-8", subject))
	writeHeaderLine(&buf, "MIME-Version", "1.0")
	writeHeaderLine(&buf, "Content-Type", fmt.Sprintf(`multipart/mixed; boundary="%s"`, mixedBoundary))
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", mixedBoundary))
	buf.WriteString(fmt.Sprintf(`Content-Type: multipart/alternative; boundary="%s"`+"\r\n\r\n", altBoundary))

	plain := stringsTrim(plainBody)
	if plain == "" && strings.TrimSpace(htmlBody) != "" {
		plain = stripHTMLLikeText(htmlBody)
	}
	if plain != "" {
		buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		buf.WriteString(`Content-Type: text/plain; charset="utf-8"` + "\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(plain + "\r\n")
	}

	html := strings.TrimSpace(htmlBody)
	if html != "" {
		buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		buf.WriteString(`Content-Type: text/html; charset="utf-8"` + "\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(html + "\r\n")
	}

	buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))

	for _, path := range attachments {
		filePath := stringsTrim(path)
		if filePath == "" {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("첨부 파일 읽기 실패 (%s): %w", filePath, err)
		}

		filename := filepath.Base(filePath)
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		buf.WriteString(fmt.Sprintf("--%s\r\n", mixedBoundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", contentType, escapeHeaderValue(filename)))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", escapeHeaderValue(filename)))

		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(encoded, data)
		writeBase64Lines(&buf, encoded)
		buf.WriteString("\r\n")
	}

	buf.WriteString(fmt.Sprintf("--%s--\r\n", mixedBoundary))
	return buf.Bytes(), nil
}

func writeHeaderLine(buf *bytes.Buffer, key string, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func writeBase64Lines(buf *bytes.Buffer, encoded []byte) {
	const lineLen = 76
	for len(encoded) > 0 {
		n := lineLen
		if len(encoded) < n {
			n = len(encoded)
		}
		buf.Write(encoded[:n])
		buf.WriteString("\r\n")
		encoded = encoded[n:]
	}
}

func escapeHeaderValue(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v)
}

func stripHTMLLikeText(v string) string {
	r := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n\n",
		"<p>", "",
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
	)
	text := r.Replace(v)
	for {
		start := strings.Index(text, "<")
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end < 0 {
			break
		}
		text = text[:start] + text[start+end+1:]
	}
	return stringsTrim(text)
}

func sendSMTPMessage(cfg *SMTPConfig, to []string, msg []byte) error {
	addr := cfg.Host + ":" + cfg.Port

	switch cfg.Security {
	case "ssl":
		return sendWithImplicitTLS(cfg, addr, to, msg)
	case "none":
		return sendWithoutTLS(cfg, addr, to, msg)
	case "tls", "starttls":
		return sendWithSTARTTLS(cfg, addr, to, msg)
	default:
		return fmt.Errorf("지원하지 않는 SMTP 보안 방식입니다: %s", cfg.Security)
	}
}

func sendWithSTARTTLS(cfg *SMTPConfig, addr string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP 서버 연결 실패: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP HELLO 실패: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS 실패: %w", err)
		}
	} else {
		return fmt.Errorf("SMTP 서버가 STARTTLS를 지원하지 않습니다")
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 인증 실패: %w", err)
	}

	return sendWithClient(client, cfg.FromEmail, to, msg)
}

func sendWithoutTLS(cfg *SMTPConfig, addr string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP 서버 연결 실패: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP HELLO 실패: %w", err)
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 인증 실패: %w", err)
	}

	return sendWithClient(client, cfg.FromEmail, to, msg)
}

func sendWithImplicitTLS(cfg *SMTPConfig, addr string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: cfg.Host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("SSL SMTP 서버 연결 실패: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP 클라이언트 생성 실패: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP HELLO 실패: %w", err)
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 인증 실패: %w", err)
	}

	return sendWithClient(client, cfg.FromEmail, to, msg)
}

func sendWithClient(client *smtp.Client, from string, to []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM 실패: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO 실패 (%s): %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA 시작 실패: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("메일 본문 쓰기 실패: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("메일 본문 종료 실패: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP 종료 실패: %w", err)
	}
	return nil
}
