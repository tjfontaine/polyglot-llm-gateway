package sqldb

import (
"context"
"database/sql"
"encoding/json"
"fmt"
"time"

"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Ensure Store implements ShadowStore
var _ ports.ShadowStore = (*Store)(nil)

// SaveShadowResult saves a shadow execution result
func (s *Store) SaveShadowResult(ctx context.Context, result *domain.ShadowResult) error {
	now := time.Now()
	if result.CreatedAt.IsZero() {
		result.CreatedAt = now
	}

	var requestCanonical, requestProvider sql.NullString
	var responseRaw, responseCanonical, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString

	// Serialize request
	if result.Request != nil {
		if len(result.Request.Canonical) > 0 {
			requestCanonical = sql.NullString{String: string(result.Request.Canonical), Valid: true}
		}
		if len(result.Request.ProviderRequest) > 0 {
			requestProvider = sql.NullString{String: string(result.Request.ProviderRequest), Valid: true}
		}
	}

	// Serialize response
	if result.Response != nil {
		if len(result.Response.Raw) > 0 {
			responseRaw = sql.NullString{String: string(result.Response.Raw), Valid: true}
		}
		if len(result.Response.Canonical) > 0 {
			responseCanonical = sql.NullString{String: string(result.Response.Canonical), Valid: true}
		}
		if len(result.Response.ClientResponse) > 0 {
			responseClient = sql.NullString{String: string(result.Response.ClientResponse), Valid: true}
		}
		if result.Response.FinishReason != "" {
			responseFinishReason = sql.NullString{String: result.Response.FinishReason, Valid: true}
		}
		if result.Response.Usage != nil {
			usageJSON, _ := json.Marshal(result.Response.Usage)
			responseUsage = sql.NullString{String: string(usageJSON), Valid: true}
		}
	}

	// Serialize error
	if result.Error != nil {
		errorType = sql.NullString{String: result.Error.Type, Valid: true}
		if result.Error.Code != "" {
			errorCode = sql.NullString{String: result.Error.Code, Valid: true}
		}
		errorMessage = sql.NullString{String: result.Error.Message, Valid: true}
	}

	// Serialize divergences
	var divergencesJSON sql.NullString
	if len(result.Divergences) > 0 {
		divergencesBytes, err := json.Marshal(result.Divergences)
		if err != nil {
			return fmt.Errorf("failed to marshal divergences: %w", err)
		}
		divergencesJSON = sql.NullString{String: string(divergencesBytes), Valid: true}
		result.HasStructuralDivergence = true
	}

	hasStructuralDivergence := 0
	if result.HasStructuralDivergence {
		hasStructuralDivergence = 1
	}

	query := s.dialect.Rebind(`INSERT INTO shadow_results (
id, interaction_id, provider_name, provider_model,
request_canonical, request_provider,
response_raw, response_canonical, response_client, response_finish_reason, response_usage,
error_type, error_code, error_message,
duration_ns, tokens_in, tokens_out,
divergences, has_structural_divergence,
created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

	_, err := s.db.ExecContext(ctx, query,
result.ID, result.InteractionID, result.ProviderName, result.ProviderModel,
requestCanonical, requestProvider,
responseRaw, responseCanonical, responseClient, responseFinishReason, responseUsage,
errorType, errorCode, errorMessage,
int64(result.Duration), result.TokensIn, result.TokensOut,
divergencesJSON, hasStructuralDivergence,
result.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save shadow result: %w", err)
	}

	return nil
}

// GetShadowResult retrieves a shadow result by ID
func (s *Store) GetShadowResult(ctx context.Context, id string) (*domain.ShadowResult, error) {
	query := s.dialect.Rebind(`SELECT 
		id, interaction_id, provider_name, provider_model,
		request_canonical, request_provider,
		response_raw, response_canonical, response_client, response_finish_reason, response_usage,
		error_type, error_code, error_message,
		duration_ns, tokens_in, tokens_out,
		divergences, has_structural_divergence,
		created_at
	FROM shadow_results WHERE id = ?`)

	return s.scanShadowResult(s.db.QueryRowContext(ctx, query, id))
}

// GetShadowResults retrieves all shadow results for an interaction
func (s *Store) GetShadowResults(ctx context.Context, interactionID string) ([]*domain.ShadowResult, error) {
	query := s.dialect.Rebind(`SELECT 
		id, interaction_id, provider_name, provider_model,
		request_canonical, request_provider,
		response_raw, response_canonical, response_client, response_finish_reason, response_usage,
		error_type, error_code, error_message,
		duration_ns, tokens_in, tokens_out,
		divergences, has_structural_divergence,
		created_at
	FROM shadow_results 
	WHERE interaction_id = ?
	ORDER BY created_at ASC`)

	rows, err := s.db.QueryContext(ctx, query, interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shadow results: %w", err)
	}
	defer rows.Close()

	var results []*domain.ShadowResult
	for rows.Next() {
		result, err := s.scanShadowResultRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shadow results: %w", err)
	}

	return results, nil
}

// ListDivergentInteractions lists interactions that have shadow results with divergences
func (s *Store) ListDivergentInteractions(ctx context.Context, opts *ports.DivergenceListOptions) ([]*domain.InteractionSummary, error) {
	if opts == nil {
		opts = &ports.DivergenceListOptions{Limit: 100}
	}
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	// Build query to find interactions with divergent shadows
	query := `SELECT DISTINCT i.id, i.tenant_id, i.frontdoor, i.provider, i.app_name,
		i.requested_model, i.served_model, i.streaming, i.status, i.duration_ns,
		i.metadata, i.created_at, i.updated_at
	FROM interactions i
	INNER JOIN shadow_results sr ON i.id = sr.interaction_id
	WHERE sr.has_structural_divergence = 1`

	args := []any{}

	if opts.ProviderName != "" {
		query += " AND sr.provider_name = ?"
		args = append(args, opts.ProviderName)
	}

	query += " ORDER BY i.updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, s.dialect.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query divergent interactions: %w", err)
	}
	defer rows.Close()

	var results []*domain.InteractionSummary
	for rows.Next() {
		var summary domain.InteractionSummary
		var frontdoor, status string
		var streaming int
		var durationNs int64
		var metadataStr, appName, servedModel sql.NullString

		err := rows.Scan(
&summary.ID, &summary.TenantID, &frontdoor, &summary.Provider, &appName,
			&summary.RequestedModel, &servedModel, &streaming, &status, &durationNs,
			&metadataStr, &summary.CreatedAt, &summary.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan interaction: %w", err)
		}

		summary.Frontdoor = domain.APIType(frontdoor)
		summary.Status = domain.InteractionStatus(status)
		summary.Streaming = streaming == 1
		summary.Duration = time.Duration(durationNs)

		if appName.Valid {
			summary.AppName = appName.String
		}
		if servedModel.Valid {
			summary.ServedModel = servedModel.String
		}
		if metadataStr.Valid {
			_ = json.Unmarshal([]byte(metadataStr.String), &summary.Metadata)
		}

		results = append(results, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating divergent interactions: %w", err)
	}

	return results, nil
}

// GetDivergentShadowCount returns the count of shadow results with divergences
func (s *Store) GetDivergentShadowCount(ctx context.Context) (int, error) {
	var count int
	query := s.dialect.Rebind(`SELECT COUNT(*) FROM shadow_results WHERE has_structural_divergence = 1`)
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count divergent shadows: %w", err)
	}
	return count, nil
}

// scanShadowResult scans a single shadow result from a sql.Row
func (s *Store) scanShadowResult(row *sql.Row) (*domain.ShadowResult, error) {
	var result domain.ShadowResult
	var requestCanonical, requestProvider sql.NullString
	var responseRaw, responseCanonical, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString
	var divergencesJSON sql.NullString
	var durationNs int64
	var hasStructuralDivergence int
	var providerModel sql.NullString

	err := row.Scan(
&result.ID, &result.InteractionID, &result.ProviderName, &providerModel,
		&requestCanonical, &requestProvider,
		&responseRaw, &responseCanonical, &responseClient, &responseFinishReason, &responseUsage,
		&errorType, &errorCode, &errorMessage,
		&durationNs, &result.TokensIn, &result.TokensOut,
		&divergencesJSON, &hasStructuralDivergence,
		&result.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("shadow result not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan shadow result: %w", err)
	}

	result.Duration = time.Duration(durationNs)
	result.HasStructuralDivergence = hasStructuralDivergence == 1

	if providerModel.Valid {
		result.ProviderModel = providerModel.String
	}

	// Deserialize request
	if requestCanonical.Valid || requestProvider.Valid {
		result.Request = &domain.ShadowRequest{}
		if requestCanonical.Valid {
			result.Request.Canonical = json.RawMessage(requestCanonical.String)
		}
		if requestProvider.Valid {
			result.Request.ProviderRequest = json.RawMessage(requestProvider.String)
		}
	}

	// Deserialize response
	if responseRaw.Valid || responseCanonical.Valid || responseClient.Valid {
		result.Response = &domain.ShadowResponse{}
		if responseRaw.Valid {
			result.Response.Raw = json.RawMessage(responseRaw.String)
		}
		if responseCanonical.Valid {
			result.Response.Canonical = json.RawMessage(responseCanonical.String)
		}
		if responseClient.Valid {
			result.Response.ClientResponse = json.RawMessage(responseClient.String)
		}
		if responseFinishReason.Valid {
			result.Response.FinishReason = responseFinishReason.String
		}
		if responseUsage.Valid {
			var usage domain.Usage
			if err := json.Unmarshal([]byte(responseUsage.String), &usage); err == nil {
				result.Response.Usage = &usage
			}
		}
	}

	// Deserialize error
	if errorType.Valid {
		result.Error = &domain.InteractionError{
			Type:    errorType.String,
			Message: errorMessage.String,
		}
		if errorCode.Valid {
			result.Error.Code = errorCode.String
		}
	}

	// Deserialize divergences
	if divergencesJSON.Valid {
		if err := json.Unmarshal([]byte(divergencesJSON.String), &result.Divergences); err != nil {
			return nil, fmt.Errorf("failed to unmarshal divergences: %w", err)
		}
	}

	return &result, nil
}

// scanShadowResultRow scans a shadow result from sql.Rows
func (s *Store) scanShadowResultRow(rows *sql.Rows) (*domain.ShadowResult, error) {
	var result domain.ShadowResult
	var requestCanonical, requestProvider sql.NullString
	var responseRaw, responseCanonical, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString
	var divergencesJSON sql.NullString
	var durationNs int64
	var hasStructuralDivergence int
	var providerModel sql.NullString

	err := rows.Scan(
&result.ID, &result.InteractionID, &result.ProviderName, &providerModel,
		&requestCanonical, &requestProvider,
		&responseRaw, &responseCanonical, &responseClient, &responseFinishReason, &responseUsage,
		&errorType, &errorCode, &errorMessage,
		&durationNs, &result.TokensIn, &result.TokensOut,
		&divergencesJSON, &hasStructuralDivergence,
		&result.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to scan shadow result: %w", err)
	}

	result.Duration = time.Duration(durationNs)
	result.HasStructuralDivergence = hasStructuralDivergence == 1

	if providerModel.Valid {
		result.ProviderModel = providerModel.String
	}

	// Deserialize request
	if requestCanonical.Valid || requestProvider.Valid {
		result.Request = &domain.ShadowRequest{}
		if requestCanonical.Valid {
			result.Request.Canonical = json.RawMessage(requestCanonical.String)
		}
		if requestProvider.Valid {
			result.Request.ProviderRequest = json.RawMessage(requestProvider.String)
		}
	}

	// Deserialize response
	if responseRaw.Valid || responseCanonical.Valid || responseClient.Valid {
		result.Response = &domain.ShadowResponse{}
		if responseRaw.Valid {
			result.Response.Raw = json.RawMessage(responseRaw.String)
		}
		if responseCanonical.Valid {
			result.Response.Canonical = json.RawMessage(responseCanonical.String)
		}
		if responseClient.Valid {
			result.Response.ClientResponse = json.RawMessage(responseClient.String)
		}
		if responseFinishReason.Valid {
			result.Response.FinishReason = responseFinishReason.String
		}
		if responseUsage.Valid {
			var usage domain.Usage
			if err := json.Unmarshal([]byte(responseUsage.String), &usage); err == nil {
				result.Response.Usage = &usage
			}
		}
	}

	// Deserialize error
	if errorType.Valid {
		result.Error = &domain.InteractionError{
			Type:    errorType.String,
			Message: errorMessage.String,
		}
		if errorCode.Valid {
			result.Error.Code = errorCode.String
		}
	}

	// Deserialize divergences
	if divergencesJSON.Valid {
		if err := json.Unmarshal([]byte(divergencesJSON.String), &result.Divergences); err != nil {
			return nil, fmt.Errorf("failed to unmarshal divergences: %w", err)
		}
	}

	return &result, nil
}
