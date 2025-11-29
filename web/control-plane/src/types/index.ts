export interface Stats {
  uptime: string;
  go_version: string;
  num_goroutine: number;
  memory: {
    alloc: number;
    total_alloc: number;
    sys: number;
    num_gc: number;
  };
}

export interface ModelRewrite {
  model_exact?: string;
  model_prefix?: string;
  provider: string;
  model: string;
}

export interface ModelRouting {
  prefix_providers?: Record<string, string>;
  rewrites?: ModelRewrite[];
}

export interface AppEntry {
  name?: string;
  frontdoor?: string;
  path: string;
  provider?: string;
  default_model?: string;
  enable_responses?: boolean;
  model_routing?: ModelRouting;
}

export interface ProviderEntry {
  name: string;
  type: string;
  base_url?: string;
  supports_responses: boolean;
  enable_passthrough: boolean;
}

export interface RoutingRule {
  model_prefix?: string;
  model_exact?: string;
  provider: string;
}

export interface TenantEntry {
  id: string;
  name: string;
  provider_count: number;
  routing_rules: number;
  supports_tenant: boolean;
}

export interface Overview {
  mode: string;
  storage: { enabled: boolean; type: string; path?: string };
  apps?: AppEntry[];
  frontdoors?: { type: string; path: string; provider?: string; default_model?: string }[];
  providers: ProviderEntry[];
  routing: { default_provider: string; rules: RoutingRule[] };
  tenants: TenantEntry[];
}

export interface ResponseSummary {
  id: string;
  status: string;
  model: string;
  previous_response_id?: string;
  metadata?: Record<string, string>;
  created_at: number;
  updated_at: number;
}

export interface ResponseDetail {
  id: string;
  status: string;
  model: string;
  request?: unknown;
  response?: unknown;
  previous_response_id?: string;
  metadata?: Record<string, string>;
  created_at: number;
  updated_at: number;
}

export interface ThreadSummary {
  id: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
  message_count: number;
}

export interface ThreadMessage {
  id: string;
  role: string;
  content: string;
  created_at: number;
}

export interface ThreadDetail {
  id: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
  messages: ThreadMessage[];
}

// Unified interaction types
export interface InteractionSummary {
  id: string;
  type: 'conversation' | 'response' | 'interaction';
  status?: string;
  model?: string;
  metadata?: Record<string, string>;
  message_count?: number;
  previous_response_id?: string;
  created_at: number;
  updated_at: number;
}

export interface InteractionDetail {
  id: string;
  type: 'conversation' | 'response';
  status?: string;
  model?: string;
  metadata?: Record<string, string>;
  previous_response_id?: string;
  created_at: number;
  updated_at: number;
  // For conversations
  messages?: ThreadMessage[];
  // For responses
  request?: unknown;
  response?: ResponseData;
}

// Responses API output types
export interface ResponseData {
  id?: string;
  status?: string;
  model?: string;
  output?: ResponseOutputItem[];
  usage?: {
    input_tokens?: number;
    output_tokens?: number;
    total_tokens?: number;
  };
  [key: string]: unknown;
}

export interface ResponseOutputItem {
  type: 'message' | 'function_call' | 'function_call_output' | 'file';
  id?: string;
  role?: string;
  status?: string;
  content?: ResponseContentPart[];
  // For function_call
  name?: string;
  call_id?: string;
  arguments?: string;
  // For function_call_output
  output?: string;
}

export interface ResponseContentPart {
  type: string;
  text?: string;
  [key: string]: unknown;
}

// Unified Interaction Types (New)
export interface InteractionRequestView {
  raw?: unknown;
  canonical?: unknown;
  unmapped_fields?: string[];
  provider_request?: unknown;
}

export interface InteractionResponseView {
  raw?: unknown;
  canonical?: unknown;
  unmapped_fields?: string[];
  client_response?: unknown;
  finish_reason?: string;
  usage?: {
    input_tokens?: number;
    output_tokens?: number;
    total_tokens?: number;
  };
}

export interface InteractionErrorView {
  type: string;
  code?: string;
  message: string;
}

export interface NewInteractionDetail {
  id: string;
  tenant_id: string;
  frontdoor: string;
  provider: string;
  app_name?: string;
  requested_model: string;
  served_model?: string;
  provider_model?: string;
  streaming: boolean;
  status: string;
  duration: string;
  duration_ns: number;
  metadata?: Record<string, string>;
  request_headers?: Record<string, string>;
  created_at: number;
  updated_at: number;

  request?: InteractionRequestView;
  response?: InteractionResponseView;
  error?: InteractionErrorView;

  // Legacy fields for compatibility if needed, though we should prefer the new structure
  type?: 'interaction';
}

export type InteractionDetailUnion = InteractionDetail | NewInteractionDetail;
