# Data Flow Diagrams (DFD)

**Версия:** 1.0  
**Дата:** 24 марта 2026 г.  
**Статус:** Черновик

---

## 1. DFD Level 0 (Context Diagram)

```mermaid
graph TB
    subgraph "External Entities"
        User[("Пользователь")]
        Admin[("Администратор")]
        Keycloak[("Keycloak SSO")]
        Exchange[("MS Exchange")]
    end

    System[("Система Корпоративный Мессенджер")]

    User -->|Вход, сообщения, звонки| System
    Admin -->|Управление, мониторинг| System
    System -->|OIDC аутентификация| Keycloak
    System -->|Синхронизация календарей| Exchange
    System -->|Уведомления| User
    System -->|Отчёты| Admin
```

---

## 2. DFD Level 1: Основные процессы

```mermaid
graph TB
    subgraph "External Entities"
        User[("Пользователь")]
        Keycloak[("Keycloak")]
        Exchange[("MS Exchange")]
    end

    subgraph "Система"
        P1[("1. Аутентификация")]
        P2[("2. Управление комнатами")]
        P3[("3. Обмен сообщениями")]
        P4[("4. Видеоконференции")]
        P5[("5. Календари")]
        P6[("6. Вебхуки и боты")]

        D1[("D1: Пользователи")]
        D2[("D2: Комнаты")]
        D3[("D3: Сообщения")]
        D4[("D4: Сессии")]
        D5[("D5: Встречи")]
    end

    User -->|Credentials| P1
    P1 -->|OIDC запрос| Keycloak
    Keycloak -->|Tokens| P1
    P1 -->|Сессия| D4
    P1 -->|User data| D1

    User -->|Создать комнату| P2
    P2 -->|Сохранить| D2
    P2 -->|JWT| P4

    User -->|Отправить сообщение| P3
    P3 -->|Сохранить| D3
    P3 -->|WebSocket| User

    User -->|Начать звонок| P4
    P4 -->|XMPP| Jitsi[("Jitsi Meet")]

    User -->|Создать встречу| P5
    P5 -->|Graph API| Exchange
    P5 -->|Сохранить| D5

    P2 -->|События| P6
    P5 -->|События| P6
    P6 -->|Уведомления| User
```

---

## 3. DFD Level 2: Аутентификация

```mermaid
graph TB
    subgraph "External"
        User[("Пользователь")]
        Keycloak[("Keycloak")]
    end

    subgraph "Процессы"
        P1_1[("1.1 Инициация входа")]
        P1_2[("1.2 Обработка callback")]
        P1_3[("1.3 Валидация токена")]
        P1_4[("1.4 Создание сессии")]
    end

    subgraph "Data Stores"
        D1[("D1: Пользователи")]
        D4[("D4: Сессии")]
    end

    User -->|Открыть приложение| P1_1
    P1_1 -->|Редирект на Keycloak| User
    User -->|Login/Password| Keycloak
    Keycloak -->|Auth Code| User
    User -->|Callback с кодом| P1_2
    P1_2 -->|Обмен кода на токен| Keycloak
    Keycloak -->|Access + ID Token| P1_2
    P1_2 -->|Извлечь user info| P1_3
    P1_3 -->|Валидировать JWT| P1_3
    P1_3 -->|Создать/обновить| D1
    P1_3 -->|User data| P1_4
    P1_4 -->|Создать сессию в Redis| D4
    P1_4 -->|Session JWT| User
```

---

## 4. DFD Level 2: Управление комнатами

```mermaid
graph TB
    subgraph "External"
        User[("Пользователь")]
        Jitsi[("Jitsi Prosody")]
    end

    subgraph "Процессы"
        P2_1[("2.1 Создание комнаты")]
        P2_2[("2.2 Получение списка")]
        P2_3[("2.3 Вход в комнату")]
        P2_4[("2.4 Генерация JWT")]
    end

    subgraph "Data Stores"
        D2[("D2: Комнаты")]
        D1[("D1: Пользователи")]
    end

    User -->|POST /rooms| P2_1
    P2_1 -->|Проверить сессию| D1
    P2_1 -->|Сгенерировать имя| P2_1
    P2_1 -->|Сохранить комнату| D2
    P2_1 -->|Room data| P2_4
    P2_4 -->|Создать Jitsi JWT| Jitsi
    Jitsi -->|Подтверждение| P2_4
    P2_4 -->|JWT + URL| User

    User -->|GET /rooms| P2_2
    P2_2 -->|Получить список| D2
    D2 -->|Rooms list| P2_2
    P2_2 -->|JSON| User

    User -->|POST /rooms/:id/join| P2_3
    P2_3 -->|Проверить доступ| D2
    P2_3 -->|Запросить JWT| P2_4
    P2_4 -->|JWT| P2_3
    P2_3 -->|JWT + URL| User
```

---

## 5. DFD Level 2: Обмен сообщениями

```mermaid
graph TB
    subgraph "External"
        User[("Пользователь")]
        WS[("WebSocket Clients")]
    end

    subgraph "Процессы"
        P3_1[("3.1 Отправка сообщения")]
        P3_2[("3.2 Сохранение сообщения")]
        P3_3[("3.3 Рассылка через WebSocket")]
        P3_4[("3.4 Получение истории")]
    end

    subgraph "Data Stores"
        D3[("D3: Сообщения")]
        D2[("D2: Комнаты")]
        D1[("D1: Пользователи")]
        Cache[("Redis Pub/Sub")]
    end

    User -->|POST /messages| P3_1
    P3_1 -->|Валидировать комнату| D2
    P3_1 -->|Валидировать пользователя| D1
    P3_1 -->|Сохранить| P3_2
    P3_2 -->|INSERT message| D3
    P3_2 -->|Publish event| Cache
    Cache -->|Subscribe| P3_3
    P3_3 -->|WebSocket message| WS
    WS -->|Обновить UI| User

    User -->|GET /messages?room_id| P3_4
    P3_4 -->|SELECT WHERE room_id| D3
    D3 -->|Messages list| P3_4
    P3_4 -->|JSON| User
```

---

## 6. DFD Level 2: Видеоконференции

```mermaid
graph TB
    subgraph "External"
        User[("Пользователь")]
        JitsiMeet[("Jitsi Meet Web")]
        JVB[("Jitsi Videobridge")]
    end

    subgraph "Процессы"
        P4_1[("4.1 Инициация звонка")]
        P4_2[("4.2 Генерация JWT")]
        P4_3[("4.3 Подключение к комнате")]
        P4_4[("4.4 Мониторинг звонка")]
    end

    subgraph "Data Stores"
        D2[("D2: Комнаты")]
        Events[("D6: События звонков")]
    end

    User -->|Нажать "Начать звонок"| P4_1
    P4_1 -->|Получить комнату| D2
    P4_1 -->|Запросить JWT| P4_2
    P4_2 -->|Создать JWT claims| P4_2
    P4_2 -->|Подписать секретом| P4_2
    P4_2 -->|JWT| P4_1
    P4_1 -->|Открыть iframe с JWT| JitsiMeet
    JitsiMeet -->|XMPP authenticate| Prosody[("Prosody")]
    Prosody -->|Валидировать JWT| Prosody
    Prosody -->|Разрешить вход| JitsiMeet
    JitsiMeet -->|WebRTC stream| JVB
    JVB -->|Video to participants| User
    P4_4 -->|Получать события| Prosody
    P4_4 -->|Сохранить| Events
```

---

## 7. DFD Level 2: Календари

```mermaid
graph TB
    subgraph "External"
        User[("Пользователь")]
        GraphAPI[("MS Graph API")]
        Exchange[("Exchange Server")]
        Attendees[("Участники встречи")]
    end

    subgraph "Процессы"
        P5_1[("5.1 Получение событий")]
        P5_2[("5.2 Создание встречи")]
        P5_3[("5.3 Генерация Jitsi комнаты")]
        P5_4[("5.4 Отправка приглашений")]
        P5_5[("5.5 Синхронизация изменений")]
    end

    subgraph "Data Stores"
        D5[("D5: Встречи")]
        D2[("D2: Комнаты")]
        Cache[("Redis Cache")]
    end

    User -->|GET /calendar/events| P5_1
    P5_1 -->|Получить токен Graph| Cache
    Cache -->|Token| P5_1
    P5_1 -->|GET /events| GraphAPI
    GraphAPI -->|Events list| P5_1
    P5_1 -->|JSON| User

    User -->|POST /calendar/events| P5_2
    P5_2 -->|Создать комнату| P5_3
    P5_3 -->|Сохранить| D2
    P5_3 -->|Jitsi URL| P5_2
    P5_2 -->|POST /events с Jitsi URL| GraphAPI
    GraphAPI -->|Создать событие| Exchange
    Exchange -->|Event ID| P5_2
    P5_2 -->|Отправить приглашения| P5_4
    P5_4 -->|Email| Attendees
    P5_2 -->|Сохранить связь| D5
    P5_2 -->|Event data| User

    GraphAPI -->|Webhook изменение| P5_5
    P5_5 -->|Обновить кэш| Cache
    P5_5 -->|Обновить| D5
```

---

## 8. DFD Level 2: Вебхуки и боты

```mermaid
graph TB
    subgraph "External"
        Jitsi[("Jitsi Prosody")]
        User[("Пользователь")]
        ExternalSystem[("Внешняя система")]
    end

    subgraph "Процессы"
        P6_1[("6.1 Приём webhook от Jitsi")]
        P6_2[("6.2 Обработка события")]
        P6_3[("6.3 Рассылка уведомлений")]
        P6_4[("6.4 Парсинг команд бота")]
        P6_5[("6.5 Выполнение команды")]
    end

    subgraph "Data Stores"
        Events[("D6: События звонков")]
        D7[("D7: Вебхуки")]
        D8[("D8: Боты")]
        D3[("D3: Сообщения")]
    end

    Jitsi -->|POST /webhooks/jitsi| P6_1
    P6_1 -->|Валидировать подпись| P6_1
    P6_1 -->|Парсить payload| P6_2
    P6_2 -->|Определить тип события| P6_2
    P6_2 -->|Сохранить событие| Events
    P6_2 -->|Найти подписки| D7
    D7 -->|Webhook URLs| P6_3
    P6_3 -->|POST на каждый URL| ExternalSystem
    P6_3 -->|Системное сообщение| D3

    User -->|/command в чате| P6_4
    P6_4 -->|Распознать команду| P6_4
    P6_4 -->|Найти бота| D8
    D8 -->|Bot handler| P6_5
    P6_5 -->|Выполнить действие| P6_5
    P6_5 -->|Ответ в чат| D3
    P6_5 -->|Создать встречу| P5_2
```

---

## 9. Схема аутентификации JWT

```mermaid
sequenceDiagram
    participant U as Пользователь
    participant FE as Frontend
    participant API as Go API
    participant KC as Keycloak
    participant J as Jitsi Prosody

    Note over U,J: Аутентификация пользователя
    U->>FE: Открыть приложение
    FE->>U: Редирект на Keycloak
    U->>KC: Ввод credentials
    KC->>FE: Authorization Code
    FE->>API: POST /auth/callback (code)
    API->>KC: Обмен кода на токен
    KC->>API: Access + ID Token
    API->>API: Валидировать ID Token
    API->>API: Создать session JWT
    API->>FE: Session JWT
    FE->>FE: Сохранить JWT

    Note over U,J: Генерация Jitsi JWT
    U->>FE: Создать комнату
    FE->>API: POST /rooms
    API->>API: Создать комнату в БД
    API->>API: Сгенерировать Jitsi JWT
    Note right of API: Claims:<br/>- room<br/>- user.id<br/>- user.name<br/>- moderator<br/>- exp
    API->>FE: Room + Jitsi JWT
    FE->>FE: Открыть Jitsi iframe
    FE->>J: Подключение с JWT
    J->>J: Валидировать JWT
    J->>FE: Разрешить вход
```

---

## 10. Поток данных: Создание встречи

```mermaid
sequenceDiagram
    participant U as Пользователь
    participant FE as Frontend
    participant API as Go API
    participant DB as PostgreSQL
    participant Graph as MS Graph
    participant Exch as Exchange
    participant Att as Участники

    U->>FE: Заполнить форму встречи
    FE->>API: POST /calendar/events
    API->>API: Валидировать запрос
    API->>API: Сгенерировать room name
    API->>DB: INSERT rooms (jitsi_room)
    DB->>API: room_id
    API->>API: Создать Jitsi JWT
    API->>Graph: POST /users/:id/calendar/events
    Note right of Graph: Body:<br/>- subject<br/>- start/end<br/>- attendees<br/>- location: Jitsi URL
    Graph->>Exch: Создать событие
    Exch->>Graph: Event ID
    Graph->>Att: Отправить приглашения (email)
    Graph->>API: Event data
    API->>DB: INSERT meetings (exchange_event_id)
    API->>FE: { event_id, jitsi_url }
    FE->>U: Показать встречу в календаре
```

---

## 11. Поток данных: Real-time чат

```mermaid
sequenceDiagram
    participant U1 as Пользователь 1
    participant FE1 as Frontend 1
    participant WS as WebSocket Server
    participant API as Go API
    participant DB as PostgreSQL
    participant Redis as Redis Pub/Sub
    participant FE2 as Frontend 2
    participant U2 as Пользователь 2

    U1->>FE1: Ввести сообщение
    FE1->>API: POST /rooms/:id/messages
    API->>DB: INSERT messages
    DB->>API: message_id
    API->>Redis: PUBLISH room:{id} message
    API->>FE1: { message_id, status: sent }
    FE1->>U1: Показать сообщение (оптимистично)

    Redis->>WS: SUBSCRIBE room:{id}
    WS->>WS: Обработать событие
    WS->>FE2: WebSocket: new_message
    FE2->>U2: Показать сообщение

    Note over U1,U2: Typing indicator
    U1->>FE1: Начать ввод
    FE1->>WS: { type: typing, is_typing: true }
    WS->>Redis: PUBLISH typing:{room_id}
    Redis->>WS: SUBSCRIBE typing:{room_id}
    WS->>FE2: WebSocket: typing_status
    FE2->>U2: Показать "печатает..."
```

---

## 12. Таблица потоков данных

| ID | Источник | Назначение | Данные | Протокол | Частота |
|----|----------|------------|--------|----------|---------|
| F1 | Пользователь | Frontend | Credentials | HTTPS | По запросу |
| F2 | Frontend | Keycloak | Auth request | HTTPS (OIDC) | По запросу |
| F3 | Keycloak | Frontend | Auth code | HTTPS | По запросу |
| F4 | Frontend | Go API | Code exchange | HTTPS | По запросу |
| F5 | Go API | Keycloak | Token request | HTTPS | По запросу |
| F6 | Keycloak | Go API | Access/ID tokens | HTTPS | По запросу |
| F7 | Go API | PostgreSQL | User data | SQL (5432) | По запросу |
| F8 | Go API | Redis | Session data | Redis (6379) | По запросу |
| F9 | Frontend | Go API | REST API calls | HTTPS | По запросу |
| F10 | Frontend | WebSocket | Real-time messages | WSS | Постоянно |
| F11 | Go API | PostgreSQL | Messages CRUD | SQL | По запросу |
| F12 | Go API | Redis | Pub/Sub events | Redis | Real-time |
| F13 | WebSocket | Frontend | Push notifications | WSS | Real-time |
| F14 | Frontend | Jitsi iframe | Video conference | HTTPS | По запросу |
| F15 | Go API | Jitsi Prosody | JWT generation | JWT | По запросу |
| F16 | Jitsi Prosody | Go API | Webhook events | HTTPS | По событию |
| F17 | Go API | MS Graph | Calendar API | HTTPS | По запросу |
| F18 | MS Graph | Go API | Events data | HTTPS | По запросу |
| F19 | Go API | External | Outgoing webhooks | HTTPS | По событию |

---

## 13. Диаграмма состояний: Комната

```mermaid
stateDiagram-v2
    [*] --> Created: Создана
    Created --> Active: Первый участник
    Active --> InCall: Начался звонок
    InCall --> Active: Звонок завершён
    Active --> InCall: Новый звонок
    Active --> Archived: Нет активности (30 дней)
    InCall --> Archived: Нет активности
    Archived --> Deleted: Удалена админом
    Created --> Deleted: Удалена сразу
    Deleted --> [*]
```

---

## 14. Диаграмма состояний: Встреча

```mermaid
stateDiagram-v2
    [*] --> Draft: Черновик
    Draft --> Scheduled: Запланирована
    Scheduled --> Reminded: Напоминание (за 15 мин)
    Reminded --> InProgress: Началась (по времени)
    InProgress --> Completed: Завершена
    Scheduled --> Cancelled: Отменена
    Scheduled --> Rescheduled: Перенесена
    Rescheduled --> Scheduled: Новое время
    Completed --> [*]
    Cancelled --> [*]
```

---

## 15. Приложения

### 15.1. Глоссарий DFD

| Термин | Определение |
|--------|-------------|
| External Entity | Внешняя система или пользователь |
| Process | Обработка данных внутри системы |
| Data Store | Хранилище данных (БД, кэш) |
| Data Flow | Поток данных между компонентами |

### 15.2. Ссылки

- [Architecture.md](./Architecture.md)
- [HLD.md](./HLD.md)
- [LLD.md](./LLD.md)
