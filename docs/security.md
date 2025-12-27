# Security

Lattice provides built-in authentication with API Keys and JWT tokens, using secure constant-time comparison.

## API Key Authentication

### Setup

```go
import "github.com/storo/lettice/pkg/security"

// Create API key authenticator
apiKeyAuth := security.NewAPIKeyAuth()

// Register keys with roles and permissions
apiKeyAuth.RegisterKey("sk-production-key", &security.KeyEntry{
    AgentID:     "service-account",
    Roles:       []string{"admin"},
    Permissions: []string{"*"},
})

apiKeyAuth.RegisterKey("sk-readonly-key", &security.KeyEntry{
    AgentID:     "reader",
    Roles:       []string{"user"},
    Permissions: []string{"agents:read", "mesh:run"},
})
```

### Key Entry Options

```go
&security.KeyEntry{
    AgentID:     "user-123",          // Identifier for this key
    Roles:       []string{"admin"},   // Role-based access
    Permissions: []string{"*"},       // Fine-grained permissions
    ExpiresAt:   1735689600,          // Unix timestamp (0 = never)
}
```

### Revoke Keys

```go
apiKeyAuth.RevokeKey("sk-compromised-key")
```

## JWT Authentication

### Setup

```go
jwtAuth := security.NewJWTAuth("your-secret-key")
```

### Generate Tokens

```go
token, err := jwtAuth.Generate(&security.JWTClaims{
    AgentID:     "user-123",
    Roles:       []string{"user"},
    Permissions: []string{"mesh:run"},
    ExpiresAt:   time.Now().Add(24 * time.Hour),
})
```

### Validate Tokens

```go
claims, err := jwtAuth.Validate(ctx, token)
if err != nil {
    log.Fatal("Invalid token")
}
fmt.Println(claims.AgentID)
```

## Unified Auth

Combine multiple authentication methods:

```go
auth := security.NewAuth(
    security.WithAPIKeyAuth(apiKeyAuth),
    security.WithJWTAuth(jwtAuth),
)
```

### Configuration Options

```go
auth := security.NewAuth(
    security.WithAPIKeyAuth(apiKeyAuth),
    security.WithJWTAuth(jwtAuth),
    security.WithAPIKeyHeader("X-API-Key"),      // Default header
    security.WithQueryParamAuth("api_key"),      // Enable query param
)
```

### Authenticate Requests

```go
// From HTTP request
claims, err := auth.AuthenticateRequest(ctx, httpRequest)

// Direct authentication
claims, err := auth.Authenticate(ctx, "sk-api-key")
```

Authentication order:
1. Authorization header (Bearer token for JWT)
2. API Key header (X-API-Key)
3. Query parameter (if enabled)

## Claims

```go
claims, _ := auth.AuthenticateRequest(ctx, req)

// Check identity
fmt.Println(claims.AgentID)   // "user-123"
fmt.Println(claims.Method)    // "api_key" or "jwt"

// Check roles
if claims.HasRole("admin") {
    // Admin access
}

if claims.HasAnyRole("admin", "moderator") {
    // Admin or moderator
}

if claims.HasAllRoles("user", "verified") {
    // Must have both roles
}

// Check permissions
if claims.HasPermission("mesh:run") {
    // Can run mesh
}

// Check expiration
if claims.ExpiresAt > 0 && claims.ExpiresAt < time.Now().Unix() {
    // Expired
}
```

## HTTP Server Integration

```go
import (
    "github.com/storo/lettice/pkg/protocol/http"
    "github.com/storo/lettice/pkg/security"
)

// Setup auth
apiKeyAuth := security.NewAPIKeyAuth()
apiKeyAuth.RegisterKey("demo-key", &security.KeyEntry{
    AgentID: "demo",
    Roles:   []string{"user"},
})

auth := security.NewAuth(
    security.WithAPIKeyAuth(apiKeyAuth),
)

// Create server with auth
server := http.NewServer(mesh, http.WithAuth(auth))
server.ListenAndServe(":8080")
```

Requests require authentication:

```bash
# Without auth - 401 Unauthorized
curl http://localhost:8080/agents

# With API key - Success
curl -H "X-API-Key: demo-key" http://localhost:8080/agents

# With JWT - Success
curl -H "Authorization: Bearer eyJ..." http://localhost:8080/agents
```

Health endpoint is always public:

```bash
curl http://localhost:8080/health  # No auth required
```

## Security Best Practices

### Constant-Time Comparison

API key validation uses constant-time comparison to prevent timing attacks:

```go
// Keys are hashed with SHA-256
// Comparison uses crypto/subtle.ConstantTimeCompare
// All registered keys are checked (no early exit)
```

### Key Storage

```go
// Keys are stored as SHA-256 hashes
// Original keys are never stored in memory
apiKeyAuth.RegisterKey("sk-secret", entry)
// Internally stores: sha256("sk-secret") -> entry
```

### Key Expiration

```go
// Set expiration for temporary access
apiKeyAuth.RegisterKey("temp-key", &security.KeyEntry{
    AgentID:   "temp-user",
    ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
})
```

### Permission Patterns

```go
// Wildcard for admin
Permissions: []string{"*"}

// Specific actions
Permissions: []string{"agents:read", "mesh:run"}

// Resource-specific
Permissions: []string{"agents:researcher:run"}
```

## Errors

```go
import "github.com/storo/lettice/pkg/security"

switch err {
case security.ErrNoCredentials:
    // No auth header or key provided
case security.ErrInvalidAPIKey:
    // Key not found
case security.ErrExpiredAPIKey:
    // Key has expired
case security.ErrInvalidToken:
    // JWT validation failed
case security.ErrExpiredToken:
    // JWT has expired
}
```

## Complete Example

```go
package main

import (
    "log"
    "time"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/protocol/http"
    "github.com/storo/lettice/pkg/provider"
    "github.com/storo/lettice/pkg/security"
)

func main() {
    // Setup API key auth
    apiKeyAuth := security.NewAPIKeyAuth()

    // Production key (never expires)
    apiKeyAuth.RegisterKey("sk-prod-123", &security.KeyEntry{
        AgentID:     "production-service",
        Roles:       []string{"admin"},
        Permissions: []string{"*"},
    })

    // Temporary key (expires in 1 hour)
    apiKeyAuth.RegisterKey("sk-temp-456", &security.KeyEntry{
        AgentID:     "temp-access",
        Roles:       []string{"user"},
        Permissions: []string{"mesh:run"},
        ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
    })

    // Setup JWT auth
    jwtAuth := security.NewJWTAuth("super-secret-key")

    // Combine auth methods
    auth := security.NewAuth(
        security.WithAPIKeyAuth(apiKeyAuth),
        security.WithJWTAuth(jwtAuth),
    )

    // Create mesh and agents
    llm := provider.NewMockWithResponse("Hello!")
    agent := lattice.NewAgent("assistant").Model(llm).Build()

    mesh := lattice.NewMesh()
    mesh.Register(agent)

    // Start secure server
    server := http.NewServer(mesh, http.WithAuth(auth))
    log.Fatal(server.ListenAndServe(":8080"))
}
```

## Next Steps

- [HTTP API](http-api.md) - Full API reference
- [Mesh](mesh.md) - Mesh configuration
- [Getting Started](getting-started.md) - Quick start guide
