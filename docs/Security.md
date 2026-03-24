# Security Guide

**Версия:** 1.0  
**Дата:** 24 марта 2026 г.  
**Статус:** Черновик

---

## 1. Модель угроз (Threat Model)

### 1.1. Активы

| Актив | Критичность | Описание |
|-------|-------------|----------|
| JWT токены | Высокая | Доступ к сессиям пользователей |
| Пользовательские данные | Высокая | Email, имена, ID из Keycloak |
| Сообщения чата | Средняя | Корпоративная переписка |
| Записи звонков | Высокая | Конфиденциальные встречи |
| Календари Exchange | Средняя | Расписание встреч |
| Секреты API | Высокая | Ключи интеграций |

### 1.2. Актеры угроз

| Актер | Возможности | Цели |
|-------|-------------|------|
| Внешний злоумышленник | Сканирование портов, DDoS, XSS | Кража данных, доступ к системе |
| Внутренний злоумышленник | Легальный доступ, социнженерия | Кража конфиденциальной информации |
| Скомпрометированный сервис | Доступ к БД, внутренняя сеть | Горизонтальное перемещение |
| MITM атака | Перехват трафика | Кража токенов, данных |

### 1.3. Векторы атак и контрмеры

| Угроза | Вероятность | Влияние | Контрмеры |
|--------|-------------|---------|-----------|
| SQL Injection | Средняя | Высокое | Prepared statements (GORM), валидация |
| XSS | Средняя | Высокое | CSP, санитизация, фреймворк (React) |
| CSRF | Низкая | Среднее | SameSite cookies, CSRF tokens |
| JWT подделка | Низкая | Критическое | HS256 подписи, короткое время жизни |
| DDoS | Высокая | Высокое | Rate limiting, WAF, CDN |
| Credential stuffing | Средняя | Высокое | MFA (Keycloak), lockout policy |
| Privilege escalation | Средняя | Критическое | RBAC, принцип наименьших привилегий |

---

## 2. Аутентификация и авторизация

### 2.1. OIDC Flow (Keycloak)

```
┌─────────┐      ┌─────────┐      ┌─────────┐      ┌─────────┐
│  User   │      │Frontend │      │  Go API │      │Keycloak │
└────┬────┘      └────┬────┘      └────┬────┘      └────┬────┘
     │                │                │                │
     │  1. Открыть    │                │                │
     │───────────────>│                │                │
     │                │                │                │
     │  2. Редирект на Keycloak (PKCE)                   │
     │<──────────────────────────────────────────────────│
     │                │                │                │
     │  3. Login/Password                                │
     │──────────────────────────────────────────────────>│
     │                │                │                │
     │  4. Auth Code + code_verifier                     │
     │──────────────────────────────────────────────────>│
     │                │                │                │
     │  5. Access + ID + Refresh Tokens                  │
     │<──────────────────────────────────────────────────│
     │                │                │                │
     │  6. POST /auth/callback (code)                    │
     │───────────────>│                │                │
     │                │                │                │
     │  7. Обмен кода на токен                           │
     │───────────────────────────────>│                │
     │                │                │                │
     │  8. Session JWT                                   │
     │<───────────────────────────────│                │
     │                │                │                │
```

### 2.2. JWT Claims (Session)

```json
{
  "iss": "messenger-api",
  "sub": "user-uuid",
  "aud": "messenger-frontend",
  "exp": 1709312400,
  "iat": 1709308800,
  "email": "user@company.com",
  "name": "User Name",
  "roles": ["user", "moderator"],
  "keycloak_id": "kc-uuid"
}
```

### 2.3. JWT Claims (Jitsi)

```json
{
  "iss": "jitsi",
  "aud": "jitsi",
  "exp": 1709341200,
  "room": "room-name",
  "context": {
    "user": {
      "id": "user-uuid",
      "name": "User Name",
      "email": "user@company.com",
      "moderator": true
    }
  }
}
```

### 2.4. RBAC матрица

| Роль | Чат | Звонки | Календарь | Админка | Боты |
|------|-----|--------|-----------|---------|------|
| user | ✅ | ✅ | ✅ | ❌ | ✅ |
| moderator | ✅ | ✅ (host) | ✅ | ❌ | ✅ |
| admin | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## 3. Защита данных

### 3.1. Шифрование in-transit

| Компонент | Протокол | TLS версия |
|-----------|----------|------------|
| Frontend ↔ API | HTTPS | TLS 1.3 |
| API ↔ Keycloak | HTTPS | TLS 1.3 |
| API ↔ Graph API | HTTPS | TLS 1.3 |
| Frontend ↔ Jitsi | HTTPS/WSS | TLS 1.3 |
| Jitsi ↔ Prosody | XMPP/TLS | TLS 1.2+ |
| API ↔ PostgreSQL | SQL over TLS | TLS 1.2+ |
| API ↔ Redis | Redis over TLS | TLS 1.2+ |

### 3.2. Шифрование at-rest

| Компонент | Метод | Ключи |
|-----------|-------|-------|
| PostgreSQL | TDE (Transparent Data Encryption) | AWS KMS / Vault |
| Redis | AES-256 (при persistence) | AWS KMS / Vault |
| MinIO | SSE-S3 / SSE-KMS | AWS KMS / Vault |
| Backup файлы | GPG шифрование | GPG ключи |

### 3.3. Хранение секретов

**Kubernetes Secrets:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-secrets
  namespace: messenger
type: Opaque
stringData:
  keycloak-client-secret: xxx
  exchange-client-secret: xxx
  jwt-secret: xxx
```

**HashiCorp Vault (production):**
```bash
# Запись секрета
vault kv put secret/messenger/api \
  keycloak-client-secret=xxx \
  exchange-client-secret=xxx \
  jwt-secret=xxx

# Чтение из Pod (через Vault Agent)
vault kv get secret/messenger/api
```

---

## 4. Security Headers

### 4.1. Frontend (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name chat.company.com;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' meet.company.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' wss://api.company.com https://keycloak.company.com https://meet.company.com;" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Permissions-Policy "geolocation=(), microphone=(self), camera=(self)" always;

    # ... остальная конфигурация
}
```

### 4.2. CSP директивы

| Директива | Значение | Описание |
|-----------|----------|----------|
| default-src | 'self' | По умолчанию только свой домен |
| script-src | 'self' meet.company.com | Скрипты только свои и Jitsi |
| style-src | 'self' 'unsafe-inline' | Стили свои + inline (для React) |
| img-src | 'self' data: https: | Изображения свои, data URI, HTTPS |
| font-src | 'self' data: | Шрифты свои и data URI |
| connect-src | 'self' wss://api... | WebSocket и API соединения |
| frame-src | meet.company.com | Iframe только Jitsi |

---

## 5. Input Validation

### 5.1. Валидация на Go бэкенде

```go
// internal/api/validators.go
package api

import (
    "regexp"
    "strings"
    "unicode/utf8"
)

var (
    emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func ValidateEmail(email string) bool {
    return emailRegex.MatchString(email) && len(email) <= 254
}

func ValidateUUID(id string) bool {
    return uuidRegex.MatchString(strings.ToLower(id))
}

func ValidateRoomName(name string) bool {
    if len(name) < 3 || len(name) > 50 {
        return false
    }
    // Только буквы, цифры, дефис, подчёркивание
    valid := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    return valid.MatchString(name)
}

func SanitizeInput(input string) string {
    // Удаление control characters
    input = strings.Map(func(r rune) rune {
        if r >= 32 && r != 127 {
            return r
        }
        return -1
    }, input)
    
    // Trim whitespace
    input = strings.TrimSpace(input)
    
    // Ограничение длины
    if utf8.RuneCountInString(input) > 10000 {
        input = string([]rune(input)[:10000])
    }
    
    return input
}
```

### 5.2. Валидация сообщений

```go
type MessageInput struct {
    RoomID  string `json:"room_id" validate:"required,uuid"`
    Content string `json:"content" validate:"required,min=1,max=10000"`
    Type    string `json:"type" validate:"oneof=text image file"`
}

func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
    var input MessageInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    // Валидация через go-playground/validator
    if err := h.validator.Struct(input); err != nil {
        http.Error(w, "validation failed", http.StatusBadRequest)
        return
    }

    // Санитизация
    input.Content = SanitizeInput(input.Content)

    // Дальнейшая обработка...
}
```

---

## 6. Rate Limiting

### 6.1. Go Middleware

```go
// internal/auth/ratelimit.go
package auth

import (
    "sync"
    "time"
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func NewRateLimiter(rps rate.Limit, burst int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     rps,
        burst:    burst,
    }
}

func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    if limiter, exists := rl.limiters[key]; exists {
        return limiter
    }

    limiter := rate.NewLimiter(rl.rate, rl.burst)
    rl.limiters[key] = limiter
    return limiter
}

func (rl *RateLimiter) Allow(key string) bool {
    return rl.GetLimiter(key).Allow()
}

// Использование в middleware
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.RemoteAddr // или user ID из токена
            
            if !rl.Allow(key) {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### 6.2. Лимиты

| Endpoint | Лимит | Окно | Описание |
|----------|-------|------|----------|
| /auth/login | 5 | 1 мин | Защита от брутфорса |
| /auth/callback | 10 | 1 мин | OIDC callback |
| POST /rooms | 30 | 1 мин | Создание комнат |
| POST /messages | 60 | 1 мин | Отправка сообщений |
| POST /calendar/events | 20 | 1 мин | Создание встреч |
| GET /api/* | 100 | 1 мин | Чтение данных |
| WebSocket connect | 10 | 1 мин | Подключение WS |

---

## 7. Аудит и логирование

### 7.1. События для аудита

| Событие | Уровень | Данные |
|---------|---------|--------|
| Login success | INFO | user_id, IP, timestamp |
| Login failure | WARN | email, IP, timestamp, reason |
| Room created | INFO | user_id, room_id, room_name |
| Room deleted | INFO | user_id, room_id |
| Message sent | DEBUG | user_id, room_id, message_id (не content!) |
| Call started | INFO | user_id, room_id, jitsi_url |
| Calendar event created | INFO | user_id, event_id, attendees_count |
| Permission denied | WARN | user_id, resource, action |
| Rate limit exceeded | WARN | IP, endpoint, count |
| SQL error | ERROR | query_hash, error, duration |

### 7.2. Структура лога

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "service": "api-go",
  "trace_id": "abc123def456",
  "span_id": "xyz789",
  "user_id": "user-uuid",
  "session_id": "session-uuid",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "event": "room_created",
  "room_id": "room-uuid",
  "room_name": "meeting-123",
  "duration_ms": 45
}
```

### 7.3. Запрещённые данные в логах

- ❌ Пароли
- ❌ JWT токены (полностью)
- ❌ Содержимое сообщений
- ❌ Персональные данные без маскирования
- ❌ Ключи API и секреты

**Маскирование:**
```go
func MaskToken(token string) string {
    if len(token) < 10 {
        return "***"
    }
    return token[:3] + "..." + token[len(token)-3:]
}

func MaskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "***"
    }
    local := parts[0]
    if len(local) > 2 {
        local = local[:2] + "***"
    }
    return local + "@" + parts[1]
}
```

---

## 8. Security Testing

### 8.1. SAST (Static Application Security Testing)

**Инструменты:**
- `golangci-lint` с security checkers
- `gosec` — security scanner для Go
- `sonarqube` — комплексный анализ

**Запуск:**
```bash
# Go security scan
gosec ./...

# Lint с security проверками
golangci-lint run --enable=gosec

# В CI/CD
go test -race -coverprofile=coverage.out ./...
```

### 8.2. DAST (Dynamic Application Security Testing)

**Инструмент:** OWASP ZAP

**Сканирование:**
```bash
# Baseline scan
zap-baseline.py -t https://api.company.com

# Full scan
zap-full-scan.py -t https://api.company.com -r report.html

# API scan (OpenAPI)
zap-api-scan.py -t https://api.company.com/swagger.json -f openapi
```

### 8.3. Dependency Scanning

```bash
# Go модули
govulncheck ./...

# npm зависимости
npm audit

# Контейнеры
trivy image myregistry/api-go:latest

# Kubernetes манифесты
kube-bench check
kubesec scan deployment.yaml
```

---

## 9. Incident Response

### 9.1. Классификация инцидентов

| Уровень | Описание | Время реакции | Эскалация |
|---------|----------|---------------|-----------|
| P1 (Critical) | Полная недоступность, утечка данных | 15 мин | Немедленно |
| P2 (High) | Частичная недоступность, деградация | 1 час | В течение 4 часов |
| P3 (Medium) | Отдельные функции не работают | 4 часа | В течение 24 часов |
| P4 (Low) | Косметические проблемы | 24 часа | По плану |

### 9.2. Playbook: Компрометация токена

```
1. DETECT
   - Алерт от системы мониторинга
   - Жалоба пользователя
   - Аномальная активность в логах

2. CONTAIN
   - Invalidate сессию в Redis
   - Заблокировать пользователя в Keycloak
   - Отозвать refresh tokens

3. ERADICATE
   - Расследовать источник компрометации
   - Проверить логи на несанкционированный доступ
   - Проверить другие сессии пользователя

4. RECOVER
   - Сбросить пароль пользователя
   - Выпустить новые токены
   - Уведомить пользователя

5. LESSONS LEARNED
   - Документировать инцидент
   - Обновить процедуры безопасности
   - Провести post-mortem
```

---

## 10. Compliance

### 10.1. Соответствие требованиям

| Требование | Статус | Описание |
|------------|--------|----------|
| GDPR | ✅ | Право на удаление, доступ к данным |
| 152-ФЗ | ✅ | Хранение ПДн в РФ, уведомление Роскомнадзора |
| ISO 27001 | 🟡 | Частичное соответствие (требует аудита) |
| SOC 2 | ⚪ | Не сертифицировано |

### 10.2. Data Retention

| Данные | Срок хранения | Удаление |
|--------|---------------|----------|
| Сообщения чата | 3 года | Автоматическое |
| Логи аудита | 5 лет | Архив + удаление |
| Логи приложений | 90 дней | Ротация |
| Записи звонков | 1 год | По политике компании |
| Неактивные аккаунты | 2 года | Деактивация + удаление |

---

## 11. Security Checklist

### 11.1. Pre-deployment

- [ ] Все зависимости обновлены до последних версий
- [ ] SAST scan пройден без критических уязвимостей
- [ ] DAST scan пройден
- [ ] Секреты не захардкожены, используются Secrets/Vault
- [ ] TLS настроен с современными cipher suites
- [ ] Security headers применены
- [ ] Rate limiting включён
- [ ] Логирование настроено (без чувствительных данных)

### 11.2. Post-deployment

- [ ] Firewall правила применены
- [ ] Network Policies настроены
- [ ] Мониторинг безопасности активирован
- [ ] Backup шифруется
- [ ] Доступ к production ограничен
- [ ] MFA включён для администраторов
- [ ] Инцидент-response план готов

### 11.3. Periodic

- [ ] Ежеквартальный security audit
- [ ] Ежемесячное обновление зависимостей
- [ ] Еженедельный review логов безопасности
- [ ] Ежедневная проверка алертов

---

## 12. Приложения

### 12.1. Ссылки

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [Keycloak Security Guide](https://www.keycloak.org/docs/latest/server_admin/#_security-guide)

### 12.2. Контакты security team

| Роль | Контакт |
|------|---------|
| Security Officer | security@company.com |
| Incident Response | incident@company.com |
| DPO (Data Protection) | dpo@company.com |
