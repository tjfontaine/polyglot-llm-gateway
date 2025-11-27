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
  type: 'conversation' | 'response';
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
  response?: unknown;
}
