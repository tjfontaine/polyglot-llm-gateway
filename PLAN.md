# Auditable Pipeline Plan (Anthropic → IR → Threaded IR → OpenAI Responses → Back)

## Objectives
- Make the request/response path fully auditable and debuggable end-to-end.
- Preserve exact payloads at each boundary (ingress/egress), with explicit stage/direction metadata.
- Support threaded Responses flows (previous_response_id + thread state) without losing context.
- Feed a single, coherent event stream into the control plane (no split int/resp/conversation views).

## Target Pipeline (happy-path)
1) **Ingress (Anthropic frontdoor)**  
   - Read raw request bytes; attach request_id, tenant/app/provider/model metadata.  
   - Decode → CanonicalRequest (IR).  
   - Emit event: `stage=frontdoor_decode`, `direction=ingress`, `raw_request`, `canonical_request`.
2) **Thread key resolution (optional)**  
   - Derive thread_key (configured path or metadata.user_id).  
   - Lookup thread_state (previous_response_id) if threading enabled.  
   - Mutate canonical/provider payload accordingly.  
   - Emit event: `stage=thread_resolve`, `thread_key`, `previous_response_id`.
3) **Provider encode + send (OpenAI Responses)**  
   - Encode CanonicalRequest → provider request (Responses).  
   - Emit event: `stage=provider_encode`, `direction=egress`, `provider_request`.
4) **Provider response / stream**  
   - For non-stream: decode provider response → CanonicalResponse.  
   - For stream: decode each event → CanonicalEvent; accumulate Usage/FinishReason.  
   - Emit event(s): `stage=provider_decode`, `direction=ingress`, `provider_response` (or stream chunks).
5) **Thread state update**  
   - When a final response_id is known, persist `thread_state[thread_key]=response_id`.  
   - Emit event: `stage=thread_update`, `thread_key`, `response_id`.
6) **Client encode (Anthropic)**  
   - Encode CanonicalResponse → Anthropic format.  
   - Emit event: `stage=frontdoor_encode`, `direction=egress`, `client_response`.

## Proposed Carrying Type (for storage & control plane)
`InteractionEvent` (append-only)
- `id` (uuid)
- `interaction_id` (stable per end-user “call”; for Responses reuse resp_id, otherwise int_* UUID)
- `stage` (frontdoor_decode | thread_resolve | provider_encode | provider_decode | thread_update | frontdoor_encode | error)
- `direction` (ingress | egress | internal)
- `api_type` (anthropic | openai | responses | canonical)
- `provider` / `frontdoor` / `app` / `tenant`
- `model_requested` / `model_served` / `provider_model`
- `thread_key` (nullable), `previous_response_id` (nullable)
- `raw` (json/text) – exact payload at that stage
- `canonical` (json) – if available
- `headers` (filtered) – optional
- `metadata` (json) – request_id, user_id, etc.
- `created_at`

Notes:
- This replaces the need to juggle int/resp/conversation tables for audit views; control plane can reconstruct timelines from events grouped by `interaction_id`.
- Keep `thread_state` table as is for fast lookup/updates.

## Storage Layout
- **interaction_events** (new) – append-only, wide columns as above. Indexed by interaction_id, stage, created_at.
- **thread_state** (keep) – thread_key → latest response_id (already present).
- Legacy tables (interactions/responses/conversations) can remain for compatibility; new UI reads from interaction_events. Optionally dual-write during rollout.

## Recording Rules
- Always emit at least: frontdoor_decode, provider_encode, provider_decode, frontdoor_encode.  
- If threading is enabled: emit thread_resolve (lookup) and thread_update (persist).  
- On error at any stage: emit `stage=error` with the error payload and stop sequence.
- Streaming: buffer chunk metadata to include model/usage; emit one provider_decode per meaningful chunk plus a final aggregate event with finish_reason/usage.
- Redaction: allow a config hook to strip/retain headers or fields before storage.

## Control Plane Consumption
- Query: `SELECT * FROM interaction_events WHERE interaction_id=? ORDER BY created_at`.  
- Present as a timeline with per-stage payload tabs (raw/canonical), thread metadata, and mapped models.  
- For threading, show: thread_key, previous_response_id used, updated response_id.

## Rollout Plan
1) Schema: create `interaction_events` table + indexes; keep existing tables.  
2) Code: introduce an `eventLogger` component used by frontdoors and providers; wire Anthropic/OpenAI paths first.  
3) Dual-write: emit both legacy interaction record and new events until stable.  
4) UI: control plane switches to reading event timelines; fall back to legacy if no events found.  
5) Cleanup: optional migration/archival of legacy tables once consumers are off them.

## Validation Checklist
- Unit tests per stage (encode/decode, thread resolve/update, event emission).  
- Integration: Anthropic → Responses round-trip producing ordered events with correct models/thread ids.  
- Streaming: ensures final usage/finish_reason is captured in the last event.  
- DB: verify WAL, indexes, and size growth; add retention/compaction hooks if needed.

## Open Questions / Clarifications
1) Should we dual-write to legacy tables or gate the new event log behind a feature flag?  
2) Any fields to redact before storage (headers, system prompts, user content)?  
3) Streaming granularity: store every chunk or only deltas + final summary?  
4) Control plane needs: timeline only, or also diff views (raw vs canonical) and thread lineage?  
5) Do we need migration of existing records into interaction_events, or only forward-fill?  
6) How do we want to key interaction_id for non-Responses (e.g., use request_id vs generated int_uuid)?  
7) Any SLA on storage growth/retention (TTL, compaction cadence)?
