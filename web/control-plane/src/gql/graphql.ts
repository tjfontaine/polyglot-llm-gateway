/* eslint-disable */
export type Maybe<T> = T | null;
export type InputMaybe<T> = T | null | undefined;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Int64: { input: any; output: any; }
  JSON: { input: any; output: any; }
  Time: { input: any; output: any; }
};

export type AppSummary = {
  __typename?: 'AppSummary';
  defaultModel?: Maybe<Scalars['String']['output']>;
  enableResponses: Scalars['Boolean']['output'];
  frontdoor: Scalars['String']['output'];
  modelRouting?: Maybe<ModelRoutingSummary>;
  name: Scalars['String']['output'];
  path: Scalars['String']['output'];
  provider?: Maybe<Scalars['String']['output']>;
};

export type Divergence = {
  __typename?: 'Divergence';
  description: Scalars['String']['output'];
  path: Scalars['String']['output'];
  primary?: Maybe<Scalars['JSON']['output']>;
  shadow?: Maybe<Scalars['JSON']['output']>;
  type: DivergenceType;
};

export enum DivergenceType {
  ArrayLength = 'ARRAY_LENGTH',
  ExtraField = 'EXTRA_FIELD',
  MissingField = 'MISSING_FIELD',
  NullMismatch = 'NULL_MISMATCH',
  TypeMismatch = 'TYPE_MISMATCH'
}

export type DivergentShadowsResponse = {
  __typename?: 'DivergentShadowsResponse';
  /** Interactions with divergent shadow results (these are regular interaction summaries) */
  interactions: Array<InteractionSummary>;
  limit: Scalars['Int']['output'];
  offset: Scalars['Int']['output'];
  total: Scalars['Int']['output'];
};

export type FrontdoorSummary = {
  __typename?: 'FrontdoorSummary';
  defaultModel?: Maybe<Scalars['String']['output']>;
  path: Scalars['String']['output'];
  provider?: Maybe<Scalars['String']['output']>;
  type: Scalars['String']['output'];
};

export type Interaction = {
  __typename?: 'Interaction';
  appName?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Int64']['output'];
  duration: Scalars['String']['output'];
  durationNs: Scalars['Int64']['output'];
  error?: Maybe<InteractionError>;
  frontdoor: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  metadata?: Maybe<Scalars['JSON']['output']>;
  provider: Scalars['String']['output'];
  providerModel?: Maybe<Scalars['String']['output']>;
  request?: Maybe<InteractionRequest>;
  requestHeaders?: Maybe<Scalars['JSON']['output']>;
  requestedModel: Scalars['String']['output'];
  response?: Maybe<InteractionResponse>;
  servedModel?: Maybe<Scalars['String']['output']>;
  shadows?: Maybe<Array<ShadowResult>>;
  status: Scalars['String']['output'];
  streaming: Scalars['Boolean']['output'];
  tenantId: Scalars['String']['output'];
  transformationSteps?: Maybe<Array<TransformationStep>>;
  updatedAt: Scalars['Int64']['output'];
};

export type InteractionConnection = {
  __typename?: 'InteractionConnection';
  interactions: Array<InteractionSummary>;
  total: Scalars['Int']['output'];
};

export type InteractionError = {
  __typename?: 'InteractionError';
  code?: Maybe<Scalars['String']['output']>;
  message: Scalars['String']['output'];
  type: Scalars['String']['output'];
};

export type InteractionEvent = {
  __typename?: 'InteractionEvent';
  apiType?: Maybe<Scalars['String']['output']>;
  appName?: Maybe<Scalars['String']['output']>;
  canonical?: Maybe<Scalars['JSON']['output']>;
  createdAt: Scalars['String']['output'];
  direction: Scalars['String']['output'];
  frontdoor?: Maybe<Scalars['String']['output']>;
  headers?: Maybe<Scalars['JSON']['output']>;
  id: Scalars['ID']['output'];
  interactionId: Scalars['ID']['output'];
  metadata?: Maybe<Scalars['JSON']['output']>;
  modelRequested?: Maybe<Scalars['String']['output']>;
  modelServed?: Maybe<Scalars['String']['output']>;
  previousResponseId?: Maybe<Scalars['String']['output']>;
  provider?: Maybe<Scalars['String']['output']>;
  providerModel?: Maybe<Scalars['String']['output']>;
  raw?: Maybe<Scalars['JSON']['output']>;
  stage: Scalars['String']['output'];
  threadKey?: Maybe<Scalars['String']['output']>;
};

export type InteractionEventsResponse = {
  __typename?: 'InteractionEventsResponse';
  events: Array<InteractionEvent>;
  interactionId: Scalars['ID']['output'];
};

export type InteractionFilter = {
  /** Filter by frontdoor type (openai, anthropic, responses) */
  frontdoor?: InputMaybe<Scalars['String']['input']>;
  /** Filter by provider name */
  provider?: InputMaybe<Scalars['String']['input']>;
  /** Filter by status */
  status?: InputMaybe<Scalars['String']['input']>;
};

export type InteractionRequest = {
  __typename?: 'InteractionRequest';
  canonical?: Maybe<Scalars['JSON']['output']>;
  providerRequest?: Maybe<Scalars['JSON']['output']>;
  raw?: Maybe<Scalars['JSON']['output']>;
  unmappedFields?: Maybe<Array<Scalars['String']['output']>>;
};

export type InteractionResponse = {
  __typename?: 'InteractionResponse';
  canonical?: Maybe<Scalars['JSON']['output']>;
  clientResponse?: Maybe<Scalars['JSON']['output']>;
  finishReason?: Maybe<Scalars['String']['output']>;
  raw?: Maybe<Scalars['JSON']['output']>;
  unmappedFields?: Maybe<Array<Scalars['String']['output']>>;
  usage?: Maybe<Usage>;
};

export type InteractionSummary = {
  __typename?: 'InteractionSummary';
  createdAt: Scalars['Int64']['output'];
  id: Scalars['ID']['output'];
  messageCount?: Maybe<Scalars['Int']['output']>;
  metadata?: Maybe<Scalars['JSON']['output']>;
  model?: Maybe<Scalars['String']['output']>;
  previousResponseId?: Maybe<Scalars['String']['output']>;
  status?: Maybe<Scalars['String']['output']>;
  type: Scalars['String']['output'];
  updatedAt: Scalars['Int64']['output'];
};

export type MemoryStats = {
  __typename?: 'MemoryStats';
  alloc: Scalars['Int64']['output'];
  numGC: Scalars['Int']['output'];
  sys: Scalars['Int64']['output'];
  totalAlloc: Scalars['Int64']['output'];
};

export type ModelRewriteSummary = {
  __typename?: 'ModelRewriteSummary';
  model: Scalars['String']['output'];
  modelExact?: Maybe<Scalars['String']['output']>;
  modelPrefix?: Maybe<Scalars['String']['output']>;
  provider: Scalars['String']['output'];
};

export type ModelRoutingSummary = {
  __typename?: 'ModelRoutingSummary';
  prefixProviders?: Maybe<Scalars['JSON']['output']>;
  rewrites?: Maybe<Array<ModelRewriteSummary>>;
};

export type Overview = {
  __typename?: 'Overview';
  apps: Array<AppSummary>;
  frontdoors: Array<FrontdoorSummary>;
  mode: Scalars['String']['output'];
  providers: Array<ProviderSummary>;
  routing: RoutingSummary;
  storage: StorageSummary;
  tenants: Array<TenantSummary>;
};

export type ProviderSummary = {
  __typename?: 'ProviderSummary';
  baseUrl?: Maybe<Scalars['String']['output']>;
  enablePassthrough: Scalars['Boolean']['output'];
  name: Scalars['String']['output'];
  supportsResponses: Scalars['Boolean']['output'];
  type: Scalars['String']['output'];
};

export type Query = {
  __typename?: 'Query';
  /** List interactions with divergent shadow results */
  divergentShadows: DivergentShadowsResponse;
  /** Get a single interaction by ID */
  interaction?: Maybe<Interaction>;
  /** Get events for an interaction */
  interactionEvents: InteractionEventsResponse;
  /** List interactions with optional filtering */
  interactions: InteractionConnection;
  /** Gateway configuration overview */
  overview: Overview;
  /** Get a single shadow result by ID */
  shadow?: Maybe<ShadowResult>;
  /** Get shadow results for an interaction */
  shadowResults: ShadowResultsResponse;
  /** Runtime statistics for the gateway */
  stats: Stats;
};


export type QueryDivergentShadowsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  provider?: InputMaybe<Scalars['String']['input']>;
};


export type QueryInteractionArgs = {
  id: Scalars['ID']['input'];
};


export type QueryInteractionEventsArgs = {
  interactionId: Scalars['ID']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryInteractionsArgs = {
  filter?: InputMaybe<InteractionFilter>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryShadowArgs = {
  id: Scalars['ID']['input'];
};


export type QueryShadowResultsArgs = {
  interactionId: Scalars['ID']['input'];
};

export type RoutingRule = {
  __typename?: 'RoutingRule';
  modelExact?: Maybe<Scalars['String']['output']>;
  modelPrefix?: Maybe<Scalars['String']['output']>;
  provider: Scalars['String']['output'];
};

export type RoutingSummary = {
  __typename?: 'RoutingSummary';
  defaultProvider: Scalars['String']['output'];
  rules: Array<RoutingRule>;
};

export type ShadowError = {
  __typename?: 'ShadowError';
  code?: Maybe<Scalars['String']['output']>;
  message: Scalars['String']['output'];
  type: Scalars['String']['output'];
};

export type ShadowRequest = {
  __typename?: 'ShadowRequest';
  canonical?: Maybe<Scalars['JSON']['output']>;
  providerRequest?: Maybe<Scalars['JSON']['output']>;
};

export type ShadowResponse = {
  __typename?: 'ShadowResponse';
  canonical?: Maybe<Scalars['JSON']['output']>;
  clientResponse?: Maybe<Scalars['JSON']['output']>;
  finishReason?: Maybe<Scalars['String']['output']>;
  raw?: Maybe<Scalars['JSON']['output']>;
  usage?: Maybe<ShadowUsage>;
};

export type ShadowResult = {
  __typename?: 'ShadowResult';
  createdAt: Scalars['Int64']['output'];
  divergences?: Maybe<Array<Divergence>>;
  durationNs: Scalars['Int64']['output'];
  error?: Maybe<ShadowError>;
  hasStructuralDivergence: Scalars['Boolean']['output'];
  id: Scalars['ID']['output'];
  interactionId: Scalars['ID']['output'];
  providerModel?: Maybe<Scalars['String']['output']>;
  providerName: Scalars['String']['output'];
  request?: Maybe<ShadowRequest>;
  response?: Maybe<ShadowResponse>;
  tokensIn?: Maybe<Scalars['Int']['output']>;
  tokensOut?: Maybe<Scalars['Int']['output']>;
};

export type ShadowResultsResponse = {
  __typename?: 'ShadowResultsResponse';
  interactionId: Scalars['ID']['output'];
  shadows: Array<ShadowResult>;
};

export type ShadowUsage = {
  __typename?: 'ShadowUsage';
  completionTokens?: Maybe<Scalars['Int']['output']>;
  promptTokens?: Maybe<Scalars['Int']['output']>;
  totalTokens?: Maybe<Scalars['Int']['output']>;
};

export type Stats = {
  __typename?: 'Stats';
  goVersion: Scalars['String']['output'];
  memory: MemoryStats;
  numGoroutine: Scalars['Int']['output'];
  uptime: Scalars['String']['output'];
};

export type StorageSummary = {
  __typename?: 'StorageSummary';
  enabled: Scalars['Boolean']['output'];
  path?: Maybe<Scalars['String']['output']>;
  type: Scalars['String']['output'];
};

export type TenantSummary = {
  __typename?: 'TenantSummary';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  providerCount: Scalars['Int']['output'];
  routingRules: Scalars['Int']['output'];
  supportsTenant: Scalars['Boolean']['output'];
};

export type TransformationStep = {
  __typename?: 'TransformationStep';
  after?: Maybe<Scalars['JSON']['output']>;
  before?: Maybe<Scalars['JSON']['output']>;
  description: Scalars['String']['output'];
  stage: Scalars['String']['output'];
};

export type Usage = {
  __typename?: 'Usage';
  inputTokens?: Maybe<Scalars['Int']['output']>;
  outputTokens?: Maybe<Scalars['Int']['output']>;
  totalTokens?: Maybe<Scalars['Int']['output']>;
};
