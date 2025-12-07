# Example Webhook Server

This is an example TypeScript webhook server that demonstrates the pipeline webhook contract.

## Running

Requires [Deno](https://deno.land/) runtime:

```bash
deno run --allow-net server.ts
```

The server runs on port 8088 and handles both pre-pipeline (request) and post-pipeline (response) stages.

## Features

- **Content Filtering** (pre-stage): Blocks requests containing sensitive keywords
- **Response Mutation** (post-stage): Adds a disclaimer to assistant responses

## Configuration

Add pipeline configuration to your gateway config:

```yaml
apps:
  - name: secure-api
    frontdoor: openai
    path: /v1
    pipeline:
      stages:
        - name: content-filter
          type: pre
          url: http://localhost:8088
          timeout: 5s
          on_error: deny
          order: 1
          
        - name: add-disclaimer
          type: post
          url: http://localhost:8088
          timeout: 5s
          on_error: allow  # Don't fail if disclaimer fails
          order: 1
```

## Webhook Contract

The server receives JSON with this structure:

```typescript
interface StageInput {
  phase: "request" | "response";
  request: CanonicalRequest;
  response?: CanonicalResponse;  // Only for post-stage
  metadata: {
    app_name: string;
    request_id: string;
    interaction_id: string;
    timestamp: string;
  };
}
```

And must return:

```typescript
interface StageOutput {
  action: "allow" | "deny" | "mutate";
  request?: CanonicalRequest;   // If mutating request
  response?: CanonicalResponse; // If mutating response
  deny_reason?: string;         // If denying
}
```
