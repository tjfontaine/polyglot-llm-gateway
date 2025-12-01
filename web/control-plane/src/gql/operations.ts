import { gql } from 'urql';

// Stats query
export const StatsQuery = gql`
  query Stats {
    stats {
      uptime
      goVersion
      numGoroutine
      memory {
        alloc
        totalAlloc
        sys
        numGC
      }
    }
  }
`;

// Overview query
export const OverviewQuery = gql`
  query Overview {
    overview {
      mode
      storage {
        enabled
        type
        path
      }
      apps {
        name
        frontdoor
        path
        provider
        defaultModel
        enableResponses
        modelRouting {
          prefixProviders
          rewrites {
            modelExact
            modelPrefix
            provider
            model
          }
        }
      }
      frontdoors {
        type
        path
        provider
        defaultModel
      }
      providers {
        name
        type
        baseUrl
        supportsResponses
        enablePassthrough
      }
      routing {
        defaultProvider
        rules {
          modelPrefix
          modelExact
          provider
        }
      }
      tenants {
        id
        name
        providerCount
        routingRules
        supportsTenant
      }
    }
  }
`;

// Interactions list query
export const InteractionsQuery = gql`
  query Interactions($filter: InteractionFilter, $limit: Int, $offset: Int) {
    interactions(filter: $filter, limit: $limit, offset: $offset) {
      interactions {
        id
        type
        status
        model
        metadata
        messageCount
        previousResponseId
        createdAt
        updatedAt
      }
      total
    }
  }
`;

// Single interaction detail query
export const InteractionQuery = gql`
  query Interaction($id: ID!) {
    interaction(id: $id) {
      id
      tenantId
      frontdoor
      provider
      appName
      requestedModel
      servedModel
      providerModel
      streaming
      status
      duration
      durationNs
      metadata
      requestHeaders
      createdAt
      updatedAt
      request {
        raw
        canonical
        unmappedFields
        providerRequest
      }
      response {
        raw
        canonical
        unmappedFields
        clientResponse
        finishReason
        usage {
          inputTokens
          outputTokens
          totalTokens
        }
      }
      error {
        type
        code
        message
      }
      transformationSteps {
        stage
        description
        before
        after
      }
      shadows {
        id
        providerName
        providerModel
        durationNs
        tokensIn
        tokensOut
        hasStructuralDivergence
        divergences {
          type
          path
          description
          primary
          shadow
        }
      }
    }
  }
`;

// Interaction events query
export const InteractionEventsQuery = gql`
  query InteractionEvents($interactionId: ID!, $limit: Int) {
    interactionEvents(interactionId: $interactionId, limit: $limit) {
      interactionId
      events {
        id
        interactionId
        stage
        direction
        apiType
        frontdoor
        provider
        appName
        modelRequested
        modelServed
        providerModel
        threadKey
        previousResponseId
        raw
        canonical
        headers
        metadata
        createdAt
      }
    }
  }
`;

// Shadow results for an interaction
export const ShadowResultsQuery = gql`
  query ShadowResults($interactionId: ID!) {
    shadowResults(interactionId: $interactionId) {
      interactionId
      shadows {
        id
        interactionId
        providerName
        providerModel
        request {
          canonical
          providerRequest
        }
        response {
          raw
          canonical
          clientResponse
          finishReason
          usage {
            promptTokens
            completionTokens
            totalTokens
          }
        }
        error {
          type
          code
          message
        }
        durationNs
        tokensIn
        tokensOut
        divergences {
          type
          path
          description
          primary
          shadow
        }
        hasStructuralDivergence
        createdAt
      }
    }
  }
`;

// Divergent shadows query
export const DivergentShadowsQuery = gql`
  query DivergentShadows($limit: Int, $offset: Int, $provider: String) {
    divergentShadows(limit: $limit, offset: $offset, provider: $provider) {
      interactions {
        id
        type
        status
        model
        metadata
        createdAt
        updatedAt
      }
      total
      limit
      offset
    }
  }
`;

// Single shadow detail
export const ShadowQuery = gql`
  query Shadow($id: ID!) {
    shadow(id: $id) {
      id
      interactionId
      providerName
      providerModel
      request {
        canonical
        providerRequest
      }
      response {
        raw
        canonical
        clientResponse
        finishReason
        usage {
          promptTokens
          completionTokens
          totalTokens
        }
      }
      error {
        type
        code
        message
      }
      durationNs
      tokensIn
      tokensOut
      divergences {
        type
        path
        description
        primary
        shadow
      }
      hasStructuralDivergence
      createdAt
    }
  }
`;
