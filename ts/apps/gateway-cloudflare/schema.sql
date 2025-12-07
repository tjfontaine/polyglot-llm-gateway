-- D1 Database Schema for polyglot-llm-gateway
-- Run with: wrangler d1 execute polyglot-gateway --file=./schema.sql

-- Conversations table
CREATE TABLE IF NOT EXISTS conversations (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_name TEXT,
  model TEXT,
  metadata TEXT DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  usage TEXT,
  timestamp TEXT NOT NULL,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);

-- Responses table (for Responses API)
CREATE TABLE IF NOT EXISTS responses (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_name TEXT,
  thread_key TEXT,
  previous_response_id TEXT,
  model TEXT NOT NULL,
  status TEXT NOT NULL,
  request TEXT,
  response TEXT,
  error TEXT,
  usage TEXT,
  metadata TEXT DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_responses_tenant ON responses(tenant_id);
CREATE INDEX IF NOT EXISTS idx_responses_thread ON responses(thread_key);
CREATE INDEX IF NOT EXISTS idx_responses_updated ON responses(updated_at);

-- Interaction events table
CREATE TABLE IF NOT EXISTS interaction_events (
  id TEXT PRIMARY KEY,
  interaction_id TEXT NOT NULL,
  type TEXT NOT NULL,
  payload TEXT,
  timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_interaction ON interaction_events(interaction_id);

-- Shadow results table
CREATE TABLE IF NOT EXISTS shadow_results (
  id TEXT PRIMARY KEY,
  interaction_id TEXT NOT NULL,
  provider_name TEXT NOT NULL,
  request TEXT,
  response TEXT,
  error TEXT,
  duration_ms INTEGER NOT NULL,
  divergences TEXT NOT NULL DEFAULT '[]',
  has_structural_divergence INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_shadow_interaction ON shadow_results(interaction_id);
CREATE INDEX IF NOT EXISTS idx_shadow_divergence ON shadow_results(has_structural_divergence);

-- Thread state table (for Responses API threading)
CREATE TABLE IF NOT EXISTS thread_state (
  thread_key TEXT PRIMARY KEY,
  response_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
