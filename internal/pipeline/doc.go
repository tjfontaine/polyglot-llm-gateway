// Package pipeline provides the webhook pipeline execution engine.
//
// The pipeline allows users to define ordered stages that can approve/deny,
// mutate, or observe requests and responses. Each stage is a webhook call
// to an external service.
//
// # Architecture
//
// Pipelines are configured per-app and execute in two phases:
//   - Pre-pipeline: Runs before routing, can mutate or deny the request
//   - Post-pipeline: Runs after response, can mutate or deny (squelch) the response
//
// # Webhook Contract
//
// Webhooks receive StageInput and must return StageOutput:
//
//	POST <webhook_url>
//	Content-Type: application/json
//
//	{
//	  "phase": "request" | "response",
//	  "request": { ... canonical request ... },
//	  "response": { ... canonical response ... },  // only in post phase
//	  "metadata": { "app_name": "...", "request_id": "...", ... }
//	}
//
// Response:
//
//	{
//	  "action": "allow" | "deny" | "mutate",
//	  "request": { ... },      // if mutating request
//	  "response": { ... },     // if mutating response
//	  "deny_reason": "..."     // if denying
//	}
package pipeline
