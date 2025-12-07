/**
 * GraphQL schema definition for the gateway control plane.
 *
 * This schema is designed to match the Go implementation.
 *
 * @module graphql/schema
 */

// The GraphQL schema as a string
export const typeDefs = `#graphql
# Gateway Control Plane GraphQL Schema

scalar Time
scalar JSON
scalar Int64

# =============================================================================
# Query Root
# =============================================================================

type Query {
  """Runtime statistics for the gateway"""
  stats: Stats!
  
  """Gateway configuration overview"""
  overview: Overview!
  
  """List interactions with optional filtering"""
  interactions(
    filter: InteractionFilter
    limit: Int = 50
    offset: Int = 0
  ): InteractionConnection!
  
  """Get a single interaction by ID"""
  interaction(id: ID!): Interaction
  
  """Get events for an interaction"""
  interactionEvents(interactionId: ID!, limit: Int = 500): InteractionEventsResponse!
  
  """Get shadow results for an interaction"""
  shadowResults(interactionId: ID!): ShadowResultsResponse!
  
  """List interactions with divergent shadow results"""
  divergentShadows(
    limit: Int = 100
    offset: Int = 0
    provider: String
  ): DivergentShadowsResponse!
  
  """Get a single shadow result by ID"""
  shadow(id: ID!): ShadowResult
}

# =============================================================================
# Stats Types
# =============================================================================

type Stats {
  uptime: String!
  runtime: String!
  memory: MemoryStats
}

type MemoryStats {
  heapUsed: Int64
  heapTotal: Int64
  external: Int64
  rss: Int64
}

# =============================================================================
# Overview Types
# =============================================================================

type Overview {
  mode: String!
  storage: StorageSummary!
  apps: [AppSummary!]!
  frontdoors: [FrontdoorSummary!]!
  providers: [ProviderSummary!]!
  routing: RoutingSummary!
}

type StorageSummary {
  enabled: Boolean!
  type: String!
}

type AppSummary {
  name: String!
  frontdoor: String!
  path: String!
  provider: String
  defaultModel: String
}

type FrontdoorSummary {
  type: String!
  path: String!
}

type ProviderSummary {
  name: String!
  type: String!
  baseUrl: String
}

type RoutingSummary {
  defaultProvider: String
  rules: [RoutingRule!]!
}

type RoutingRule {
  modelPrefix: String
  modelExact: String
  provider: String!
}

# =============================================================================
# Interaction Types
# =============================================================================

input InteractionFilter {
  """Filter by frontdoor type (openai, anthropic, responses)"""
  frontdoor: String
  """Filter by provider name"""
  provider: String
  """Filter by status"""
  status: String
}

type InteractionConnection {
  interactions: [InteractionSummary!]!
  total: Int!
}

type InteractionSummary {
  id: ID!
  type: String!
  status: String
  model: String
  provider: String
  durationMs: Int
  createdAt: Int64!
  updatedAt: Int64!
}

type Interaction {
  id: ID!
  tenantId: String!
  frontdoor: String!
  provider: String!
  appName: String
  requestedModel: String
  servedModel: String
  streaming: Boolean!
  status: String!
  durationMs: Int
  createdAt: Int64!
  updatedAt: Int64!
  
  request: InteractionRequest
  response: InteractionResponse
  error: InteractionError
  transformationSteps: [TransformationStep!]
}

type InteractionRequest {
  raw: JSON
  canonical: JSON
  unmappedFields: [String!]
  providerRequest: JSON
}

type InteractionResponse {
  raw: JSON
  canonical: JSON
  unmappedFields: [String!]
  clientResponse: JSON
  finishReason: String
  usage: Usage
}

type InteractionError {
  type: String!
  code: String
  message: String!
}

type TransformationStep {
  stage: String!
  description: String!
  codec: String
  details: JSON
  warnings: [String!]
}

type Usage {
  promptTokens: Int
  completionTokens: Int
  totalTokens: Int
}

# =============================================================================
# Interaction Events
# =============================================================================

type InteractionEventsResponse {
  interactionId: ID!
  events: [InteractionEvent!]!
}

type InteractionEvent {
  id: ID!
  interactionId: ID!
  type: String!
  timestamp: String!
  payload: JSON
}

# =============================================================================
# Shadow Mode Types
# =============================================================================

type ShadowResultsResponse {
  interactionId: ID!
  shadows: [ShadowResult!]!
}

type ShadowResult {
  id: ID!
  interactionId: ID!
  providerName: String!
  providerModel: String
  durationMs: Int
  tokensIn: Int
  tokensOut: Int
  divergences: [Divergence!]
  hasDivergence: Boolean!
  createdAt: Int64!
}

type Divergence {
  type: String!
  path: String!
  description: String!
  primary: JSON
  shadow: JSON
}

type DivergentShadowsResponse {
  interactions: [InteractionSummary!]!
  total: Int!
  limit: Int!
  offset: Int!
}
`;

/**
 * GraphQL types for TypeScript.
 */
export interface GraphQLStats {
    uptime: string;
    runtime: string;
    memory?: {
        heapUsed?: number;
        heapTotal?: number;
        external?: number;
        rss?: number;
    };
}

export interface GraphQLOverview {
    mode: string;
    storage: {
        enabled: boolean;
        type: string;
    };
    apps: Array<{
        name: string;
        frontdoor: string;
        path: string;
        provider?: string;
        defaultModel?: string;
    }>;
    frontdoors: Array<{
        type: string;
        path: string;
    }>;
    providers: Array<{
        name: string;
        type: string;
        baseUrl?: string;
    }>;
    routing: {
        defaultProvider?: string;
        rules: Array<{
            modelPrefix?: string;
            modelExact?: string;
            provider: string;
        }>;
    };
}

export interface GraphQLInteractionSummary {
    id: string;
    type: string;
    status?: string;
    model?: string;
    provider?: string;
    durationMs?: number;
    createdAt: number;
    updatedAt: number;
}

export interface GraphQLInteractionConnection {
    interactions: GraphQLInteractionSummary[];
    total: number;
}

export interface GraphQLInteractionFilter {
    frontdoor?: string;
    provider?: string;
    status?: string;
}
