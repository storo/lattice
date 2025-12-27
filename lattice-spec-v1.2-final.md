# Lattice: Agent Mesh Framework para Go

## Especificación Técnica v1.2 (Final)

> **Changelog**:
> - v1.0: Diseño inicial
> - v1.1: Corrige ciclo de importación, añade injector, storage, security
> - v1.2: Añade detección de ciclos, mejora JSON Schema, security hardening

---

## 1. Visión General

**Lattice** es un framework de Agent Mesh en Go para construir sistemas de agentes de IA distribuidos con mínimo código.

### Filosofía

```
"Complexity hidden, power exposed"
```

- **Mínimo código** para el desarrollador
- **Máxima flexibilidad** bajo el capó
- **Production-ready** desde el día uno
- **Protocolos abiertos** (A2A, MCP, gRPC, HTTP)

### Qué es un Agent Mesh

A diferencia de frameworks tradicionales donde defines conexiones explícitas entre agentes (grafos), en Lattice:

1. Cada agente declara **qué capacidades provee** (`Provides`)
2. Cada agente declara **qué capacidades necesita** (`Needs`)
3. El **Mesh** conecta todo automáticamente en runtime
4. El **Mesh inyecta los agentes descubiertos como Tools** al LLM
5. El **Mesh detecta y previene ciclos de ejecución**

---

## 2. Arquitectura

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LATTICE FRAMEWORK                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Agent     │  │    Mesh     │  │  Patterns   │  │  Registry   │        │
│  │   Builder   │  │   Runtime   │  │   Library   │  │  Discovery  │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │               │
│         └────────────────┴────────────────┴────────────────┘               │
│                                   │                                         │
│                          ┌────────┴────────┐                               │
│                          │   Core Runtime   │                               │
│                          │  + Cycle Detect  │                               │
│                          └────────┬────────┘                               │
│                                   │                                         │
│  ┌────────────────────────────────┼────────────────────────────────────┐   │
│  │                        Security Layer                                │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐                 │   │
│  │  │  mTLS   │  │   JWT   │  │ API Key │  │  RBAC   │                 │   │
│  │  │         │  │         │  │(hashed) │  │         │                 │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                   │                                         │
│  ┌────────────────────────────────┼────────────────────────────────────┐   │
│  │                         Protocol Layer                               │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │   │
│  │  │   A2A   │  │   MCP   │  │  gRPC   │  │  HTTP   │  │  NATS   │   │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                   │                                         │
│  ┌────────────────────────────────┼────────────────────────────────────┐   │
│  │                        Storage Layer                                 │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐                 │   │
│  │  │ Memory  │  │  Redis  │  │Postgres │  │  SQLite │                 │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                   │                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Providers (LLMs)                              │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │   │
│  │  │ OpenAI  │  │Anthropic│  │ Ollama  │  │ Groq    │  │ Custom  │   │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Estructura del Proyecto

```
lattice/
├── cmd/
│   └── lattice/                  # CLI tool
│       └── main.go
├── pkg/
│   ├── core/                     # INTERFACES COMPARTIDAS
│   │   ├── types.go              # Agent, Mesh, Tool interfaces
│   │   ├── capability.go         # Capability type
│   │   ├── result.go             # Result, Error types
│   │   ├── context.go            # Execution context + call chain
│   │   └── schema.go             # ⚠️ NUEVO: JSON Schema helpers
│   │
│   ├── agent/                    # Implementación de Agent
│   │   ├── agent.go
│   │   ├── builder.go
│   │   ├── executor.go
│   │   └── card.go
│   │
│   ├── mesh/                     # El corazón del sistema
│   │   ├── mesh.go
│   │   ├── router.go
│   │   ├── balancer.go
│   │   ├── resolver.go
│   │   ├── injector.go
│   │   ├── orchestrator.go
│   │   └── cycle.go              # ⚠️ NUEVO: Detección de ciclos
│   │
│   ├── registry/
│   │   ├── registry.go
│   │   ├── local.go
│   │   ├── consul.go
│   │   └── etcd.go
│   │
│   ├── storage/
│   │   ├── store.go
│   │   ├── memory.go
│   │   ├── redis.go
│   │   └── postgres.go
│   │
│   ├── security/
│   │   ├── auth.go
│   │   ├── mtls.go
│   │   ├── jwt.go
│   │   ├── apikey.go             # ⚠️ MEJORADO: Constant-time compare
│   │   └── rbac.go
│   │
│   ├── protocol/
│   │   ├── protocol.go
│   │   ├── trace.go
│   │   ├── a2a/
│   │   ├── mcp/
│   │   ├── grpc/
│   │   └── http/
│   │
│   ├── provider/
│   │   ├── provider.go
│   │   ├── anthropic/
│   │   ├── openai/
│   │   ├── ollama/
│   │   └── groq/
│   │
│   ├── tool/
│   │   ├── tool.go
│   │   ├── registry.go
│   │   ├── schema.go             # ⚠️ NUEVO: Schema generation
│   │   └── builtin/
│   │
│   ├── memory/
│   │   ├── memory.go
│   │   ├── conversation.go
│   │   └── shared.go
│   │
│   ├── patterns/
│   │   ├── react.go
│   │   ├── supervisor.go
│   │   ├── sequential.go
│   │   ├── parallel.go
│   │   └── ...
│   │
│   ├── middleware/
│   │   ├── logging.go
│   │   ├── tracing.go
│   │   ├── metrics.go
│   │   ├── ratelimit.go
│   │   ├── retry.go
│   │   └── auth.go
│   │
│   └── config/
│       ├── config.go
│       └── loader.go
│
├── internal/
│   ├── util/
│   └── errors/
│
├── examples/
├── docs/
├── go.mod
└── README.md
```

---

## 4. Interfaces Core (pkg/core/)

### 4.1 Types Core

```go
// pkg/core/types.go

package core

import (
    "context"
    "encoding/json"
    "time"
)

// ============================================================================
// CAPABILITY
// ============================================================================

type Capability string

const (
    CapResearch   Capability = "research"
    CapWriting    Capability = "writing"
    CapCoding     Capability = "coding"
    CapAnalysis   Capability = "analysis"
    CapPlanning   Capability = "planning"
)

func Cap(name string) Capability { return Capability(name) }

// ============================================================================
// AGENT INTERFACE
// ============================================================================

type Agent interface {
    ID() string
    Name() string
    Description() string
    Provides() []Capability
    Needs() []Capability
    Run(ctx context.Context, input string) (*Result, error)
    RunStream(ctx context.Context, input string) (<-chan StreamChunk, error)
    Stop() error
    Card() *AgentCard
    Tools() []Tool
}

// ============================================================================
// TOOL INTERFACE
// ============================================================================

type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}

// ============================================================================
// RESULT
// ============================================================================

type Result struct {
    Output    string
    Metadata  map[string]any
    TokensIn  int
    TokensOut int
    Duration  time.Duration
    TraceID   string
    CallChain []string  // ⚠️ NUEVO: Para detección de ciclos
    Error     error
}

type StreamChunk struct {
    Content string
    Done    bool
    Error   error
}

// ============================================================================
// AGENT CARD
// ============================================================================

type AgentCard struct {
    Name         string           `json:"name"`
    Description  string           `json:"description"`
    URL          string           `json:"url"`
    Version      string           `json:"version"`
    Capabilities CardCapabilities `json:"capabilities"`
    Skills       []Skill          `json:"skills"`
    Tools        []string         `json:"tools"`
    Model        string           `json:"model"`
    Protocols    []string         `json:"protocols"`
    InputSchema  json.RawMessage  `json:"input_schema,omitempty"`
    OutputSchema json.RawMessage  `json:"output_schema,omitempty"`
    Metadata     map[string]string `json:"metadata,omitempty"`
}

type CardCapabilities struct {
    Provides  []Capability `json:"provides"`
    Needs     []Capability `json:"needs"`
    Streaming bool         `json:"streaming"`
}

type Skill struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

### 4.2 Execution Context (Detección de Ciclos)

```go
// pkg/core/context.go

package core

import "context"

// ============================================================================
// EXECUTION CONTEXT - Para tracking de call chain y detección de ciclos
// ============================================================================

type contextKey string

const (
    callChainKey    contextKey = "lattice.call_chain"
    hopCountKey     contextKey = "lattice.hop_count"
    traceIDKey      contextKey = "lattice.trace_id"
    timeoutKey      contextKey = "lattice.timeout"
)

// CallChain retorna la cadena de agentes que han sido llamados
func CallChain(ctx context.Context) []string {
    if chain, ok := ctx.Value(callChainKey).([]string); ok {
        return chain
    }
    return nil
}

// WithCallChain añade un agente a la cadena de llamadas
func WithCallChain(ctx context.Context, agentID string) context.Context {
    chain := CallChain(ctx)
    newChain := make([]string, len(chain)+1)
    copy(newChain, chain)
    newChain[len(chain)] = agentID
    return context.WithValue(ctx, callChainKey, newChain)
}

// HopCount retorna el número de saltos en la cadena
func HopCount(ctx context.Context) int {
    if count, ok := ctx.Value(hopCountKey).(int); ok {
        return count
    }
    return 0
}

// WithHopCount incrementa el contador de saltos
func WithHopCount(ctx context.Context) context.Context {
    return context.WithValue(ctx, hopCountKey, HopCount(ctx)+1)
}

// TraceID retorna el trace ID de la ejecución
func TraceID(ctx context.Context) string {
    if id, ok := ctx.Value(traceIDKey).(string); ok {
        return id
    }
    return ""
}

// WithTraceID establece el trace ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
}

// InCallChain verifica si un agente ya está en la cadena (ciclo detectado)
func InCallChain(ctx context.Context, agentID string) bool {
    for _, id := range CallChain(ctx) {
        if id == agentID {
            return true
        }
    }
    return false
}
```

### 4.3 JSON Schema Helpers

```go
// pkg/core/schema.go

package core

import (
    "encoding/json"
    "reflect"
)

// ============================================================================
// JSON SCHEMA GENERATION
// ============================================================================

// SchemaProperty representa una propiedad del schema
type SchemaProperty struct {
    Type        string `json:"type"`
    Description string `json:"description,omitempty"`
}

// Schema representa un JSON Schema básico
type Schema struct {
    Type       string                    `json:"type"`
    Properties map[string]SchemaProperty `json:"properties,omitempty"`
    Required   []string                  `json:"required,omitempty"`
}

// ToJSON convierte el schema a json.RawMessage
func (s *Schema) ToJSON() json.RawMessage {
    data, _ := json.Marshal(s)
    return data
}

// NewObjectSchema crea un schema de objeto con propiedades
func NewObjectSchema() *SchemaBuilder {
    return &SchemaBuilder{
        schema: &Schema{
            Type:       "object",
            Properties: make(map[string]SchemaProperty),
            Required:   []string{},
        },
    }
}

// SchemaBuilder para construir schemas fluídamente
type SchemaBuilder struct {
    schema *Schema
}

func (b *SchemaBuilder) Property(name, typ, desc string) *SchemaBuilder {
    b.schema.Properties[name] = SchemaProperty{Type: typ, Description: desc}
    return b
}

func (b *SchemaBuilder) Required(names ...string) *SchemaBuilder {
    b.schema.Required = append(b.schema.Required, names...)
    return b
}

func (b *SchemaBuilder) Build() json.RawMessage {
    return b.schema.ToJSON()
}

// ============================================================================
// SCHEMA FROM STRUCT (usando reflection)
// ============================================================================

// SchemaFromStruct genera un JSON Schema desde una struct Go
// Usa tags: `json:"name"` y `schema:"description"`
func SchemaFromStruct(v any) json.RawMessage {
    t := reflect.TypeOf(v)
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    schema := &Schema{
        Type:       "object",
        Properties: make(map[string]SchemaProperty),
        Required:   []string{},
    }
    
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        
        // Obtener nombre del tag json
        jsonTag := field.Tag.Get("json")
        if jsonTag == "" || jsonTag == "-" {
            continue
        }
        // Remover ",omitempty" si existe
        name := jsonTag
        if idx := len(jsonTag); idx > 0 {
            for j, c := range jsonTag {
                if c == ',' {
                    name = jsonTag[:j]
                    break
                }
            }
        }
        
        // Obtener descripción del tag schema
        desc := field.Tag.Get("schema")
        
        // Mapear tipo Go a JSON Schema type
        jsonType := goTypeToJSONType(field.Type)
        
        schema.Properties[name] = SchemaProperty{
            Type:        jsonType,
            Description: desc,
        }
        
        // Si no tiene omitempty, es required
        if !containsOmitempty(jsonTag) {
            schema.Required = append(schema.Required, name)
        }
    }
    
    return schema.ToJSON()
}

func goTypeToJSONType(t reflect.Type) string {
    switch t.Kind() {
    case reflect.String:
        return "string"
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
         reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return "integer"
    case reflect.Float32, reflect.Float64:
        return "number"
    case reflect.Bool:
        return "boolean"
    case reflect.Slice, reflect.Array:
        return "array"
    case reflect.Map, reflect.Struct:
        return "object"
    default:
        return "string"
    }
}

func containsOmitempty(tag string) bool {
    return len(tag) > 0 && tag[len(tag)-1] == 'y' // ends with "omitempty"
}
```

---

## 5. Detección de Ciclos de Ejecución (CRÍTICO)

### 5.1 El Problema

```
Agente A (Writer) necesita "review"
Agente B (Reviewer) necesita "writing"

A.Run() → llama a B (via Tool)
  B.Run() → llama a A (via Tool)
    A.Run() → llama a B (via Tool)
      ... BOOM 💥 (Stack overflow + $$$)
```

### 5.2 La Solución

```go
// pkg/mesh/cycle.go

package mesh

import (
    "errors"
    "fmt"
    
    "github.com/voidlab/lattice/pkg/core"
)

var (
    ErrCycleDetected     = errors.New("execution cycle detected")
    ErrMaxHopsExceeded   = errors.New("maximum hop count exceeded")
)

// CycleDetector previene loops infinitos entre agentes
type CycleDetector struct {
    maxHops int
}

func NewCycleDetector(maxHops int) *CycleDetector {
    if maxHops <= 0 {
        maxHops = 10 // Default sensato
    }
    return &CycleDetector{maxHops: maxHops}
}

// Check verifica si es seguro ejecutar el agente
func (cd *CycleDetector) Check(ctx context.Context, agentID string) error {
    // 1. Verificar si el agente ya está en la cadena (ciclo directo)
    if core.InCallChain(ctx, agentID) {
        chain := core.CallChain(ctx)
        return fmt.Errorf("%w: %v -> %s", ErrCycleDetected, chain, agentID)
    }
    
    // 2. Verificar hop count (previene cadenas muy largas)
    if core.HopCount(ctx) >= cd.maxHops {
        return fmt.Errorf("%w: limit is %d", ErrMaxHopsExceeded, cd.maxHops)
    }
    
    return nil
}

// PrepareContext prepara el contexto para la siguiente ejecución
func (cd *CycleDetector) PrepareContext(ctx context.Context, agentID string) context.Context {
    ctx = core.WithCallChain(ctx, agentID)
    ctx = core.WithHopCount(ctx)
    return ctx
}
```

### 5.3 Integración en Injector

```go
// pkg/mesh/injector.go (actualizado)

package mesh

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/voidlab/lattice/pkg/core"
)

// AgentToolInput define el schema de entrada para AgentTool
// Usamos struct + tags para generar el schema automáticamente
type AgentToolInput struct {
    Task    string `json:"task" schema:"The specific task or question to delegate to the specialized agent"`
    Context string `json:"context,omitempty" schema:"Additional context that might help the agent understand the task better"`
}

// AgentTool wrappea un agente como Tool con detección de ciclos
type AgentTool struct {
    capability    core.Capability
    providers     []core.Agent
    balancer      Balancer
    cycleDetector *CycleDetector
}

func (t *AgentTool) Name() string {
    return fmt.Sprintf("delegate_to_%s", t.capability)
}

func (t *AgentTool) Description() string {
    return fmt.Sprintf(
        "Delegate a task to a specialized agent that provides '%s' capability. "+
        "Use this when you need expert help with %s-related tasks. "+
        "The agent will process your request and return the result.",
        t.capability, t.capability,
    )
}

func (t *AgentTool) Schema() json.RawMessage {
    // ⚠️ MEJORADO: Usar SchemaFromStruct en lugar de map[string]any manual
    return core.SchemaFromStruct(AgentToolInput{})
}

func (t *AgentTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
    var input AgentToolInput
    if err := json.Unmarshal(params, &input); err != nil {
        return "", fmt.Errorf("invalid input: %w", err)
    }
    
    // Seleccionar proveedor
    provider := t.balancer.Select(t.providers)
    
    // ⚠️ CRÍTICO: Verificar ciclos ANTES de ejecutar
    if err := t.cycleDetector.Check(ctx, provider.ID()); err != nil {
        return "", err
    }
    
    // Preparar contexto con call chain actualizada
    ctx = t.cycleDetector.PrepareContext(ctx, provider.ID())
    
    // ⚠️ IMPORTANTE: Ajustar timeout para agentes downstream
    // El agente delegado puede tardar, necesita tiempo suficiente
    // Esto se configura en el Mesh, no aquí
    
    // Construir input
    fullInput := input.Task
    if input.Context != "" {
        fullInput = fmt.Sprintf("Context: %s\n\nTask: %s", input.Context, input.Task)
    }
    
    // Ejecutar
    result, err := provider.Run(ctx, fullInput)
    if err != nil {
        return "", fmt.Errorf("agent %s failed: %w", provider.Name(), err)
    }
    
    return result.Output, nil
}
```

### 5.4 Configuración de MaxHops

```yaml
# lattice.yaml
mesh:
  # Detección de ciclos
  cycle_detection:
    enabled: true
    max_hops: 10  # Máximo de agentes en una cadena de ejecución
```

---

## 6. Security Hardening

### 6.1 API Key con Constant-Time Compare

```go
// pkg/security/apikey.go

package security

import (
    "context"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"
    "errors"
    "sync"
)

var (
    ErrInvalidAPIKey = errors.New("invalid API key")
    ErrAPIKeyExpired = errors.New("API key expired")
)

// APIKeyAuth autenticación por API Key con timing-safe comparison
type APIKeyAuth struct {
    mu   sync.RWMutex
    keys map[string]*APIKeyEntry // hash(key) -> entry
}

type APIKeyEntry struct {
    AgentID     string
    Roles       []string
    Permissions []string
    ExpiresAt   int64 // Unix timestamp, 0 = no expira
}

func NewAPIKeyAuth() *APIKeyAuth {
    return &APIKeyAuth{
        keys: make(map[string]*APIKeyEntry),
    }
}

// RegisterKey registra una API key (almacena el hash, no la key plana)
func (a *APIKeyAuth) RegisterKey(key string, entry *APIKeyEntry) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    hash := hashKey(key)
    a.keys[hash] = entry
}

// Authenticate verifica una API key de forma segura
func (a *APIKeyAuth) Authenticate(ctx context.Context, key string) (*Claims, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    keyHash := hashKey(key)
    
    // ⚠️ CRÍTICO: Iterar sobre TODAS las keys para evitar timing attacks
    // No podemos hacer lookup directo porque revelaría si el hash existe
    var matchedEntry *APIKeyEntry
    
    for storedHash, entry := range a.keys {
        // ⚠️ CRÍTICO: Constant-time comparison
        if constantTimeEqual(keyHash, storedHash) {
            matchedEntry = entry
            // NO hacer break - seguir iterando para mantener tiempo constante
        }
    }
    
    if matchedEntry == nil {
        // Simular trabajo para evitar timing attack
        dummyCompare()
        return nil, ErrInvalidAPIKey
    }
    
    // Verificar expiración
    if matchedEntry.ExpiresAt > 0 {
        now := time.Now().Unix()
        if now > matchedEntry.ExpiresAt {
            return nil, ErrAPIKeyExpired
        }
    }
    
    return &Claims{
        AgentID:     matchedEntry.AgentID,
        Roles:       matchedEntry.Roles,
        Permissions: matchedEntry.Permissions,
        ExpiresAt:   matchedEntry.ExpiresAt,
    }, nil
}

// hashKey genera un hash SHA-256 de la key
func hashKey(key string) string {
    h := sha256.Sum256([]byte(key))
    return hex.EncodeToString(h[:])
}

// constantTimeEqual compara dos strings en tiempo constante
func constantTimeEqual(a, b string) bool {
    // Asegurar misma longitud para subtle.ConstantTimeCompare
    if len(a) != len(b) {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// dummyCompare simula trabajo para mantener timing constante
func dummyCompare() {
    dummy := "0000000000000000000000000000000000000000000000000000000000000000"
    constantTimeEqual(dummy, dummy)
}
```

---

## 7. Timeouts para Agent Delegation

### 7.1 El Problema

```
Writer llama a Researcher (via Tool)
Researcher tarda 40 segundos generando un reporte
Writer tiene timeout de 30 segundos
→ Writer falla antes de recibir el resultado
```

### 7.2 La Solución: Timeout Propagation

```go
// pkg/mesh/timeout.go

package mesh

import (
    "context"
    "time"
)

// TimeoutConfig configuración de timeouts
type TimeoutConfig struct {
    // Timeout base para un agente
    BaseTimeout time.Duration
    
    // Factor multiplicador por cada hop
    // Si BaseTimeout = 30s y HopMultiplier = 1.5:
    // - Hop 0 (root): 30s
    // - Hop 1: 45s
    // - Hop 2: 67.5s
    HopMultiplier float64
    
    // Timeout máximo absoluto
    MaxTimeout time.Duration
}

func DefaultTimeoutConfig() *TimeoutConfig {
    return &TimeoutConfig{
        BaseTimeout:   30 * time.Second,
        HopMultiplier: 1.5,
        MaxTimeout:    5 * time.Minute,
    }
}

// CalculateTimeout calcula el timeout apropiado basado en el hop count
func (tc *TimeoutConfig) CalculateTimeout(ctx context.Context) time.Duration {
    hopCount := core.HopCount(ctx)
    
    // Calcular timeout con multiplicador
    timeout := tc.BaseTimeout
    for i := 0; i < hopCount; i++ {
        timeout = time.Duration(float64(timeout) * tc.HopMultiplier)
    }
    
    // Limitar al máximo
    if timeout > tc.MaxTimeout {
        timeout = tc.MaxTimeout
    }
    
    return timeout
}

// WithCalculatedTimeout aplica el timeout calculado al contexto
func (tc *TimeoutConfig) WithCalculatedTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
    timeout := tc.CalculateTimeout(ctx)
    return context.WithTimeout(ctx, timeout)
}
```

### 7.3 Configuración

```yaml
# lattice.yaml
mesh:
  timeouts:
    base: 30s
    hop_multiplier: 1.5
    max: 5m
```

---

## 8. Configuración Completa

```yaml
# lattice.yaml
lattice:
  name: "my-agent-system"
  version: "1.0.0"

defaults:
  model:
    provider: anthropic
    name: claude-sonnet-4-20250514
    temperature: 0.7
    max_tokens: 4096
  
  timeout: 30s
  retries: 3

mesh:
  discovery:
    type: local  # local | consul | etcd | kubernetes
  
  load_balancer: round_robin  # round_robin | least_conn | random
  
  # ⚠️ Detección de ciclos
  cycle_detection:
    enabled: true
    max_hops: 10
  
  # ⚠️ Timeouts para delegation
  timeouts:
    base: 30s
    hop_multiplier: 1.5
    max: 5m

storage:
  type: memory  # memory | redis | postgres | sqlite
  redis:
    address: "localhost:6379"
    password: ""
    db: 0

security:
  mtls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    ca_file: "/path/to/ca.pem"
  
  jwt:
    enabled: false
    secret: "${JWT_SECRET}"
    issuer: "lattice"
  
  api_keys:
    enabled: true
    # ⚠️ Las keys se hashean automáticamente
    keys:
      - key: "${API_KEY_1}"
        agent_id: "external-client-1"
        roles: ["read", "execute"]
        expires_at: 0  # 0 = no expira

protocols:
  a2a:
    enabled: true
    port: 8080
  mcp:
    enabled: true
    port: 8081
    transport: http
  grpc:
    enabled: true
    port: 9090
  http:
    enabled: true
    port: 8000

observability:
  tracing:
    enabled: true
    exporter: otlp
    endpoint: "localhost:4317"
    propagation: true
  metrics:
    enabled: true
    port: 9100
  logging:
    level: info
    format: json

providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
  openai:
    api_key: ${OPENAI_API_KEY}
  ollama:
    base_url: "http://localhost:11434"
```

---

## 9. API de Usuario Final

### 9.1 Simple Agent

```go
package main

import (
    "context"
    "fmt"
    "github.com/voidlab/lattice"
)

func main() {
    ctx := context.Background()
    
    result, _ := lattice.Agent("greeter").
        Model(lattice.Claude()).
        Run(ctx, "Hello!")
    
    fmt.Println(result.Output)
}
```

### 9.2 Multi-Agent con Ciclo Detection

```go
package main

import (
    "context"
    "fmt"
    "github.com/voidlab/lattice"
)

func main() {
    ctx := context.Background()
    
    mesh := lattice.NewMesh(
        lattice.WithMaxHops(5),  // Máximo 5 agentes en cadena
    )
    
    // Researcher provee research
    researcher := lattice.Agent("researcher").
        Model(lattice.Claude()).
        Provides(lattice.CapResearch).
        Build()
    
    // Writer necesita research
    writer := lattice.Agent("writer").
        Model(lattice.Claude()).
        Needs(lattice.CapResearch).
        Provides(lattice.CapWriting).
        Build()
    
    // Reviewer necesita writing (pero NO necesita research, evita ciclo)
    reviewer := lattice.Agent("reviewer").
        Model(lattice.Claude()).
        Needs(lattice.CapWriting).
        Build()
    
    mesh.Register(researcher, writer, reviewer)
    
    // Cadena: reviewer -> writer -> researcher (OK, 3 hops)
    result, err := mesh.Run(ctx, "Review an article about Go")
    if err != nil {
        // Si hay ciclo, error aquí
        fmt.Println("Error:", err)
        return
    }
    
    fmt.Println(result.Output)
}
```

### 9.3 Observar Call Chain

```go
result, _ := mesh.Run(ctx, "Complex task")

// Ver la cadena de ejecución
fmt.Println("Call chain:", result.CallChain)
// Output: [reviewer, writer, researcher]

fmt.Println("Trace ID:", result.TraceID)
// Output: abc123-def456-...
```

---

## 10. Fases de Implementación

### Fase 1: Core Foundation (Semanas 1-2)
- [ ] `pkg/core/types.go` - Interfaces
- [ ] `pkg/core/capability.go` - Capability type
- [ ] `pkg/core/context.go` - Execution context + call chain
- [ ] `pkg/core/schema.go` - JSON Schema helpers
- [ ] `pkg/storage/` - Store interface + memory impl
- [ ] `pkg/agent/` - Agent + Builder
- [ ] `pkg/provider/` - Anthropic + OpenAI
- [ ] Tests unitarios

### Fase 2: Mesh Runtime (Semanas 3-4)
- [ ] `pkg/mesh/mesh.go` - Mesh principal
- [ ] `pkg/mesh/cycle.go` - **Detección de ciclos**
- [ ] `pkg/mesh/timeout.go` - Timeout propagation
- [ ] `pkg/mesh/resolver.go` - Resolver
- [ ] `pkg/mesh/injector.go` - **Inyección de Tools**
- [ ] `pkg/mesh/balancer.go` - Load balancing
- [ ] `pkg/registry/local.go` - Registry in-memory
- [ ] Tests de integración

### Fase 3: Protocolos (Semanas 5-6)
- [ ] `pkg/protocol/trace.go` - W3C Trace Context
- [ ] `pkg/protocol/a2a/` - Completo
- [ ] `pkg/protocol/mcp/` - Server + Client
- [ ] `pkg/protocol/grpc/` - Interno

### Fase 4: Security & Storage (Semanas 7-8)
- [ ] `pkg/security/apikey.go` - **Constant-time compare**
- [ ] `pkg/security/jwt.go`
- [ ] `pkg/security/mtls.go`
- [ ] `pkg/storage/redis.go`
- [ ] `pkg/registry/consul.go`

### Fase 5: Patterns (Semanas 9-10)
- [ ] Patterns agénticos
- [ ] Tests

### Fase 6: Production (Semanas 11-12)
- [ ] OpenTelemetry completo
- [ ] CLI tool
- [ ] Documentación
- [ ] Ejemplos

---

## 11. Principios de Diseño

1. **Zero Config Start**: Funciona sin configuración
2. **Progressive Disclosure**: Simple por defecto, configurable si necesitas
3. **Fail Fast**: Errores claros en tiempo de compilación
4. **Observable by Default**: Logging, tracing, métricas incluido
5. **Protocol Agnostic**: Mismo código, diferentes protocolos
6. **Test Friendly**: Interfaces para mock fácil
7. **No Import Cycles**: Interfaces en `pkg/core/`
8. **Distributed First**: Estado en Store externo
9. **Secure by Default**: Auth + timing-safe operations
10. **Cycle Safe**: Detección y prevención de loops

---

## Autor

VoidLab - Chile 🇨🇱

---
