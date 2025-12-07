/**
 * Example webhook server demonstrating the pipeline webhook contract.
 * 
 * Run with: deno run --allow-net server.ts
 * 
 * Features:
 * 1. Content filtering on request (pre-stage)
 * 2. Response mutation (post-stage)
 */

import type {
    StageInput,
    StageOutput,
    CanonicalRequest,
    CanonicalResponse,
} from "./types.ts";

import { allow, deny, mutateRequest, mutateResponse } from "./types.ts";

// Blocked keywords for content filtering
const BLOCKED_KEYWORDS = ["secret", "password", "api_key"];

/**
 * Pre-pipeline handler: Filter incoming requests
 */
function handleRequest(input: StageInput): StageOutput {
    console.log(`[PRE] Processing request for model: ${input.request.model}`);

    // Check each message for blocked content
    for (const msg of input.request.messages) {
        const content = msg.content?.toLowerCase() ?? "";

        for (const keyword of BLOCKED_KEYWORDS) {
            if (content.includes(keyword)) {
                console.log(`[PRE] Denying request - found blocked keyword: ${keyword}`);
                return deny(`Content policy violation: message contains blocked keyword "${keyword}"`);
            }
        }
    }

    // Example: Force model upgrade for certain models
    if (input.request.model === "gpt-3.5-turbo") {
        console.log(`[PRE] Upgrading model from gpt-3.5-turbo to gpt-4o-mini`);
        return mutateRequest({
            ...input.request,
            model: "gpt-4o-mini",
        });
    }

    console.log(`[PRE] Request allowed`);
    return allow();
}

/**
 * Post-pipeline handler: Process/filter responses
 */
function handleResponse(input: StageInput): StageOutput {
    console.log(`[POST] Processing response from model: ${input.response?.model}`);

    if (!input.response?.choices?.length) {
        return allow();
    }

    // Example mutation: Add a disclaimer to the response
    const mutatedResponse: CanonicalResponse = {
        ...input.response,
        choices: input.response.choices.map(choice => ({
            ...choice,
            message: {
                ...choice.message,
                content: choice.message.role === "assistant"
                    ? choice.message.content + "\n\n---\n*Response processed by webhook pipeline.*"
                    : choice.message.content,
            },
        })),
    };

    console.log(`[POST] Mutating response with disclaimer`);
    return mutateResponse(mutatedResponse);
}

// Start server
Deno.serve({ port: 8088 }, async (req) => {
    if (req.method !== "POST") {
        return new Response("Method not allowed", { status: 405 });
    }

    try {
        const input: StageInput = await req.json();
        console.log(`\n=== Webhook called: phase=${input.phase} ===`);
        console.log(`Metadata:`, input.metadata);

        const output = input.phase === "request"
            ? handleRequest(input)
            : handleResponse(input);

        console.log(`Result: ${output.action}`);
        return Response.json(output);
    } catch (error) {
        console.error("Error:", error);
        return Response.json(allow()); // Fail-open on errors
    }
});

console.log(`
🚀 Example webhook server running on http://localhost:8088

This webhook demonstrates:
  • Content filtering (blocks messages with "secret", "password", "api_key")
  • Model upgrade (gpt-3.5-turbo → gpt-4o-mini)
  • Response disclaimer (adds footer to assistant messages)

Configure your gateway with:

apps:
  - name: my-app
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
          on_error: allow
          order: 1
`);
