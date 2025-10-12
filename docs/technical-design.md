# Security Token Service (STS) - Technical Design Document

**Version**: 1.0  
**Date**: 2025-10-09  
**Status**: Draft  
**Author**: [Your Name/Team]

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Overview](#system-overview)
3. [Architecture](#architecture)
4. [Component Design](#component-design)
5. [API Specification](#api-specification)
6. [Data Models](#data-models)
7. [Security Design](#security-design)
8. [Key Management](#key-management)
9. [Error Handling](#error-handling)
10. [Observability](#observability)
11. [Performance & Scalability](#performance--scalability)
12. [Configuration](#configuration)
13. [Testing Strategy](#testing-strategy)
14. [Open Questions](#open-questions)
15. [Appendices](#appendices)

---

## 1. Executive Summary

### 1.1 Purpose
The Security Token Service (STS) is a critical authentication middleware that validates JWTs from trusted Identity Providers (IdPs), enriches user claims with role information from an Entitlement System, and issues new signed JWTs for downstream service consumption.

### 1.2 Goals
- Provide secure, reliable token exchange and enrichment
- Achieve <100ms p95 latency for token operations
- Support horizontal scaling to handle [X] requests/second
- Maintain 99.9% availability
- Enable zero-downtime key rotation

### 1.3 Non-Goals
- User authentication (delegated to IdP)
- Authorization decisions (delegated to downstream services)
- User management or provisioning
- Token storage or session management

---

## 2. System Overview

### 2.1 High-Level Flow

```
┌─────────┐         ┌─────────┐         ┌──────────────┐         ┌─────────────┐
│ Client  │────────▶│   STS   │────────▶│ Entitlement  │         │    Vault    │
│         │  JWT    │         │  Query  │    System    │         │             │
└─────────┘         └─────────┘         └──────────────┘         └─────────────┘
                         │                      │                        │
                         │                      │                        │
                         ▼                      ▼                        ▼
                    ┌─────────┐         ┌──────────────┐         ┌─────────────┐
                    │   IdP   │         │  Get Roles   │         │  Get Keys   │
                    │ (JWKS)  │         │              │         │             │
                    └─────────┘         └──────────────┘         └─────────────┘
                         │                      │                        │
                         └──────────────────────┴────────────────────────┘
                                                │
                                                ▼
                                         ┌─────────────┐
                                         │  New JWT    │
                                         │  (enriched) │
                                         └─────────────┘
```

### 2.2 Request Flow

1. **Token Reception**: Client sends JWT in Authorization header
2. **Token Validation**: 
   - Parse JWT and extract header
   - Retrieve IdP public key using `kid` from JWKS endpoint
   - Validate signature, expiry, issuer, audience
3. **User Enrichment**:
   - Extract UPN from validated token
   - Query Entitlement System API for user roles
   - Handle caching and error scenarios
4. **Token Issuance**:
   - Create new JWT with enriched claims
   - Retrieve signing key from Vault
   - Sign token with current active key
   - Return new token to client

### 2.3 Dependencies

| Dependency | Purpose | SLA Requirement | Failure Impact |
|------------|---------|-----------------|----------------|
| Identity Provider (IdP) | JWT validation (JWKS endpoint) | 99.9% | Cannot validate tokens |
| Entitlement System | Role enrichment | 99.5% | Cannot enrich tokens |
| Vault | Key retrieval | 99.9% | Cannot sign tokens |

---

## 3. Architecture

### 3.1 System Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        STS Service                             │
│                                                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   HTTP       │  │  Middleware  │  │   Handlers   │        │
│  │   Server     │──│  - Auth      │──│  - Exchange  │        │
│  │              │  │  - Logging   │  │  - Health    │        │
│  │              │  │  - Metrics   │  │  - JWKS      │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│                                              │                 │
│  ┌──────────────────────────────────────────┴────────────┐   │
│  │                  Core Services                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │   │
│  │  │   Token      │  │  Enrichment  │  │    Key      │ │   │
│  │  │  Validator   │  │   Service    │  │  Manager    │ │   │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │   │
│  └────────────────────────────────────────────────────────┘   │
│                                              │                 │
│  ┌──────────────────────────────────────────┴────────────┐   │
│  │                  Infrastructure                        │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │   │
│  │  │    Cache     │  │   Circuit    │  │   HTTP      │ │   │
│  │  │   Manager    │  │   Breaker    │  │   Client    │ │   │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                │
└───────────────────────────────────────────────────────────────┘
```

### 3.2 Component Layers

#### 3.2.1 HTTP Layer
- Request routing
- Middleware chain (logging, metrics, recovery, rate limiting)
- Request/response serialization

#### 3.2.2 Service Layer
- Business logic orchestration
- Token validation and issuance
- User enrichment coordination

#### 3.2.3 Infrastructure Layer
- External API clients
- Caching
- Circuit breakers
- Retry logic

### 3.3 Technology Stack

| Component | Technology | Justification |
|-----------|------------|---------------|
| Language | Go 1.21+ | Performance, concurrency, strong typing |
| HTTP Framework | `gin` or `echo` | Middleware support, performance |
| JWT Library | `lestrrat-go/jwx` | Full JWKS support, active maintenance |
| Vault Client | `hashicorp/vault` | Official client, well-tested |
| Cache | In-memory (sync.Map) + Redis (optional) | Low latency, simple deployment |
| Metrics | Prometheus | Industry standard, rich ecosystem |
| Logging | `zap` | Structured, high performance |
| Config | `viper` | Hot-reload support, multiple formats |
| Circuit Breaker | `sony/gobreaker` | Battle-tested, simple API |

---

## 4. Component Design

### 4.1 Token Validator

**Responsibility**: Validate incoming JWTs from IdP

**Interface**:
```go
type TokenValidator interface {
    // Validate validates the JWT and returns parsed claims
    Validate(ctx context.Context, token string) (*Claims, error)
    
    // RefreshKeys forces a refresh of the IdP's public keys
    RefreshKeys(ctx context.Context) error
}
```

**Key Features**:
- JWKS endpoint caching with TTL
- Algorithm whitelist enforcement (RS256, ES256)
- Clock skew tolerance (configurable, default 60s)
- Issuer and audience validation
- Automatic key rotation detection

**Configuration**:
```yaml
validator:
  issuer: "https://idp.example.com"
  audience: "sts-service"
  jwks_url: "https://idp.example.com/.well-known/jwks.json"
  jwks_cache_ttl: 1h
  allowed_algorithms: ["RS256", "ES256"]
  clock_skew_seconds: 60
```

### 4.2 Enrichment Service

**Responsibility**: Fetch user roles from Entitlement System

**Interface**:
```go
type EnrichmentService interface {
    // GetRoles retrieves roles for a given UPN
    GetRoles(ctx context.Context, upn string) ([]string, error)
    
    // InvalidateCache clears cached roles for a user
    InvalidateCache(upn string) error
}
```

**Key Features**:
- HTTP client with connection pooling
- Response caching with configurable TTL
- Circuit breaker integration
- Retry logic with exponential backoff
- Timeout configuration (connection, read, total)

**Configuration**:
```yaml
enrichment:
  api_url: "https://entitlement.example.com/api/v1/roles"
  timeout: 5s
  cache_ttl: 5m
  circuit_breaker:
    max_requests: 3
    interval: 10s
    timeout: 30s
  retry:
    max_attempts: 3
    initial_delay: 100ms
    max_delay: 1s
```

### 4.3 Key Manager

**Responsibility**: Manage signing keys for token issuance

**Interface**:
```go
type KeyManager interface {
    // GetCurrentKey returns the active signing key
    GetCurrentKey(ctx context.Context) (*SigningKey, error)
    
    // GetKey retrieves a specific key by ID
    GetKey(ctx context.Context, kid string) (*SigningKey, error)
    
    // RotateKeys triggers key rotation
    RotateKeys(ctx context.Context) error
    
    // GetJWKS returns public keys in JWKS format
    GetJWKS(ctx context.Context) (*JWKS, error)
}

type SigningKey struct {
    ID         string
    Algorithm  string
    PrivateKey interface{}
    PublicKey  interface{}
    CreatedAt  time.Time
    ExpiresAt  time.Time
    Status     KeyStatus // Active, Deprecated, Revoked
}
```

**Key Features**:
- Vault integration for key storage
- In-memory key caching
- Automatic key rotation
- Grace period for deprecated keys
- JWKS endpoint for public key distribution

**Configuration**:
```yaml
key_manager:
  vault:
    address: "https://vault.example.com"
    path: "secret/sts/keys"
    role: "sts-service"
  rotation:
    frequency: 30d
    grace_period: 7d
  algorithm: "RS256"
  key_size: 4096
```

### 4.4 Cache Manager

**Responsibility**: Unified caching interface

**Interface**:
```go
type CacheManager interface {
    Get(ctx context.Context, key string) (interface{}, bool)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
}
```

**Implementation Options**:
1. **In-Memory**: `sync.Map` with TTL tracking
2. **Redis**: Distributed cache for multi-instance deployments
3. **Hybrid**: In-memory L1 + Redis L2

### 4.5 Circuit Breaker

**Responsibility**: Prevent cascading failures

**Configuration per Dependency**:
```yaml
circuit_breaker:
  idp:
    max_requests: 5
    interval: 10s
    timeout: 60s
  entitlement:
    max_requests: 3
    interval: 10s
    timeout: 30s
  vault:
    max_requests: 5
    interval: 10s
    timeout: 60s
```

---

## 5. API Specification

### 5.1 Token Exchange Endpoint

**Endpoint**: `POST /v1/token/exchange`

**Request**:
```http
POST /v1/token/exchange HTTP/1.1
Host: sts.example.com
Content-Type: application/json
Authorization: Bearer <IdP_JWT>

{
  "requested_token_type": "urn:ietf:params:oauth:token-type:jwt",
  "audience": "downstream-service"
}
```

**Response (Success)**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "access_token": "<new_jwt>",
  "token_type": "Bearer",
  "expires_in": 43200,
  "issued_token_type": "urn:ietf:params:oauth:token-type:jwt"
}
```

**Response (Error)**:
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/problem+json

{
  "type": "https://sts.example.com/errors/invalid-token",
  "title": "Invalid Token",
  "status": 401,
  "detail": "Token signature validation failed",
  "instance": "/v1/token/exchange",
  "trace_id": "abc123"
}
```

### 5.2 Health Check Endpoints

**Liveness**: `GET /health/live`
```json
{
  "status": "ok",
  "timestamp": "2025-10-09T12:00:00Z"
}
```

**Readiness**: `GET /health/ready`
```json
{
  "status": "ready",
  "checks": {
    "vault": "ok",
    "keys_loaded": "ok",
    "idp_reachable": "ok"
  },
  "timestamp": "2025-10-09T12:00:00Z"
}
```

### 5.3 JWKS Endpoint

**Endpoint**: `GET /.well-known/jwks.json`

**Response**:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-2025-10-09",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

### 5.4 Metrics Endpoint

**Endpoint**: `GET /metrics`

Prometheus-formatted metrics

### 5.5 Admin Endpoints

**Key Rotation**: `POST /admin/keys/rotate`
```json
{
  "force": false
}
```

**Cache Invalidation**: `POST /admin/cache/invalidate`
```json
{
  "key": "user:john@example.com"
}
```

---

## 6. Data Models

### 6.1 JWT Claims Structure

**Input JWT (from IdP)**:
```json
{
  "iss": "https://idp.example.com",
  "sub": "user-12345",
  "aud": "sts-service",
  "exp": 1696867200,
  "iat": 1696863600,
  "nbf": 1696863600,
  "upn": "john.doe@example.com",
  "email": "john.doe@example.com",
  "name": "John Doe"
}
```

**Output JWT (from STS)**:
```json
{
  "iss": "https://sts.example.com",
  "sub": "user-12345",
  "aud": "downstream-service",
  "exp": 1696906800,
  "iat": 1696863600,
  "nbf": 1696863600,
  "jti": "token-uuid-12345",
  "upn": "john.doe@example.com",
  "email": "john.doe@example.com",
  "name": "John Doe",
  "roles": ["admin", "user", "developer"],
  "original_issuer": "https://idp.example.com"
}
```

### 6.2 Error Response Model

Following RFC 7807 Problem Details:
```go
type ProblemDetail struct {
    Type     string                 `json:"type"`
    Title    string                 `json:"title"`
    Status   int                    `json:"status"`
    Detail   string                 `json:"detail"`
    Instance string                 `json:"instance"`
    TraceID  string                 `json:"trace_id,omitempty"`
    Extra    map[string]interface{} `json:"-"`
}
```

### 6.3 Configuration Model

```go
type Config struct {
    Server      ServerConfig      `yaml:"server"`
    Validator   ValidatorConfig   `yaml:"validator"`
    Enrichment  EnrichmentConfig  `yaml:"enrichment"`
    KeyManager  KeyManagerConfig  `yaml:"key_manager"`
    Cache       CacheConfig       `yaml:"cache"`
    Observability ObservabilityConfig `yaml:"observability"`
}
```

---

## 7. Security Design

### 7.1 JWT Validation Rules

| Check | Implementation | Failure Action |
|-------|----------------|----------------|
| Signature | Verify with IdP public key | Reject (401) |
| Algorithm | Whitelist check | Reject (401) |
| Expiry (`exp`) | Current time < exp + skew | Reject (401) |
| Not Before (`nbf`) | Current time >= nbf - skew | Reject (401) |
| Issuer (`iss`) | Exact match with config | Reject (401) |
| Audience (`aud`) | Contains STS audience | Reject (401) |
| Token Size | < 8KB | Reject (400) |

### 7.2 Key Management Security

**Key Storage**:
- All private keys stored in Vault
- Keys encrypted at rest
- Access controlled via Vault policies
- Audit logging enabled

**Key Rotation**:
- Automated rotation every 30 days
- 7-day grace period for old keys
- Emergency rotation procedure documented
- Key usage metrics tracked

**Key Access**:
- Service authentication via Vault AppRole
- Token renewal before expiry
- Connection pooling with TLS
- Retry logic for transient failures

### 7.3 Transport Security

- TLS 1.2+ required for all external communication
- Certificate validation enforced
- Mutual TLS (mTLS) support for high-security environments
- HTTP Strict Transport Security (HSTS) headers

### 7.4 Rate Limiting

**Strategy**: Token bucket algorithm

**Limits**:
- Global: 10,000 requests/second
- Per-client: 100 requests/second (based on client_id or IP)
- Burst allowance: 2x sustained rate

**Response**:
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 5
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1696863605
```

### 7.5 Input Validation

- Maximum token size: 8KB
- UPN format: email regex validation
- Audience: alphanumeric + hyphens
- Request body: JSON schema validation
- Headers: whitelist allowed headers

---

## 8. Key Management

### 8.1 Key Lifecycle

```
┌─────────┐     ┌─────────┐     ┌────────────┐     ┌──────────┐
│ Created │────▶│ Active  │────▶│ Deprecated │────▶│ Revoked  │
└─────────┘     └─────────┘     └────────────┘     └──────────┘
                     │                  │
                     │                  │
                     └──────────────────┘
                     (Grace Period: 7d)
```

### 8.2 Key States

| State | Description | Can Sign | Can Verify | In JWKS |
|-------|-------------|----------|------------|---------|
| Created | Just generated | No | No | No |
| Active | Current signing key | Yes | Yes | Yes |
| Deprecated | Grace period | No | Yes | Yes |
| Revoked | Compromised | No | No | No |

### 8.3 Rotation Process

1. **Pre-Rotation** (T-1h):
   - Generate new key in Vault
   - Validate key generation
   - Store key metadata

2. **Activation** (T):
   - Mark new key as Active
   - Mark old key as Deprecated
   - Update JWKS endpoint
   - Emit metrics event

3. **Grace Period** (T to T+7d):
   - New tokens signed with new key
   - Old tokens still validated
   - Monitor old key usage

4. **Cleanup** (T+7d):
   - Mark old key as Revoked
   - Remove from JWKS
   - Archive key metadata

### 8.4 Emergency Rotation

**Trigger**: Key compromise detected

**Process**:
1. Immediately revoke compromised key
2. Generate and activate new key
3. Update JWKS endpoint
4. Notify downstream services
5. Force token refresh for all users

---

## 9. Error Handling

### 9.1 Error Categories

| Category | HTTP Status | Retry | Example |
|----------|-------------|-------|---------|
| Client Error | 400 | No | Malformed request |
| Authentication Error | 401 | No | Invalid token |
| Authorization Error | 403 | No | Insufficient permissions |
| Not Found | 404 | No | Unknown endpoint |
| Rate Limited | 429 | Yes | Too many requests |
| Server Error | 500 | Yes | Internal error |
| Service Unavailable | 503 | Yes | Dependency down |
| Gateway Timeout | 504 | Yes | Dependency timeout |

### 9.2 Error Codes

```go
const (
    ErrCodeInvalidToken       = "INVALID_TOKEN"
    ErrCodeExpiredToken       = "EXPIRED_TOKEN"
    ErrCodeInvalidSignature   = "INVALID_SIGNATURE"
    ErrCodeUnknownUser        = "UNKNOWN_USER"
    ErrCodeEnrichmentFailed   = "ENRICHMENT_FAILED"
    ErrCodeKeyUnavailable     = "KEY_UNAVAILABLE"
    ErrCodeRateLimited        = "RATE_LIMITED"
    ErrCodeInternalError      = "INTERNAL_ERROR"
)
```

### 9.3 Fallback Strategies

**Entitlement Service Down**:
- Option 1: Fail closed (reject token exchange)
- Option 2: Issue token with empty roles (configurable)
- Option 3: Use cached roles if available (with staleness warning)

**Vault Unavailable**:
- Use cached signing key (if available)
- Fail fast if no cached key
- Alert on-call team

**IdP JWKS Unavailable**:
- Use cached public keys
- Reject if key not in cache
- Alert if cache age > threshold

---

## 10. Observability

### 10.1 Logging

**Log Levels**:
- **DEBUG**: Detailed flow information
- **INFO**: Normal operations (token issued, key rotated)
- **WARN**: Degraded state (cache miss, retry)
- **ERROR**: Operation failures
- **FATAL**: Service cannot continue

**Structured Log Format**:
```json
{
  "timestamp": "2025-10-09T12:00:00Z",
  "level": "info",
  "message": "Token exchanged successfully",
  "trace_id": "abc123",
  "span_id": "def456",
  "upn": "john.doe@example.com",
  "duration_ms": 45,
  "kid": "key-2025-10-09",
  "audience": "downstream-service"
}
```

**Sensitive Data Handling**:
- Never log full tokens
- Log only: kid, sub, exp, iss
- Redact PII in non-production environments

### 10.2 Metrics

**Token Operations**:
```
sts_token_exchange_total{status="success|failure",reason=""}
sts_token_exchange_duration_seconds{quantile="0.5|0.95|0.99"}
sts_token_validation_failures_total{reason="expired|invalid_sig|..."}
```

**Cache Metrics**:
```
sts_cache_hits_total{cache_type="jwks|roles"}
sts_cache_misses_total{cache_type="jwks|roles"}
sts_cache_evictions_total{cache_type="jwks|roles"}
```

**Dependency Metrics**:
```
sts_http_request_duration_seconds{service="idp|entitlement|vault"}
sts_http_request_total{service="",status=""}
sts_circuit_breaker_state{service="",state="open|half_open|closed"}
```

**Key Management Metrics**:
```
sts_active_keys_total
sts_key_rotation_total{status="success|failure"}
sts_key_usage_total{kid=""}
```

### 10.3 Distributed Tracing

**Implementation**: OpenTelemetry

**Spans**:
1. `token.exchange` (root)
   - `token.validate`
     - `jwks.fetch`
     - `jwt.verify`
   - `enrichment.get_roles`
     - `http.call`
     - `cache.lookup`
   - `token.sign`
     - `vault.get_key`

**Trace Attributes**:
- `user.upn`
- `token.kid`
- `token.issuer`
- `token.audience`
- `cache.hit`

### 10.4 Alerting

**Critical Alerts**:
- Token validation failure rate > 5%
- Entitlement service circuit breaker open
- Vault connection failures
- Key rotation failures
- Service error rate > 1%

**Warning Alerts**:
- Cache hit rate < 80%
- P95 latency > 200ms
- Deprecated key usage after grace period
- Rate limit threshold reached

---

## 11. Performance & Scalability

### 11.1 Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| P50 Latency | < 50ms | End-to-end token exchange |
| P95 Latency | < 100ms | End-to-end token exchange |
| P99 Latency | < 200ms | End-to-end token exchange |
| Throughput | 10,000 req/s | Per instance |
| Availability | 99.9% | Monthly uptime |
| Error Rate | < 0.1% | Excluding client errors |

### 11.2 Scalability Design

**Horizontal Scaling**:
- Stateless service design
- No local session storage
- Shared cache (Redis) for multi-instance
- Load balancer with health checks

**Resource Limits**:
```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

**Concurrency**:
- Worker pool for token processing
- Connection pooling for HTTP clients
- Goroutine limits to prevent exhaustion

### 11.3 Optimization Strategies

**Caching**:
- JWKS cache: 1 hour TTL
- Roles cache: 5 minutes TTL
- Negative cache: 1 minute TTL

**Connection Pooling**:
```go
http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

**Batch Operations**:
- Consider batching entitlement queries if supported by API
- Aggregate metrics before export

---

## 12. Configuration

### 12.1 Configuration File Structure

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  shutdown_timeout: 30s
  tls:
    enabled: true
    cert_file: "/etc/sts/tls/cert.pem"
    key_file: "/etc/sts/tls/key.pem"

validator:
  issuer: "https://idp.example.com"
  audience: "sts-service"
  jwks_url: "https://idp.example.com/.well-known/jwks.json"
  jwks_cache_ttl: 1h
  allowed_algorithms: ["RS256", "ES256"]
  clock_skew_seconds: 60

enrichment:
  api_url: "https://entitlement.example.com/api/v1/roles"
  timeout: 5s
  cache_ttl: 5m
  circuit_breaker:
    max_requests: 3
    interval: 10s
    timeout: 30s
  retry:
    max_attempts: 3
    initial_delay: 100ms
    max_delay: 1s

key_manager:
  vault:
    address: "https://vault.example.com"
    path: "secret/sts/keys"
    role: "sts-service"
    auth_method: "approle"
  rotation:
    frequency: 720h  # 30 days
    grace_period: 168h  # 7 days
  algorithm: "RS256"
  key_size: 4096

cache:
  type: "memory"  # memory | redis
  redis:
    address: "redis:6379"
    password: ""
    db: 0
    pool_size: 10

observability:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
    sample_rate: 0.1

rate_limiting:
  enabled: true
  global_limit: 10000
  per_client_limit: 100
  burst_multiplier: 2
```

### 12.2 Environment Variables

```bash
# Override config values
STS_SERVER_PORT=8080
STS_VAULT_ADDRESS=https://vault.example.com
STS_VAULT_TOKEN=s.xxxxx
STS_LOG_LEVEL=debug
```

### 12.3 Feature Flags

```yaml
features:
  token_replay_detection: true
  role_caching: true
  distributed_tracing: true
  admin_endpoints: false
```

---

## 13. Testing Strategy

### 13.1 Unit Tests

**Coverage Target**: > 80%

**Focus Areas**:
- JWT validation logic
- Claim enrichment
- Key rotation logic
- Error handling
- Cache operations

**Example**:
```go
func TestTokenValidator_Validate_ExpiredToken(t *testing.T) {
    // Test expired token rejection
}
```

### 13.2 Integration Tests

**Test Scenarios**:
- Full token exchange flow with mocked dependencies
- Circuit breaker behavior
- Cache hit/miss scenarios
- Key rotation during active traffic
- Graceful degradation

**Tools**:
- `httptest` for HTTP testing
- Mock servers for external APIs
- Test containers for Redis

### 13.3 Contract Tests

**Purpose**: Verify API compatibility

**Contracts**:
- IdP JWKS endpoint format
- Entitlement API request/response
- Vault API interactions

**Tools**: Pact or similar

### 13.4 Performance Tests

**Scenarios**:
- Sustained load: 5,000 req/s for 10 minutes
- Spike test: 0 to 10,000 req/s in 10 seconds
- Stress test: Increase until failure
- Soak test: Sustained load for 24 hours

**Tools**: k6, Gatling, or JMeter

### 13.5 Security Tests

**Tests**:
- Algorithm confusion attacks
- Token replay attempts
- Signature manipulation
- Expired token usage
- Invalid issuer/audience
- Oversized tokens

**Tools**: OWASP ZAP, Burp Suite

### 13.6 Chaos Tests

**Scenarios**:
- IdP unavailable
- Entitlement API timeout
- Vault connection loss
- Network latency injection
- Memory pressure
- CPU throttling

**Tools**: Chaos Mesh, Gremlin

---

## 14. Open Questions

### 14.1 Business Logic

1. **Token Revocation**: What's the revocation strategy? Distributed cache? Revocation list? Acceptable latency?

2. **Failure Mode**: When entitlement service is unavailable:
   - Fail closed (reject all requests)?
   - Issue tokens with empty roles?
   - Use cached roles with staleness indicator?

3. **Empty Roles**: Are tokens with zero roles valid? Or should they be rejected?

4. **Token Refresh**: Should we support token refresh before expiry? Or require full re-authentication?

### 14.2 Technical Decisions

5. **Cache Backend**: In-memory only or Redis for multi-instance deployments?

6. **Key Algorithm**: RSA 4096 or ECDSA P-256? Trade-offs?

7. **Multi-Tenancy**: Single IdP or support multiple? How to configure?

8. **Rate Limiting**: Per-user, per-client, or global? What are the limits?

### 14.3 Operational

9. **Audit Retention**: How long to keep audit logs? Where to store them (DB, log aggregator)?

10. **Geographic Distribution**: Single region or multi-region? Key synchronization strategy?

11. **Backward Compatibility**: How to handle API version changes? Deprecation policy?

12. **Monitoring**: What alerting thresholds? Who gets paged?

---

## 15. Appendices

### 15.1 Glossary

- **JWT**: JSON Web Token
- **JWKS**: JSON Web Key Set
- **IdP**: Identity Provider
- **STS**: Security Token Service
- **UPN**: User Principal Name
- **kid**: Key ID (JWT header parameter)
- **iss**: Issuer (JWT claim)
- **aud**: Audience (JWT claim)
- **exp**: Expiration time (JWT claim)
- **nbf**: Not before (JWT claim)
- **jti**: JWT ID (JWT claim)

### 15.2 References

- [RFC 7519 - JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)
- [RFC 7517 - JSON Web Key (JWK)](https://tools.ietf.org/html/rfc7517)
- [RFC 8693 - OAuth 2.0 Token Exchange](https://tools.ietf.org/html/rfc8693)
- [RFC 7807 - Problem Details for HTTP APIs](https://tools.ietf.org/html/rfc7807)
- [OWASP JWT Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)

### 15.3 Decision Log

| Date | Decision | Rationale | Owner |
|------|----------|-----------|-------|
| 2025-10-09 | Use Go for implementation | Performance, concurrency | Team |
| 2025-10-09 | 12-hour token lifetime | Balance security and UX | Security |
| TBD | Cache backend selection | Pending deployment model | Ops |

### 15.4 Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-09 | [Author] | Initial draft |

---

## Document Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Technical Lead | | | |
| Security Lead | | | |
| Product Owner | | | |
| Architecture Review | | | |
