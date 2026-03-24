# Network Topology

**Версия:** 1.0  
**Дата:** 24 марта 2026 г.  
**Статус:** Черновик

---

## 1. Общая схема сетевой топологии

```
                                    INTERNET
                                        │
                                        ▼
                            ┌───────────────────────┐
                            │   Corporate Firewall  │
                            │   (Palo Alto / pfSense) │
                            └───────────┬───────────┘
                                        │
                                        ▼
                            ┌───────────────────────┐
                            │      DMZ Zone         │
                            │  ┌─────────────────┐  │
                            │  │  Load Balancer  │  │
                            │  │  (F5 / HAProxy) │  │
                            │  └────────┬────────┘  │
                            └───────────┼───────────┘
                                        │
            ┌───────────────────────────┼───────────────────────────┐
            │                           │                           │
            ▼                           ▼                           ▼
    ┌───────────────┐           ┌───────────────┐           ┌───────────────┐
    │  Public Zone  │           │ App Zone      │           │  Data Zone    │
    │               │           │               │           │               │
    │ ┌───────────┐ │           │ ┌───────────┐ │           │ ┌───────────┐ │
    │ │ Ingress   │ │           │ │ K8s Nodes │ │           │ │ PostgreSQL│ │
    │ │ Controller│ │           │ │ (Workers) │ │           │ │ Cluster   │ │
    │ └───────────┘ │           │ └───────────┘ │           │ └───────────┘ │
    │               │           │               │           │               │
    │ ┌───────────┐ │           │ ┌───────────┐ │           │ ┌───────────┐ │
    │ │ Keycloak  │ │           │ │  Pods:    │ │           │ │  Redis    │ │
    │ │ (SSO)     │ │           │ │  - API    │ │           │ │  Cluster  │ │
    │ └───────────┘ │           │ │  - FE     │ │           │ └───────────┘ │
    │               │           │ │  - WS     │ │           │               │
    └───────────────┘           │ └───────────┘ │           │ ┌───────────┐ │
                                │               │           │ │   MinIO   │ │
                                │ ┌───────────┐ │           │ │  Storage  │ │
                                │ │  Jitsi    │ │           │ └───────────┘ │
                                │ │  Stack    │ │           │               │
                                │ └───────────┘ │           └───────────────┘
                                └───────────────┘
```

---

## 2. Сегментация сети

### 2.1. VLAN

| VLAN ID | Название | Назначение | Подсеть |
|---------|----------|------------|---------|
| 10 | MGMT | Управление инфраструктурой | 10.0.10.0/24 |
| 20 | DMZ | Публичные сервисы | 10.0.20.0/24 |
| 30 | APP | Приложение (K8s Nodes) | 10.0.30.0/24 |
| 40 | DATA | Базы данных и хранилища | 10.0.40.0/24 |
| 50 | JITSI | Jitsi компоненты | 10.0.50.0/24 |
| 100 | POD | Pod network (Calico/Flannel) | 10.244.0.0/16 |
| 101 | SVC | Service network | 10.96.0.0/12 |

### 2.2. Zone Security Levels

| Zone | Security Level | Описание |
|------|---------------|----------|
| Internet | 0 (Untrusted) | Внешний мир |
| DMZ | 50 (Semi-Trusted) | Публичные сервисы |
| APP | 75 (Trusted) | Приложение |
| DATA | 100 (Highly Trusted) | Базы данных |

---

## 3. Таблица портов и протоколов

### 3.1. Внешний доступ (из Internet)

| Сервис | Порт | Протокол | Источник | Назначение | Описание |
|--------|------|----------|----------|------------|----------|
| HTTPS (chat) | 443 | TCP/HTTPS | Any | Ingress | Мессенджер |
| HTTPS (api) | 443 | TCP/HTTPS | Any | Ingress | REST API |
| HTTPS (meet) | 443 | TCP/HTTPS | Any | Ingress | Jitsi Meet |
| Jitsi Video | 10000 | UDP | Any | JVB | RTP видео |
| Jitsi TCP | 4443 | TCP/TLS | Any | JVB | Fallback для клиентов |
| XMPP BOSH | 5280 | TCP/HTTP | Any | Prosody | HTTP-bind для клиентов |

### 3.2. Внутренняя коммуникация (Kubernetes)

| Компонент | Порт | Протокол | Источник | Назначение | Описание |
|-----------|------|----------|----------|------------|----------|
| Kubernetes API | 6443 | TCP/HTTPS | Admin | Master | Управление кластером |
| etcd | 2379-2380 | TCP | Master | Master | Хранилище состояний |
| Kubelet | 10250 | TCP/HTTPS | Master | Node | Управление нодами |
| NodePort range | 30000-32767 | TCP/UDP | Ingress | Pods | Сервисы NodePort |
| Pod network | - | IPIP/VXLAN | Pods | Pods | CNI (Calico/Flannel) |

### 3.3. Базы данных и хранилища

| Сервис | Порт | Протокол | Источник | Назначение | Описание |
|--------|------|----------|----------|------------|----------|
| PostgreSQL | 5432 | TCP | API Pods | PostgreSQL | SQL запросы |
| Redis | 6379 | TCP | API Pods | Redis | Кэш, сессии, pub/sub |
| MinIO API | 9000 | TCP/HTTP | API Pods | MinIO | S3-совместимое API |
| MinIO Console | 9001 | TCP/HTTP | Admin | MinIO | Веб-консоль |

### 3.4. Jitsi внутренняя коммуникация

| Компонент | Порт | Протокол | Источник | Назначение | Описание |
|-----------|------|----------|----------|------------|----------|
| Prosody C2S | 5222 | TCP/XMPP | Jitsi Web | Prosody | Client-to-Server |
| Prosody S2S | 5269 | TCP/XMPP | Prosody | Prosody | Server-to-Server |
| Prosody Component | 5347 | TCP/XMPP | Jicofo | Prosody | Компоненты |
| Jicofo | 8787 | TCP/Colibri | JVB | Jicofo | Управление конференциями |
| JVB WebSocket | 8788 | TCP/WS | Jitsi Web | JVB | WebSocket для видео |

### 3.5. Интеграции

| Сервис | Порт | Протокол | Источник | Назначение | Описание |
|--------|------|----------|----------|------------|----------|
| Keycloak | 443 | TCP/HTTPS | API, FE | Keycloak | OIDC аутентификация |
| MS Graph API | 443 | TCP/HTTPS | API | graph.microsoft.com | Календари Exchange |
| Azure AD | 443 | TCP/HTTPS | API | login.microsoftonline.com | OAuth токены |

---

## 4. Firewall правила

### 4.1. Входной трафик (Ingress)

```
# Разрешить HTTPS из Internet
ALLOW TCP/443 FROM Any TO Ingress-Controller

# Разрешить Jitsi UDP видео
ALLOW UDP/10000 FROM Any TO JVB-Nodes

# Разрешить Jitsi TCP fallback
ALLOW TCP/4443 FROM Any TO JVB-Nodes

# Разрешить XMPP BOSH
ALLOW TCP/5280 FROM Any TO Prosody

# Запретить всё остальное
DENY ALL FROM Any TO Any
```

### 4.2. Межзонный трафик

```
# DMZ → APP: только необходимые порты
ALLOW TCP/8080 FROM DMZ TO APP (API Pods)
ALLOW TCP/80 FROM DMZ TO APP (FE Pods)
ALLOW TCP/8788 FROM DMZ TO APP (JVB WebSocket)

# APP → DATA: только БД
ALLOW TCP/5432 FROM APP TO DATA (PostgreSQL)
ALLOW TCP/6379 FROM APP TO DATA (Redis)
ALLOW TCP/9000 FROM APP TO DATA (MinIO)

# DATA → APP: запретить (stateful return только)
DENY ALL FROM DATA TO APP

# APP → Internet: только внешние API
ALLOW TCP/443 FROM APP TO graph.microsoft.com
ALLOW TCP/443 FROM APP TO login.microsoftonline.com
```

### 4.3. Network Policies (Kubernetes)

```yaml
# Разрешить трафик только внутри namespace messenger
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
  namespace: messenger
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
---
# Разрешить ingress от Ingress Controller
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-ingress
  namespace: messenger
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
---
# Разрешить egress только к БД
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-to-database
  namespace: messenger
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: infra
    ports:
    - protocol: TCP
      port: 5432
    - protocol: TCP
      port: 6379
---
# Разрешить egress к внешним API
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-external-api
  namespace: messenger
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - 10.0.0.0/8
        - 172.16.0.0/12
        - 192.168.0.0/16
    ports:
    - protocol: TCP
      port: 443
```

---

## 5. TLS/SSL стратегия

### 5.1. Сертификаты

| Домен | Тип | Issuer | Срок действия |
|-------|-----|--------|---------------|
| chat.company.com | DV | Let's Encrypt | 90 дней (auto-renew) |
| api.company.com | DV | Let's Encrypt | 90 дней (auto-renew) |
| meet.company.com | DV | Let's Encrypt | 90 дней (auto-renew) |
| keycloak.company.com | DV | Let's Encrypt | 90 дней (auto-renew) |
| *.company.com | Wildcard | Corporate CA | 1 год |

### 5.2. TLS конфигурация (Nginx Ingress)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: messenger-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
      add_header X-Content-Type-Options "nosniff" always;
      add_header X-Frame-Options "SAMEORIGIN" always;
      add_header X-XSS-Protection "1; mode=block" always;
      add_header Referrer-Policy "strict-origin-when-cross-origin" always;
      ssl_protocols TLSv1.2 TLSv1.3;
      ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
      ssl_prefer_server_ciphers on;
      ssl_session_cache shared:SSL:10m;
      ssl_session_timeout 10m;
spec:
  # ...
```

### 5.3. mTLS для сервисов (опционально)

**Инструмент:** Istio Service Mesh

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: messenger
spec:
  mtls:
    mode: STRICT
```

---

## 6. Балансировка нагрузки

### 6.1. Load Balancer (L4)

```
┌─────────────────────────────────────────┐
│         Load Balancer (F5/HAProxy)      │
├─────────────────────────────────────────┤
│  Frontend: VIP 10.0.20.100:443          │
│  Backend Pool:                          │
│    - k8s-node-1:30443 (weight 1)        │
│    - k8s-node-2:30443 (weight 1)        │
│    - k8s-node-3:30443 (weight 1)        │
│  Health Check: HTTPS GET /health        │
│  Algorithm: Round Robin                 │
│  Session Persistence: Source IP         │
└─────────────────────────────────────────┘
```

### 6.2. Ingress Controller (L7)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-ingress-controller
  namespace: ingress-nginx
spec:
  replicas: 2
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                - ingress-nginx
            topologyKey: kubernetes.io/hostname
      containers:
      - name: controller
        image: registry.k8s.io/ingress-nginx/controller:v1.9.0
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

---

## 7. DNS конфигурация

### 7.1. Public DNS Records

| Запись | Тип | Значение | TTL |
|--------|-----|----------|-----|
| chat.company.com | A | 203.0.113.100 (LB VIP) | 300 |
| api.company.com | A | 203.0.113.100 (LB VIP) | 300 |
| meet.company.com | A | 203.0.113.100 (LB VIP) | 300 |
| keycloak.company.com | A | 203.0.113.100 (LB VIP) | 300 |
| _acme-challenge.company.com | TXT | ACME validation | 300 |

### 7.2. Internal DNS Records

| Запись | Тип | Значение | TTL |
|--------|-----|----------|-----|
| postgresql.infra.svc.cluster.local | A | 10.0.40.10 | 60 |
| redis.infra.svc.cluster.local | A | 10.0.40.20 | 60 |
| keycloak.infra.svc.cluster.local | A | 10.0.30.100 | 60 |
| jitsi-meet.jitsi.svc.cluster.local | A | 10.0.50.10 | 60 |

---

## 8. QoS и Traffic Shaping

### 8.1. Приоритеты трафика

| Тип трафика | DSCP | Приоритет | Описание |
|-------------|------|-----------|----------|
| Jitsi Video (RTP) | EF (46) | Highest | Видео конференции |
| Jitsi Signaling | AF41 (34) | High | XMPP сигнализация |
| API REST | AF21 (18) | Medium | Бизнес-логика |
| WebSocket Chat | AF21 (18) | Medium | Чат сообщения |
| Background | BE (0) | Low | Фоновые задачи |

### 8.2. Rate Limiting (Nginx)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: messenger-ingress
  annotations:
    nginx.ingress.kubernetes.io/limit-rps: "100"
    nginx.ingress.kubernetes.io/limit-connections: "10"
    nginx.ingress.kubernetes.io/limit-burst-multiplier: "3"
    nginx.ingress.kubernetes.io/limit-rate-after: "10m"
    nginx.ingress.kubernetes.io/limit-rate: "5m"
```

---

## 9. Мониторинг сети

### 9.1. Метрики для сбора

| Метрика | Источник | Частота | Порог алерта |
|---------|----------|---------|--------------|
| Network throughput | Node/Pod | 15s | >80% bandwidth |
| Packet loss | Node | 15s | >0.1% |
| Latency (p99) | Pod-to-Pod | 15s | >100ms |
| TCP retransmits | Node | 15s | >1% |
| Connection errors | Service | 15s | >10/min |
| SSL handshake errors | Ingress | 15s | >5/min |

### 9.2. Инструменты

- **Prometheus** + **Node Exporter** — метрики сети
- **Grafana** — дашборды
- **Alertmanager** — алерты
- **Flowspec** (опционально) — NetFlow/sFlow анализ

---

## 10. Security Groups (Cloud)

### 10.1. AWS Security Groups

```json
{
  "GroupName": "k8s-worker-sg",
  "IpPermissions": [
    {
      "IpProtocol": "tcp",
      "FromPort": 443,
      "ToPort": 443,
      "IpRanges": [{"CidrIp": "0.0.0.0/0"}]
    },
    {
      "IpProtocol": "udp",
      "FromPort": 10000,
      "ToPort": 10000,
      "IpRanges": [{"CidrIp": "0.0.0.0/0"}]
    },
    {
      "IpProtocol": "tcp",
      "FromPort": 10250,
      "ToPort": 10250,
      "IpRanges": [{"CidrIp": "10.0.10.0/24"}]
    },
    {
      "IpProtocol": "tcp",
      "FromPort": 5432,
      "ToPort": 5432,
      "IpRanges": [{"CidrIp": "10.0.30.0/24"}]
    }
  ]
}
```

---

## 11. Disaster Recovery: Network

### 11.1. Failover сценарий

```
Primary Site (Region A)          Secondary Site (Region B)
        │                               │
        │◄─────── DNS Failover ────────►│
        │         (Route53 Health)      │
        │                               │
   [Active]                       [Standby]
        │                               │
        └───────────◄───────────────────┘
                    │
              Global Load Balancer
              (Route53 / CloudFlare)
```

### 11.2. DNS Failover конфигурация

```yaml
# Route53 Health Check
apiVersion: route53.aws/v1
kind: HealthCheck
metadata:
  name: primary-site-health
spec:
  fqdn: api.company.com
  port: 443
  type: HTTPS
  resourcePath: /health
  failureThreshold: 3
  requestInterval: 30

# Route53 Failover Record
apiVersion: route53.aws/v1
kind: RecordSet
metadata:
  name: api-company-com
spec:
  name: api.company.com
  type: A
  failover: PRIMARY
  healthCheckId: primary-site-health
  setIdentifier: primary
---
apiVersion: route53.aws/v1
kind: RecordSet
metadata:
  name: api-company-com-secondary
spec:
  name: api.company.com
  type: A
  failover: SECONDARY
  setIdentifier: secondary
```

---

## 12. Приложения

### 12.1. Чеклист настройки сети

- [ ] VLAN настроены на коммутаторах
- [ ] Firewall правила применены
- [ ] Load Balancer сконфигурирован
- [ ] DNS записи созданы
- [ ] TLS сертификаты получены
- [ ] Network Policies применены
- [ ] QoS политики настроены
- [ ] Мониторинг сети активирован
- [ ] Failover протестирован

### 12.2. Ссылки

- [Infrastructure.md](./Infrastructure.md)
- [Security.md](./Security.md)
- [Architecture.md](./Architecture.md)
