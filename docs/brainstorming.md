# Security Token Service (STS) - Production Considerations Brainstorming

**Date**: 2025-10-09  
**Purpose**: Identify key aspects for building a production-grade STS service in Go

## Service Overview

The Security Token Service (STS) acts as a middleman that:
- Receives and validates JWTs from Identity Providers
- Enriches user data by querying entitlement systems
- Issues new JWTs with enriched claims (roles)
- Manages cryptographic keys with rotation strategy

**Token Lifetime**: 12 hours  
**Key Management**: Active key set with "kid" header identification and grace period rotation

---

## Security & Cryptography

### JWT Security
- **JWT Algorithm Enforcement**: Explicitly whitelist allowed signing algorithms (e.g., RS256, ES256) to prevent algorithm confusion attacks
- **Token Replay Prevention**: Consider adding `jti` (JWT ID) claim with short-term cache/bloom filter to detect replayed tokens
- **Audience & Issuer Validation**: Strict validation of `aud` and `iss` claims to prevent token substitution attacks
- **Clock Skew Tolerance**: Configure acceptable time drift (typically 30-60s) for `nbf` and `exp` validation

### Cryptographic Standards
- **Key Strength Requirements**: Define minimum key sizes (e.g., RSA 2048-bit minimum, prefer 4096-bit or ECDSA P-256)
- **Secure Key Storage**: Vault connection pooling, retry logic, and fallback mechanisms for key retrieval failures
- **Token Binding**: Consider binding tokens to client certificates or other proof-of-possession mechanisms for high-security scenarios

---

## Resilience & Reliability

### Fault Tolerance
- **Circuit Breakers**: Implement circuit breakers for external dependencies:
  - IdP well-known endpoint
  - Entitlement API
  - Vault
- **Timeout Configuration**: Granular timeouts for each external call (connection, read, total)
- **Retry Logic**: Exponential backoff with jitter for transient failures, distinguish retryable vs non-retryable errors
- **Graceful Degradation**: Define behavior when entitlement service is unavailable (fail closed vs issue limited token)

### Caching Strategy
- **IdP Public Keys**: Cache with TTL based on well-known config
- **Entitlement Data**: Cache with appropriate TTL (consider stale-while-revalidate pattern)
- **Negative Results**: Cache failed lookups to prevent thundering herd
- **Cache Invalidation**: Strategy for clearing stale data

### Traffic Management
- **Rate Limiting**: Per-client rate limiting to prevent abuse and protect downstream services
- **Bulkhead Pattern**: Isolate resource pools for different operations to prevent cascading failures
- **Load Shedding**: Reject requests when overloaded to maintain SLA for accepted requests

---

## Observability & Monitoring

### Structured Logging
- Log all token validations (success/failure with reason)
- Log key rotation events
- Include correlation IDs for request tracing
- **Sanitize sensitive data**: Never log full tokens, only metadata (kid, sub, exp, etc.)

### Metrics
- **Token Operations**:
  - Token issuance rate
  - Latency percentiles (p50, p95, p99)
  - Validation failure rates by reason (expired, invalid signature, etc.)
- **Caching**:
  - Cache hit/miss rates
  - Cache eviction rates
- **Dependencies**:
  - External dependency latency and error rates
  - Circuit breaker state changes
- **Key Management**:
  - Active key usage distribution
  - Key rotation events

### Tracing & Health
- **Distributed Tracing**: OpenTelemetry integration for end-to-end request tracing
- **Health Checks**:
  - Liveness probe (service is running)
  - Readiness probe (can serve traffic - vault accessible, keys loaded)
  - Dependency health checks with circuit breaker status

---

## Token Lifecycle Management

### Revocation & Validation
- **Token Revocation Strategy**: 
  - Emergency token revocation mechanism (distributed cache, revocation list)
  - Consider short-lived access tokens + refresh tokens pattern
- **Token Introspection Endpoint**: Allow downstream services to validate tokens
- **Token Refresh**: Consider offering refresh mechanism before 12-hour expiry

### Key Rotation
- **Grace Period Handling**: Clear policy for overlapping key validity during rotation
- **Maximum Token Lifetime**: Configurable cap even if client requests longer duration
- **Old Token Validation**: Support validating tokens signed with previous keys during grace period

---

## API Design & Validation

### Input Validation
- Strict schema validation for all inputs
- Maximum token size limits (prevent DoS)
- UPN format validation
- Prevent injection attacks in claims
- Content-Type enforcement

### Error Handling
- **Standardized Error Format**: RFC 7807 Problem Details
- **Error Message Security**: Don't leak sensitive information in error messages
- **HTTP Status Codes**: Distinguish client errors (4xx) from server errors (5xx)
- **Error Categories**:
  - Invalid token format
  - Expired token
  - Invalid signature
  - Unknown user
  - Entitlement service unavailable

### API Design
- **API Versioning**: Strategy for evolving the API without breaking clients
- **CORS Policy**: If accessed from browsers, define appropriate CORS rules
- **Request/Response Format**: JSON with clear schema

---

## Configuration & Feature Management

### Dynamic Configuration
- **Configuration Hot-Reload**: Ability to update configs without restart:
  - Cache TTLs
  - Timeouts
  - Rate limits
  - Feature flags
- **Feature Flags**: Toggle features like:
  - Caching strategies
  - Specific validation rules
  - Enrichment sources
  - Debug modes

### Multi-Environment Support
- **Environment-Specific Config**: Dev/staging/prod configuration management
- **Secret Rotation**: Handle vault secret rotation without downtime
- **Multi-Tenancy**: If supporting multiple IdPs or entitlement systems, tenant isolation strategy

---

## Performance & Scalability

### Resource Management
- **Connection Pooling**: HTTP client connection pools for external APIs
- **Concurrency Control**: Worker pool sizing, goroutine limits
- **Memory Management**: 
  - Token cache size limits
  - LRU eviction policies
  - Memory profiling hooks

### Optimization
- **Batch Operations**: If high volume, consider batching entitlement lookups
- **Horizontal Scaling**: Ensure service is stateless (or uses shared state store)
- **Async Operations**: Non-blocking operations where possible

---

## Compliance & Audit

### Audit Trail
- **Immutable Logging**: All token issuances with:
  - User identity (UPN)
  - Timestamp
  - Granted roles
  - Token expiry
  - Requesting client
- **Token Claims Audit**: Log what claims are added/modified during enrichment

### Data Protection
- **Data Residency**: Consider where user data is processed and stored
- **PII Handling**: 
  - Minimize PII in logs and caches
  - Encryption at rest if persisted
  - Data retention policies
- **Compliance Standards**: GDPR, SOC2, or industry-specific requirements

---

## Testing Strategy

### Test Levels
- **Unit Tests**: 
  - Crypto operations
  - Validation logic
  - Claim enrichment
  - Key rotation logic
- **Integration Tests**: 
  - Mock external dependencies
  - Test error scenarios
  - Test cache behavior
- **Contract Tests**: Verify compatibility with IdP and entitlement API contracts
- **End-to-End Tests**: Full token exchange flow

### Specialized Testing
- **Chaos Testing**: 
  - Simulate dependency failures
  - Network issues
  - Key rotation during traffic
- **Performance Tests**: 
  - Load testing to establish baseline
  - Identify bottlenecks
  - Stress testing
- **Security Tests**: 
  - Fuzzing
  - Penetration testing
  - Token manipulation attempts
  - Algorithm confusion attacks

---

## Key Rotation Specifics

### Rotation Strategy
- **Rotation Frequency**: Define rotation schedule (e.g., every 30/90 days)
- **Automated Rotation**: Trigger mechanism (cron, event-based)
- **Key Overlap Period**: How long old keys remain valid for verification
- **Key Rollback**: Procedure if new key is compromised

### Key Metadata
- Track key creation time
- Usage count per key
- Last used timestamp
- Deprecation status

### Distribution
- **Multi-Region Key Sync**: If distributed, ensure consistent key availability
- **Key Discovery**: How services discover new keys (JWKS endpoint)

---

## Edge Cases & Error Scenarios

### Input Validation Edge Cases
- **Malformed JWT**: Handle non-JWT inputs gracefully
- **Oversized Tokens**: Reject tokens exceeding size limits
- **Missing Required Claims**: Handle incomplete JWTs

### Business Logic Edge Cases
- **Unknown User**: Policy when UPN not found in entitlement system
- **Empty Roles**: Decide if tokens with no roles are valid
- **Entitlement API Pagination**: Handle large role sets
- **Duplicate Roles**: Deduplication strategy

### Timing & Concurrency
- **IdP Key Rotation**: Handle scenario where IdP rotates keys mid-flight
- **Concurrent Requests**: Same user making multiple concurrent token requests
- **Token Expiry During Processing**: Handle tokens that expire during enrichment

### Dependency Failures
- **Vault Unavailable**: Fallback strategy or fail fast
- **Entitlement API Timeout**: Partial data vs complete failure
- **IdP Well-Known Endpoint Down**: Use cached keys or reject requests

---

## Go-Specific Considerations

### Language Features
- **Context Propagation**: Use context.Context for cancellation and timeouts
- **Goroutine Management**: Proper lifecycle management, avoid leaks
- **Error Handling**: Wrap errors with context, use structured error types
- **Dependency Injection**: Clean architecture with interfaces

### Libraries & Frameworks
- **JWT Libraries**: `golang-jwt/jwt` or `lestrrat-go/jwx`
- **HTTP Framework**: `gin`, `echo`, or standard `net/http` with middleware
- **Vault Client**: HashiCorp Vault official Go client
- **Metrics**: Prometheus client
- **Logging**: `zap` or `zerolog` for structured logging
- **Configuration**: `viper` or `envconfig`

### Best Practices
- **Graceful Shutdown**: Handle SIGTERM/SIGINT properly
- **Resource Cleanup**: Defer statements, proper connection closing
- **Panic Recovery**: Middleware to recover from panics
- **Memory Profiling**: pprof endpoints for debugging

---

## Questions to Answer

1. **Token Revocation**: What's the strategy for emergency revocation? Distributed cache? Revocation list?
2. **Entitlement Caching**: How stale can entitlement data be? What's the TTL?
3. **Failure Mode**: When entitlement service is down, fail closed (reject) or fail open (issue limited token)?
4. **Multi-Tenancy**: Single IdP or multiple? Single entitlement system or multiple?
5. **Token Refresh**: Support refresh tokens or require re-authentication after 12 hours?
6. **Key Algorithm**: RSA or ECDSA? What key size?
7. **Rate Limiting**: Per-user, per-client, or global? What limits?
8. **Audit Retention**: How long to keep audit logs? Where to store them?
9. **Geographic Distribution**: Single region or multi-region deployment?
10. **Backward Compatibility**: How to handle API changes without breaking existing clients?

---

## Next Steps

1. Create detailed technical design document
2. Define API contracts (OpenAPI/Swagger spec)
3. Design database/cache schema (if needed)
4. Create project structure and interfaces
5. Define configuration schema
6. Set up observability stack integration
7. Create test strategy document
8. Define deployment architecture (future phase)
