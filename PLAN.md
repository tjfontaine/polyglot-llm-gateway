# Shadow Mode Implementation Plan

## Overview

Shadow mode enables per-app comparison of how requests would perform against an alternate provider. For each request, the gateway executes both the primary provider and one or more shadow providers **in parallel**, storing the complete transformation pipeline for both. The control plane UI provides side-by-side comparison with diff views, highlighting structural divergences (missing fields, different formats) rather than expected content differences.

## Goals

1. **Full Pipeline Visibility**: Capture the entire transformation flow for shadow requests—raw request → canonical → provider request → provider response → canonical response → frontdoor-encoded client response.
2. **Zero Impact on Primary Path**: Shadow execution runs in parallel but never blocks or affects the primary response. Shadow failures are logged and stored, not propagated.
3. **Side-by-Side Comparison UI**: Control plane shows primary vs shadow responses with structural diff highlighting.
4. **Divergence Detection**: Flag requests where the shadow response differs structurally (missing fields, different types) beyond expected assistant message content differences.

---

## Configuration Schema

### App-Level Shadow Configuration

\`\`\`yaml
apps:
  - name: prod-api
    frontdoor: anthropic
    path: /anthropic
    provider: openai-gpt4o
    shadow:
      enabled: true
      providers:
        - name: anthropic-claude
          # Optional: override model for shadow
          model: claude-sonnet-4-20250514
        - name: openai-gpt4o-mini
      # Optional: timeout for shadow requests (default: 30s)
      timeout: 30s
      # Optional: whether to store streaming chunks or just final result
      store_stream_chunks: false
\`\`\`

### Config Struct Changes

\`\`\`go
// internal/pkg/config/config.go

type AppConfig struct {
    Name            string             \`koanf:"name"\`
    Frontdoor       string             \`koanf:"frontdoor"\`
    Path            string             \`koanf:"path"\`
    Provider        string             \`koanf:"provider"\`
    DefaultModel    string             \`koanf:"default_model"\`
    ModelRouting    ModelRoutingConfig \`koanf:"model_routing"\`
    Models          []ModelListItem    \`koanf:"models"\`
    EnableResponses bool               \`koanf:"enable_responses"\`
    Shadow          ShadowConfig       \`koanf:"shadow"\`  // NEW
}

type ShadowConfig struct {
    Enabled           bool                  \`koanf:"enabled"\`
    Providers         []ShadowProviderConfig \`koanf:"providers"\`
    Timeout           time.Duration         \`koanf:"timeout"\`
    StoreStreamChunks bool                  \`koanf:"store_stream_chunks"\`
}

type ShadowProviderConfig struct {
    Name  string \`koanf:"name"\`
    Model string \`koanf:"model"\` // Optional: override model for this shadow
}
\`\`\`

---

## Domain Model

### Shadow Result Type

\`\`\`go
// internal/core/domain/shadow.go

// ShadowResult captures the full pipeline execution for a shadow provider.
type ShadowResult struct {
    ID              string            \`json:"id"\`
    InteractionID   string            \`json:"interaction_id"\`
    ProviderName    string            \`json:"provider_name"\`
    ProviderModel   string            \`json:"provider_model,omitempty"\`
    
    // Pipeline stages (mirrors primary interaction structure)
    Request         *ShadowRequest    \`json:"request"\`
    Response        *ShadowResponse   \`json:"response,omitempty"\`
    Error           *InteractionError \`json:"error,omitempty"\`
    
    // Metrics
    Duration        time.Duration     \`json:"duration_ns"\`
    TokensIn        int               \`json:"tokens_in,omitempty"\`
    TokensOut       int               \`json:"tokens_out,omitempty"\`
    
    // Comparison metadata
    Divergences     []Divergence      \`json:"divergences,omitempty"\`
    
    CreatedAt       time.Time         \`json:"created_at"\`
}

// ShadowRequest captures the shadow request transformation.
type ShadowRequest struct {
    // Canonical request (same as primary, may have model overridden)
    Canonical       json.RawMessage \`json:"canonical,omitempty"\`
    // Provider-specific request sent to shadow provider
    ProviderRequest json.RawMessage \`json:"provider_request,omitempty"\`
}

// ShadowResponse captures the shadow response transformation.
type ShadowResponse struct {
    // Raw response from shadow provider
    Raw             json.RawMessage \`json:"raw,omitempty"\`
    // Canonical translation of shadow response
    Canonical       json.RawMessage \`json:"canonical,omitempty"\`
    // What the client WOULD have received (re-encoded via frontdoor codec)
    ClientResponse  json.RawMessage \`json:"client_response,omitempty"\`
    // Finish reason from shadow
    FinishReason    string          \`json:"finish_reason,omitempty"\`
    // Usage from shadow
    Usage           *Usage          \`json:"usage,omitempty"\`
}

// Divergence describes a structural difference between primary and shadow.
type Divergence struct {
    Type        DivergenceType \`json:"type"\`
    Path        string         \`json:"path"\`        // JSON path where divergence occurred
    Description string         \`json:"description"\`
    Primary     interface{}    \`json:"primary,omitempty"\`
    Shadow      interface{}    \`json:"shadow,omitempty"\`
}

type DivergenceType string

const (
    DivergenceMissingField   DivergenceType = "missing_field"
    DivergenceExtraField     DivergenceType = "extra_field"
    DivergenceTypeMismatch   DivergenceType = "type_mismatch"
    DivergenceArrayLength    DivergenceType = "array_length"
    DivergenceNullMismatch   DivergenceType = "null_mismatch"
)
\`\`\`

---

## Storage Schema

### New Table: \`shadow_results\`

\`\`\`sql
CREATE TABLE IF NOT EXISTS shadow_results (
    id TEXT PRIMARY KEY,
    interaction_id TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    provider_model TEXT,
    
    -- Request pipeline
    request_canonical TEXT,
    request_provider TEXT,
    
    -- Response pipeline
    response_raw TEXT,
    response_canonical TEXT,
    response_client TEXT,
    response_finish_reason TEXT,
    response_usage TEXT,  -- JSON: {"input": N, "output": M}
    
    -- Error (if shadow failed)
    error_type TEXT,
    error_code TEXT,
    error_message TEXT,
    
    -- Metrics
    duration_ns INTEGER,
    tokens_in INTEGER,
    tokens_out INTEGER,
    
    -- Divergence analysis
    divergences TEXT,  -- JSON array of Divergence objects
    has_structural_divergence INTEGER DEFAULT 0,  -- Quick filter flag
    
    created_at DATETIME NOT NULL,
    
    FOREIGN KEY (interaction_id) REFERENCES interactions(id)
);

CREATE INDEX idx_shadow_results_interaction ON shadow_results(interaction_id);
CREATE INDEX idx_shadow_results_divergence ON shadow_results(has_structural_divergence);
CREATE INDEX idx_shadow_results_provider ON shadow_results(provider_name);
\`\`\`

### Storage Interface Extension

\`\`\`go
// internal/core/ports/storage.go

type ShadowStore interface {
    SaveShadowResult(ctx context.Context, result *domain.ShadowResult) error
    GetShadowResults(ctx context.Context, interactionID string) ([]*domain.ShadowResult, error)
    ListDivergentInteractions(ctx context.Context, opts *DivergenceListOptions) ([]*domain.InteractionSummary, error)
}

type DivergenceListOptions struct {
    Limit  int
    Offset int
    // Only return interactions with structural divergences
    OnlyDivergent bool
    // Filter by divergence type
    DivergenceTypes []domain.DivergenceType
}
\`\`\`

---

## Execution Architecture

### Shadow Executor

\`\`\`go
// internal/shadow/executor.go

// Executor runs shadow requests in parallel with primary execution.
type Executor struct {
    providers       map[string]ports.Provider
    store           ports.ShadowStore
    frontdoorCodecs map[domain.APIType]codec.Codec
    logger          *slog.Logger
}

// ExecuteParams contains everything needed to run shadow requests.
type ExecuteParams struct {
    InteractionID   string
    FrontdoorType   domain.APIType
    CanonicalReq    *domain.CanonicalRequest
    PrimaryResponse *domain.CanonicalResponse  // For comparison after primary completes
    ShadowConfigs   []config.ShadowProviderConfig
    Timeout         time.Duration
}

// Execute runs all configured shadow providers in parallel.
// This should be called in a goroutine after the primary response starts.
// It waits for all shadows to complete (or timeout) before returning.
func (e *Executor) Execute(ctx context.Context, params ExecuteParams) {
    var wg sync.WaitGroup
    
    for _, shadowCfg := range params.ShadowConfigs {
        wg.Add(1)
        go func(cfg config.ShadowProviderConfig) {
            defer wg.Done()
            e.executeShadow(ctx, params, cfg)
        }(shadowCfg)
    }
    
    wg.Wait()
}

func (e *Executor) executeShadow(ctx context.Context, params ExecuteParams, cfg config.ShadowProviderConfig) {
    startTime := time.Now()
    result := &domain.ShadowResult{
        ID:            uuid.NewString(),
        InteractionID: params.InteractionID,
        ProviderName:  cfg.Name,
        CreatedAt:     startTime,
    }
    
    defer func() {
        result.Duration = time.Since(startTime)
        if err := e.store.SaveShadowResult(ctx, result); err != nil {
            e.logger.Error("failed to save shadow result",
                slog.String("interaction_id", params.InteractionID),
                slog.String("shadow_provider", cfg.Name),
                slog.String("error", err.Error()),
            )
        }
    }()
    
    provider, ok := e.providers[cfg.Name]
    if !ok {
        result.Error = &domain.InteractionError{
            Type:    "configuration_error",
            Message: fmt.Sprintf("shadow provider %q not found", cfg.Name),
        }
        return
    }
    
    // Clone canonical request with optional model override
    shadowReq := params.CanonicalReq.Clone()
    if cfg.Model != "" {
        shadowReq.Model = cfg.Model
        result.ProviderModel = cfg.Model
    }
    
    // Capture canonical request
    canonJSON, _ := json.Marshal(shadowReq)
    result.Request = &domain.ShadowRequest{
        Canonical: canonJSON,
    }
    
    // Execute shadow request (with timeout from context)
    var shadowResp *domain.CanonicalResponse
    var err error
    
    if shadowReq.Stream {
        shadowResp, err = e.executeStreamingShadow(ctx, provider, shadowReq)
    } else {
        shadowResp, err = provider.Complete(ctx, shadowReq)
    }
    
    if err != nil {
        result.Error = &domain.InteractionError{
            Type:    "provider_error",
            Message: err.Error(),
        }
        return
    }
    
    // Capture response pipeline
    result.Response = &domain.ShadowResponse{}
    
    if len(shadowResp.RawResponse) > 0 {
        result.Response.Raw = shadowResp.RawResponse
    }
    
    canonRespJSON, _ := json.Marshal(shadowResp)
    result.Response.Canonical = canonRespJSON
    
    if shadowResp.Usage != nil {
        result.Response.Usage = shadowResp.Usage
        result.TokensIn = shadowResp.Usage.PromptTokens
        result.TokensOut = shadowResp.Usage.CompletionTokens
    }
    
    // Re-encode through frontdoor codec to get "what client would see"
    frontdoorCodec := e.frontdoorCodecs[params.FrontdoorType]
    if frontdoorCodec != nil {
        clientBytes, err := frontdoorCodec.EncodeResponse(shadowResp)
        if err == nil {
            result.Response.ClientResponse = clientBytes
        }
    }
    
    // Compute divergences against primary (if available)
    if params.PrimaryResponse != nil {
        result.Divergences = e.computeDivergences(params.PrimaryResponse, shadowResp)
    }
}
\`\`\`

### Divergence Detection

\`\`\`go
// internal/shadow/divergence.go

// computeDivergences compares primary and shadow responses for structural differences.
// It ignores expected content differences (assistant message text) and focuses on:
// - Missing/extra fields
// - Type mismatches
// - Array length differences (for tool calls, etc.)
// - Null vs non-null mismatches
func (e *Executor) computeDivergences(primary, shadow *domain.CanonicalResponse) []domain.Divergence {
    var divergences []domain.Divergence
    
    // Compare at canonical level for structural analysis
    divergences = append(divergences, 
        compareStructure("", toMap(primary), toMap(shadow), ignorePaths)...)
    
    return divergences
}

// Paths to ignore for content comparison (these will differ by design)
var ignorePaths = map[string]bool{
    "id":                          true,
    "choices.*.message.content":   true,
    "choices.*.delta.content":     true,
    "created":                     true,
    "system_fingerprint":          true,
}

func compareStructure(path string, primary, shadow map[string]interface{}, ignore map[string]bool) []domain.Divergence {
    var divergences []domain.Divergence
    
    // Check for fields in primary missing from shadow
    for key, pVal := range primary {
        fieldPath := joinPath(path, key)
        if shouldIgnore(fieldPath, ignore) {
            continue
        }
        
        sVal, exists := shadow[key]
        if !exists {
            divergences = append(divergences, domain.Divergence{
                Type:        domain.DivergenceMissingField,
                Path:        fieldPath,
                Description: fmt.Sprintf("field %q present in primary but missing in shadow", key),
                Primary:     pVal,
            })
            continue
        }
        
        // Recurse for nested objects
        if pMap, ok := pVal.(map[string]interface{}); ok {
            if sMap, ok := sVal.(map[string]interface{}); ok {
                divergences = append(divergences, compareStructure(fieldPath, pMap, sMap, ignore)...)
            } else {
                divergences = append(divergences, domain.Divergence{
                    Type:        domain.DivergenceTypeMismatch,
                    Path:        fieldPath,
                    Description: fmt.Sprintf("type mismatch: primary is object, shadow is %T", sVal),
                    Primary:     pVal,
                    Shadow:      sVal,
                })
            }
        }
        
        // Check array lengths
        if pArr, ok := pVal.([]interface{}); ok {
            if sArr, ok := sVal.([]interface{}); ok {
                if len(pArr) != len(sArr) {
                    divergences = append(divergences, domain.Divergence{
                        Type:        domain.DivergenceArrayLength,
                        Path:        fieldPath,
                        Description: fmt.Sprintf("array length mismatch: primary=%d, shadow=%d", len(pArr), len(sArr)),
                        Primary:     len(pArr),
                        Shadow:      len(sArr),
                    })
                }
            }
        }
        
        // Check null mismatches
        if pVal == nil && sVal != nil {
            divergences = append(divergences, domain.Divergence{
                Type:        domain.DivergenceNullMismatch,
                Path:        fieldPath,
                Description: "primary is null, shadow is not",
                Shadow:      sVal,
            })
        } else if pVal != nil && sVal == nil {
            divergences = append(divergences, domain.Divergence{
                Type:        domain.DivergenceNullMismatch,
                Path:        fieldPath,
                Description: "primary is not null, shadow is null",
                Primary:     pVal,
            })
        }
    }
    
    // Check for extra fields in shadow
    for key, sVal := range shadow {
        fieldPath := joinPath(path, key)
        if shouldIgnore(fieldPath, ignore) {
            continue
        }
        if _, exists := primary[key]; !exists {
            divergences = append(divergences, domain.Divergence{
                Type:        domain.DivergenceExtraField,
                Path:        fieldPath,
                Description: fmt.Sprintf("field %q present in shadow but missing in primary", key),
                Shadow:      sVal,
            })
        }
    }
    
    return divergences
}
\`\`\`

---

## Integration Points

### Handler Integration

The shadow executor is invoked from frontdoor handlers after the primary request begins (for streaming) or completes (for non-streaming).

\`\`\`go
// internal/api/anthropic/handler.go (modified HandleMessages)

func (h *FrontdoorHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
    // ... existing code through canonical request creation ...
    
    // Check if shadow mode is enabled for this app
    shadowCfg := h.getShadowConfig()  // From app config
    var shadowResultChan chan struct{}
    
    if shadowCfg != nil && shadowCfg.Enabled {
        shadowResultChan = make(chan struct{})
        
        // Create shadow context with configured timeout
        shadowCtx, shadowCancel := context.WithTimeout(
            context.Background(),  // Independent of request context
            shadowCfg.Timeout,
        )
        
        go func() {
            defer shadowCancel()
            defer close(shadowResultChan)
            
            // Execute shadows - they will wait for primary response
            // before computing divergences
            h.shadowExecutor.Execute(shadowCtx, shadow.ExecuteParams{
                InteractionID:   interactionID,
                FrontdoorType:   domain.APITypeAnthropic,
                CanonicalReq:    canonReq,
                PrimaryResponse: nil,  // Will be populated via channel after primary completes
                ShadowConfigs:   shadowCfg.Providers,
                Timeout:         shadowCfg.Timeout,
            })
        }()
    }
    
    // ... existing primary request execution ...
    
    // After primary completes, notify shadow executor of primary response
    // (for divergence computation)
    if shadowCfg != nil && shadowCfg.Enabled && primaryResponse != nil {
        h.shadowExecutor.SetPrimaryResponse(interactionID, primaryResponse)
    }
    
    // ... rest of handler ...
}
\`\`\`

### Architecture Decision: Parallel with Deferred Comparison

Since shadows run in parallel with primary:

1. **Shadow starts immediately** when canonical request is available
2. **Shadow executes independently** against its provider
3. **Primary response is captured** when it completes
4. **Divergence computation** happens after both complete (shadow waits if needed)
5. **Shadow result is stored** with divergences

This ensures:
- No latency impact on primary path (shadow runs in background)
- Full pipeline capture for both sides
- Accurate divergence comparison

---

## Control Plane API

### New Endpoints

\`\`\`go
// GET /admin/api/interactions/{id}/shadows
// Returns all shadow results for an interaction
type ShadowResultsResponse struct {
    Shadows []ShadowResultView \`json:"shadows"\`
}

type ShadowResultView struct {
    ID              string                 \`json:"id"\`
    ProviderName    string                 \`json:"provider_name"\`
    ProviderModel   string                 \`json:"provider_model,omitempty"\`
    Duration        int64                  \`json:"duration_ms"\`
    TokensIn        int                    \`json:"tokens_in,omitempty"\`
    TokensOut       int                    \`json:"tokens_out,omitempty"\`
    Status          string                 \`json:"status"\`  // "success" | "error"
    Error           *InteractionErrorView  \`json:"error,omitempty"\`
    Divergences     []DivergenceView       \`json:"divergences,omitempty"\`
    HasDivergences  bool                   \`json:"has_divergences"\`
}

// GET /admin/api/interactions/{id}/compare/{shadowId}
// Returns side-by-side comparison data
type ComparisonResponse struct {
    Primary PipelineView   \`json:"primary"\`
    Shadow  PipelineView   \`json:"shadow"\`
    Diff    DiffView       \`json:"diff"\`
}

type PipelineView struct {
    Request struct {
        Raw       json.RawMessage \`json:"raw,omitempty"\`
        Canonical json.RawMessage \`json:"canonical,omitempty"\`
        Provider  json.RawMessage \`json:"provider,omitempty"\`
    } \`json:"request"\`
    Response struct {
        Raw           json.RawMessage \`json:"raw,omitempty"\`
        Canonical     json.RawMessage \`json:"canonical,omitempty"\`
        ClientEncoded json.RawMessage \`json:"client_encoded,omitempty"\`
    } \`json:"response"\`
    Metrics struct {
        DurationMs int \`json:"duration_ms"\`
        TokensIn   int \`json:"tokens_in"\`
        TokensOut  int \`json:"tokens_out"\`
    } \`json:"metrics"\`
}

type DiffView struct {
    Divergences     []DivergenceView \`json:"divergences"\`
    // Unified diff of client-encoded responses (for text comparison)
    ClientResponseDiff string \`json:"client_response_diff,omitempty"\`
}

// GET /admin/api/divergences
// List interactions with structural divergences
type DivergenceListResponse struct {
    Interactions []InteractionWithDivergences \`json:"interactions"\`
    Total        int                          \`json:"total"\`
}

type InteractionWithDivergences struct {
    Interaction     InteractionSummaryView \`json:"interaction"\`
    ShadowCount     int                    \`json:"shadow_count"\`
    DivergenceCount int                    \`json:"divergence_count"\`
    DivergenceTypes []string               \`json:"divergence_types"\`
}
\`\`\`

---

## Frontend Components

### Shadow Comparison View

\`\`\`
src/
├── pages/
│   ├── InteractionDetail.tsx    # Updated to show shadow tab
│   └── Divergences.tsx          # New page listing divergent interactions
├── components/
│   ├── shadow/
│   │   ├── ShadowSummary.tsx    # Summary cards for each shadow result
│   │   ├── ShadowComparison.tsx # Side-by-side pipeline comparison
│   │   ├── DivergenceList.tsx   # List of structural divergences
│   │   ├── PipelineStage.tsx    # Collapsible view of a pipeline stage
│   │   └── JsonDiff.tsx         # JSON diff viewer with highlighting
│   └── ...
\`\`\`

### Interaction Detail Enhancement

\`\`\`tsx
// pages/InteractionDetail.tsx

function InteractionDetail({ id }: { id: string }) {
    const { data: interaction } = useApi(\`/api/interactions/\${id}\`);
    const { data: shadows } = useApi(\`/api/interactions/\${id}/shadows\`);
    
    return (
        <Layout>
            <Tabs defaultValue="primary">
                <TabList>
                    <Tab value="primary">Primary Response</Tab>
                    <Tab value="pipeline">Pipeline</Tab>
                    {shadows?.shadows?.length > 0 && (
                        <Tab value="shadows">
                            Shadows ({shadows.shadows.length})
                            {shadows.shadows.some(s => s.has_divergences) && (
                                <Badge variant="warning">Divergent</Badge>
                            )}
                        </Tab>
                    )}
                </TabList>
                
                <TabPanel value="primary">
                    <PrimaryResponseView interaction={interaction} />
                </TabPanel>
                
                <TabPanel value="pipeline">
                    <PipelineView interaction={interaction} />
                </TabPanel>
                
                <TabPanel value="shadows">
                    <ShadowsView 
                        interactionId={id} 
                        shadows={shadows?.shadows} 
                    />
                </TabPanel>
            </Tabs>
        </Layout>
    );
}
\`\`\`

### Side-by-Side Comparison

\`\`\`tsx
// components/shadow/ShadowComparison.tsx

function ShadowComparison({ interactionId, shadowId }: Props) {
    const { data } = useApi(\`/api/interactions/\${interactionId}/compare/\${shadowId}\`);
    const [viewMode, setViewMode] = useState<'side-by-side' | 'diff'>('side-by-side');
    const [stage, setStage] = useState<'request' | 'response'>('response');
    
    return (
        <div>
            <div className="flex justify-between mb-4">
                <SegmentedControl 
                    value={stage} 
                    onChange={setStage}
                    options={[
                        { value: 'request', label: 'Request Pipeline' },
                        { value: 'response', label: 'Response Pipeline' },
                    ]}
                />
                <SegmentedControl
                    value={viewMode}
                    onChange={setViewMode}
                    options={[
                        { value: 'side-by-side', label: 'Side by Side' },
                        { value: 'diff', label: 'Diff View' },
                    ]}
                />
            </div>
            
            {/* Metrics comparison bar */}
            <MetricsComparison 
                primary={data?.primary.metrics}
                shadow={data?.shadow.metrics}
            />
            
            {/* Divergence alerts */}
            {data?.diff.divergences?.length > 0 && (
                <DivergenceAlert divergences={data.diff.divergences} />
            )}
            
            {/* Main comparison view */}
            {viewMode === 'side-by-side' ? (
                <SideBySideView
                    primary={data?.primary[stage]}
                    shadow={data?.shadow[stage]}
                    stage={stage}
                />
            ) : (
                <JsonDiff
                    left={data?.primary[stage]}
                    right={data?.shadow[stage]}
                    leftLabel="Primary"
                    rightLabel="Shadow"
                />
            )}
        </div>
    );
}
\`\`\`

### Divergence Highlighting

\`\`\`tsx
// components/shadow/DivergenceList.tsx

function DivergenceList({ divergences }: { divergences: Divergence[] }) {
    const grouped = groupBy(divergences, d => d.type);
    
    return (
        <div className="space-y-4">
            {Object.entries(grouped).map(([type, items]) => (
                <div key={type}>
                    <h4 className="font-medium flex items-center gap-2">
                        <DivergenceIcon type={type} />
                        {divergenceTypeLabels[type]}
                        <Badge>{items.length}</Badge>
                    </h4>
                    <ul className="mt-2 space-y-1">
                        {items.map((d, i) => (
                            <li key={i} className="text-sm font-mono">
                                <span className="text-blue-600">{d.path}</span>
                                <span className="text-gray-500 ml-2">{d.description}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            ))}
        </div>
    );
}

const divergenceTypeLabels = {
    missing_field: 'Missing Fields (in shadow)',
    extra_field: 'Extra Fields (in shadow)',
    type_mismatch: 'Type Mismatches',
    array_length: 'Array Length Differences',
    null_mismatch: 'Null/Non-null Mismatches',
};
\`\`\`

---

## Implementation Phases

### Phase 1: Core Infrastructure (Foundation)
**Files to create/modify:**

1. **Config schema** (\`internal/pkg/config/config.go\`)
   - Add \`ShadowConfig\` and \`ShadowProviderConfig\` structs
   - Add \`Shadow\` field to \`AppConfig\`

2. **Domain types** (\`internal/core/domain/shadow.go\`)
   - Create \`ShadowResult\`, \`ShadowRequest\`, \`ShadowResponse\`
   - Create \`Divergence\` and \`DivergenceType\`

3. **Storage** (\`internal/storage/sqlite/shadow_store.go\`)
   - Create table schema
   - Implement \`ShadowStore\` interface

4. **Add Clone method** (\`internal/core/domain/types.go\`)
   - Add \`Clone()\` method to \`CanonicalRequest\`

### Phase 2: Shadow Execution Engine
**Files to create:**

1. **Shadow executor** (\`internal/shadow/executor.go\`)
   - Main execution logic with parallel goroutines
   - Timeout handling
   - Error capture

2. **Divergence detection** (\`internal/shadow/divergence.go\`)
   - Structural comparison logic
   - Path matching and ignore rules

3. **Primary response channel** (\`internal/shadow/coordinator.go\`)
   - Coordination between primary completion and shadow comparison

### Phase 3: Handler Integration
**Files to modify:**

1. **Anthropic handler** (\`internal/api/anthropic/handler.go\`)
   - Inject shadow executor
   - Trigger shadow execution after canonical request ready
   - Notify shadow executor of primary response

2. **OpenAI handler** (\`internal/api/openai/handler.go\`)
   - Same shadow integration

3. **Responses handler** (\`internal/api/responses/handler.go\`)
   - Same shadow integration

4. **Server wiring** (\`internal/api/server/server.go\`)
   - Create and inject shadow executor into handlers

### Phase 4: Control Plane API
**Files to create/modify:**

1. **Shadow endpoints** (\`internal/api/controlplane/shadow.go\`)
   - \`GET /api/interactions/{id}/shadows\`
   - \`GET /api/interactions/{id}/compare/{shadowId}\`
   - \`GET /api/divergences\`

2. **Server routes** (\`internal/api/controlplane/server.go\`)
   - Register new endpoints

### Phase 5: Frontend UI
**Files to create:**

1. **Shadow components** (\`web/control-plane/src/components/shadow/\`)
   - \`ShadowSummary.tsx\`
   - \`ShadowComparison.tsx\`
   - \`DivergenceList.tsx\`
   - \`JsonDiff.tsx\`

2. **Updated pages** (\`web/control-plane/src/pages/\`)
   - Update \`InteractionDetail.tsx\` with shadows tab
   - Create \`Divergences.tsx\` for divergence list view

3. **Types** (\`web/control-plane/src/types/shadow.ts\`)
   - TypeScript interfaces for shadow-related API responses

---

## Testing Strategy

### Unit Tests

1. **Divergence detection** (\`internal/shadow/divergence_test.go\`)
   - Missing fields detected
   - Extra fields detected
   - Type mismatches detected
   - Ignored paths respected
   - Nested object comparison

2. **Shadow executor** (\`internal/shadow/executor_test.go\`)
   - Parallel execution
   - Timeout handling
   - Error capture and storage
   - Model override

3. **Storage** (\`internal/storage/sqlite/shadow_store_test.go\`)
   - Save and retrieve shadow results
   - Query divergent interactions

### Integration Tests

1. **Full pipeline** (\`internal/shadow/integration_test.go\`)
   - End-to-end shadow execution with mock providers
   - Verify all pipeline stages captured
   - Verify divergences computed correctly

2. **Handler integration** (\`internal/api/anthropic/handler_shadow_test.go\`)
   - Shadow triggers correctly
   - Primary response not affected
   - Shadow errors isolated

### Frontend Tests

1. **Shadow components** (\`web/control-plane/src/components/shadow/*.test.tsx\`)
   - Render shadow summary
   - Side-by-side comparison display
   - Diff view toggle
   - Divergence highlighting

---

## Open Questions / Future Enhancements

1. **Streaming shadow capture**: For now, aggregate streaming responses. Future: optionally store chunk-by-chunk timeline.

2. **Shadow sampling**: Current plan is 100% of requests. Future: add sampling configuration.

3. **Aggregate analytics**: Future enhancement to show divergence trends over time.

4. **Shadow for specific models only**: Future: filter which models trigger shadow mode.

5. **Webhook on divergence**: Future: notify external systems when structural divergences detected.

6. **Shadow result retention**: Consider adding TTL/cleanup for shadow results to manage storage growth.

---

## Summary

This plan enables comprehensive A/B comparison between your primary provider and alternative providers:

| Aspect | Approach |
|--------|----------|
| Configuration | Per-app, multiple shadow providers supported |
| Execution | Parallel, independent of primary path |
| Capture | Full pipeline (raw → canonical → provider → response → client-encoded) |
| Comparison | Structural divergence detection with configurable ignore paths |
| Storage | Dedicated \`shadow_results\` table with divergence flags |
| UI | Side-by-side comparison with diff view, divergence list page |
| Impact | Zero impact on primary response; shadow failures logged but isolated |
