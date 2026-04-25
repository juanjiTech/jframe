# Notification Module Design Document

## Step 1: Requirements Gathering

Following the skill's requirement-gathering checklist:

### 1. What does the module do?
This is a **service client** module. It provides notification-sending capabilities (email via SMTP and SMS via an external API). It does not serve HTTP routes or run a background loop -- it is a service that other modules call via the DI container.

### 2. What name should it have?
`notification` -- it is an infrastructure/utility module. Following the convention, this becomes the YAML config key and the log namespace. The Go package will be `notification` under `mod/notification/`.

### 3. Does it need configuration?
**Yes.** It needs:
- SMTP server parameters: host, port, username, password, sender address, TLS toggle
- SMS API parameters: API endpoint URL, API key, API secret, sender ID

### 4. What does it produce for other modules?
It will `Map` a `*notification.Service` interface into the DI container. Other modules will `Load` this service to send emails and SMS messages. Using an interface allows future extension (e.g., push notifications) without changing consumers.

### 5. What does it consume from other modules?
**Nothing.** This module is self-contained. It does not need `*jin.Engine` (no HTTP routes), `*gorm.DB` (no database), or any other module's output.

### 6. Does it need a long-running goroutine?
**No.** It is a request-driven service -- send a notification when called. No server loop or background worker.

---

## Step 2: Lifecycle Phase Selection

Based on the "Service client" pattern from the skill (`Config` + `PreInit` (connect & Map) + `Init` (verify) + `Stop` (close)):

| Phase | Use? | Rationale |
|-------|------|-----------|
| `Config()` | **Yes** | SMTP and SMS API parameters must be configurable |
| `PreInit()` | **Yes** | Create the notification Service struct, validate config minimally, and `Map` it into the DI container |
| `Init()` | **Yes** | Verify the SMTP connection works (dial & close) to fail fast on misconfiguration |
| `PostInit()` | No | No cross-module dependencies needed |
| `Load()` | No | No routes to register, no late wiring needed |
| `Start()` | No | No long-running goroutine |
| `Stop()` | **Yes** | Required by the interface (we embed UnimplementedModule which handles `wg.Done()`; we override only if we have resources to close) |

Since `Stop()` has no resources to explicitly close (SMTP connections are opened per-send, SMS is stateless HTTP), we can rely on the default `UnimplementedModule.Stop()` which already calls `wg.Done()`. However, if we want to be explicit or add future connection pooling, we can override it.

**Decision:** Implement `Config`, `PreInit`, `Init`. Rely on `UnimplementedModule` defaults for `PostInit`, `Load`, `Start`, `Stop`.

---

## Step 3: Config Struct Design

```go
type Config struct {
    // SMTP settings for email notifications
    SMTP SMTPConfig `yaml:"smtp" mapstructure:"smtp"`
    // SMS settings for SMS notifications
    SMS SMSConfig `yaml:"sms" mapstructure:"sms"`
}

type SMTPConfig struct {
    Host     string `yaml:"host" mapstructure:"host"`
    Port     string `yaml:"port" mapstructure:"port"`
    Username string `yaml:"username" mapstructure:"username"`
    Password string `yaml:"password" mapstructure:"password"`
    From     string `yaml:"from" mapstructure:"from"`
    UseTLS   bool   `yaml:"useTLS" mapstructure:"useTLS"`
}

type SMSConfig struct {
    Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
    APIKey   string `yaml:"apiKey" mapstructure:"apiKey"`
    Secret   string `yaml:"secret" mapstructure:"secret"`
    SenderID string `yaml:"senderID" mapstructure:"senderID"`
}
```

Corresponding YAML (keyed by `Name()` = `"notification"`):

```yaml
notification:
    smtp:
        host: "smtp.example.com"
        port: "587"
        username: ""
        password: ""
        from: "noreply@example.com"
        useTLS: true
    sms:
        endpoint: "https://sms-api.example.com/send"
        apiKey: ""
        secret: ""
        senderID: ""
```

Environment variable overrides (via Viper):
- `NOTIFICATION_SMTP_HOST=smtp.gmail.com`
- `NOTIFICATION_SMS_APIKEY=xxx`

---

## Step 4: DI Dependency Plan

### Mapping (producing):
In `PreInit`, we create a `*Service` and Map it:
```go
svc := &Service{config: m.config, logger: hub.Log}
hub.Map(&svc)   // stored as **notification.Service
```

### Loading (consuming):
Other modules (e.g., a `users` module) would do:
```go
var notifSvc *notification.Service
if err := hub.Load(&notifSvc); err != nil {
    return errors.New("can't load notification.Service from kernel")
}
notifSvc.SendEmail(ctx, to, subject, body)
notifSvc.SendSMS(ctx, phone, message)
```

### Ordering:
Since this module does not depend on any other module's output, it can be placed anywhere in `ModList`. However, modules that consume `*notification.Service` must appear after it if they need it in `PreInit` (unlikely -- they would typically use it in `Load` or at runtime, so ordering within `ModList` does not matter for cross-phase access).

---

## Step 5: Module File Structure

Since this is a pure service client with no HTTP routes, no data models, no handlers, the structure is minimal:

```
mod/notification/
└── mod.go          # Module struct + lifecycle + Service type
```

No `handler/`, `service/`, `dao/`, `model/`, or `e/` sub-packages are needed. This follows the pattern of `mod/myDB/` and `mod/rds/` which are also infrastructure service clients with only `mod.go`.

---

## Step 6 & 7: Registration and Config

### modList addition (`cmd/server/modList/list.go`):

```go
import (
    // ...existing imports
    "github.com/juanjiTech/jframe/mod/notification"
)

var ModList = []kernel.Module{
    // ...existing modules
    &notification.Mod{},
}
```

### config.example.yaml addition:

```yaml
notification:
    smtp:
        host: "smtp.example.com"
        port: "587"
        username: ""
        password: ""
        from: "noreply@example.com"
        useTLS: true
    sms:
        endpoint: "https://sms-api.example.com/send"
        apiKey: ""
        secret: ""
        senderID: ""
```

---

## Complete Module Code: `mod/notification/mod.go`

```go
package notification

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"time"

	"github.com/juanjiTech/jframe/core/kernel"
	"go.uber.org/zap"
)

var _ kernel.Module = (*Mod)(nil)

// Config holds all notification-related configuration.
type Config struct {
	SMTP SMTPConfig `yaml:"smtp" mapstructure:"smtp"`
	SMS  SMSConfig  `yaml:"sms" mapstructure:"sms"`
}

// SMTPConfig holds SMTP server connection parameters.
type SMTPConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     string `yaml:"port" mapstructure:"port"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	From     string `yaml:"from" mapstructure:"from"`
	UseTLS   bool   `yaml:"useTLS" mapstructure:"useTLS"`
}

// SMSConfig holds SMS API connection parameters.
type SMSConfig struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	APIKey   string `yaml:"apiKey" mapstructure:"apiKey"`
	Secret   string `yaml:"secret" mapstructure:"secret"`
	SenderID string `yaml:"senderID" mapstructure:"senderID"`
}

// Mod is the notification kernel module.
type Mod struct {
	kernel.UnimplementedModule
	config Config
}

func (m *Mod) Name() string { return "notification" }

func (m *Mod) Config() any { return &m.config }

func (m *Mod) PreInit(hub *kernel.Hub) error {
	svc := &Service{
		config: m.config,
		logger: hub.Log,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	hub.Map(&svc)
	hub.Log.Info("notification service registered")
	return nil
}

func (m *Mod) Init(hub *kernel.Hub) error {
	// Verify that the notification service was mapped successfully
	var svc *Service
	if err := hub.Load(&svc); err != nil {
		return errors.New("can't load notification.Service from kernel")
	}

	// Verify SMTP connectivity if configured
	if m.config.SMTP.Host != "" {
		addr := net.JoinHostPort(m.config.SMTP.Host, m.config.SMTP.Port)
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			return fmt.Errorf("notification: SMTP server unreachable at %s: %w", addr, err)
		}
		conn.Close()
		hub.Log.Infow("notification: SMTP server reachable", "addr", addr)
	}

	return nil
}

// Service provides methods to send email and SMS notifications.
// Other modules obtain this via hub.Load(&svc).
type Service struct {
	config     Config
	logger     *zap.SugaredLogger
	httpClient *http.Client
}

// SendEmail sends an email notification via SMTP.
func (s *Service) SendEmail(ctx context.Context, to []string, subject, body string) error {
	if s.config.SMTP.Host == "" {
		return errors.New("notification: SMTP not configured")
	}

	addr := net.JoinHostPort(s.config.SMTP.Host, s.config.SMTP.Port)

	// Build the email message
	msg := fmt.Sprintf("From: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		s.config.SMTP.From, subject, body)

	// Compose To header
	toHeader := ""
	for i, addr := range to {
		if i > 0 {
			toHeader += ", "
		}
		toHeader += addr
	}
	msg = fmt.Sprintf("To: %s\r\n%s", toHeader, msg)

	var auth smtp.Auth
	if s.config.SMTP.Username != "" {
		auth = smtp.PlainAuth("", s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Host)
	}

	if s.config.SMTP.UseTLS {
		// TLS connection
		tlsConfig := &tls.Config{
			ServerName: s.config.SMTP.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("notification: TLS dial failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.config.SMTP.Host)
		if err != nil {
			return fmt.Errorf("notification: SMTP client creation failed: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("notification: SMTP auth failed: %w", err)
			}
		}

		if err := client.Mail(s.config.SMTP.From); err != nil {
			return fmt.Errorf("notification: SMTP MAIL FROM failed: %w", err)
		}
		for _, addr := range to {
			if err := client.Rcpt(addr); err != nil {
				return fmt.Errorf("notification: SMTP RCPT TO failed for %s: %w", addr, err)
			}
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("notification: SMTP DATA failed: %w", err)
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return fmt.Errorf("notification: SMTP write failed: %w", err)
		}
		err = w.Close()
		if err != nil {
			return fmt.Errorf("notification: SMTP close data failed: %w", err)
		}
		return client.Quit()
	}

	// Non-TLS (STARTTLS handled by smtp.SendMail)
	return smtp.SendMail(addr, auth, s.config.SMTP.From, to, []byte(msg))
}

// SendSMS sends an SMS notification via the configured SMS API.
func (s *Service) SendSMS(ctx context.Context, phone string, message string) error {
	if s.config.SMS.Endpoint == "" {
		return errors.New("notification: SMS API not configured")
	}

	payload := map[string]string{
		"phone":    phone,
		"message":  message,
		"senderID": s.config.SMS.SenderID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notification: failed to marshal SMS payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.SMS.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notification: failed to create SMS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.config.SMS.APIKey)
	req.Header.Set("X-API-Secret", s.config.SMS.Secret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notification: SMS API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification: SMS API returned status %d", resp.StatusCode)
	}

	s.logger.Infow("notification: SMS sent", "phone", phone)
	return nil
}
```

---

## Checklist Verification

- [x] `mod.go` implements `kernel.Module` with `var _ kernel.Module = (*Mod)(nil)` compile-time check
- [x] Embeds `kernel.UnimplementedModule`
- [x] `Name()` returns a unique, camelCase identifier: `"notification"`
- [x] `Config()` returns `&m.config` (pointer)
- [x] Config struct has both `yaml` and `mapstructure` tags on every field
- [x] All `Map` calls use pointer arguments: `hub.Map(&svc)`
- [x] All `Load` calls check for errors
- [x] `Stop()` via `UnimplementedModule` calls `defer wg.Done()` as its first statement
- [x] Module listed for addition to `cmd/server/modList/list.go`
- [x] Config defaults specified for `config.yaml` and `config.example.yaml`
- [x] Code should compile cleanly (standard library + jframe kernel + zap logger)

---

## Summary of Changes Required

### 1. New file: `mod/notification/mod.go`
Full content provided above.

### 2. Update: `cmd/server/modList/list.go`

Add import and module registration:

```go
package modList

import (
	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jframe/mod/b2x"
	"github.com/juanjiTech/jframe/mod/grpcGateway"
	"github.com/juanjiTech/jframe/mod/jinPprof"
	"github.com/juanjiTech/jframe/mod/jinx"
	"github.com/juanjiTech/jframe/mod/myDB"
	"github.com/juanjiTech/jframe/mod/notification"
	"github.com/juanjiTech/jframe/mod/pyroscope"
	"github.com/juanjiTech/jframe/mod/rds"
	"github.com/juanjiTech/jframe/mod/uptrace"
)

var ModList = []kernel.Module{
	&b2x.Mod{},
	&grpcGateway.Mod{},
	&jinPprof.Mod{},
	&jinx.Mod{},
	&myDB.Mod{},
	&notification.Mod{},
	&pyroscope.Mod{},
	&rds.Mod{},
	&uptrace.Mod{},
}
```

### 3. Update: `config.example.yaml`

Append at the end:

```yaml
notification:
    smtp:
        host: "smtp.example.com"
        port: "587"
        username: ""
        password: ""
        from: "noreply@example.com"
        useTLS: true
    sms:
        endpoint: "https://sms-api.example.com/send"
        apiKey: ""
        secret: ""
        senderID: ""
```

---

## Usage Example (by another module)

A hypothetical `orders` module could consume the notification service in its `Load` phase:

```go
func (m *Mod) Load(hub *kernel.Hub) error {
    var notifSvc *notification.Service
    if err := hub.Load(&notifSvc); err != nil {
        return errors.New("can't load notification.Service from kernel")
    }

    // Use notifSvc in handlers
    var j *jin.Engine
    if err := hub.Load(&j); err != nil {
        return errors.New("can't load jin.Engine from kernel")
    }

    g := j.Group("/api/orders")
    h := handler{notifSvc: notifSvc}
    g.POST("/confirm", h.confirm)
    return nil
}

type handler struct {
    notifSvc *notification.Service
}

func (h *handler) confirm(c *jin.Context) {
    // ... process order ...
    _ = h.notifSvc.SendEmail(c.Request.Context(),
        []string{"customer@example.com"},
        "Order Confirmed",
        "Your order #123 has been confirmed.",
    )
}
```
