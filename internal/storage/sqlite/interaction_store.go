package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// SaveInteraction saves an interaction record
func (s *Store) SaveInteraction(ctx context.Context, interaction *domain.Interaction) error {
	now := time.Now()
	if interaction.CreatedAt.IsZero() {
		interaction.CreatedAt = now
	}
	interaction.UpdatedAt = now

	metadata, err := json.Marshal(interaction.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	requestHeaders, err := json.Marshal(interaction.RequestHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal request headers: %w", err)
	}

	var requestRaw, requestCanonical, requestUnmappedFields, requestProvider sql.NullString
	var responseRaw, responseCanonical, responseUnmappedFields, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString

	if interaction.Request != nil {
		if len(interaction.Request.Raw) > 0 {
			requestRaw = sql.NullString{String: string(interaction.Request.Raw), Valid: true}
		}
		if len(interaction.Request.CanonicalJSON) > 0 {
			requestCanonical = sql.NullString{String: string(interaction.Request.CanonicalJSON), Valid: true}
		}
		if len(interaction.Request.UnmappedFields) > 0 {
			fieldsJSON, _ := json.Marshal(interaction.Request.UnmappedFields)
			requestUnmappedFields = sql.NullString{String: string(fieldsJSON), Valid: true}
		}
		if len(interaction.Request.ProviderRequest) > 0 {
			requestProvider = sql.NullString{String: string(interaction.Request.ProviderRequest), Valid: true}
		}
	}

	if interaction.Response != nil {
		if len(interaction.Response.Raw) > 0 {
			responseRaw = sql.NullString{String: string(interaction.Response.Raw), Valid: true}
		}
		if len(interaction.Response.CanonicalJSON) > 0 {
			responseCanonical = sql.NullString{String: string(interaction.Response.CanonicalJSON), Valid: true}
		}
		if len(interaction.Response.UnmappedFields) > 0 {
			fieldsJSON, _ := json.Marshal(interaction.Response.UnmappedFields)
			responseUnmappedFields = sql.NullString{String: string(fieldsJSON), Valid: true}
		}
		if len(interaction.Response.ClientResponse) > 0 {
			responseClient = sql.NullString{String: string(interaction.Response.ClientResponse), Valid: true}
		}
		if interaction.Response.FinishReason != "" {
			responseFinishReason = sql.NullString{String: interaction.Response.FinishReason, Valid: true}
		}
		if interaction.Response.Usage != nil {
			usageJSON, _ := json.Marshal(interaction.Response.Usage)
			responseUsage = sql.NullString{String: string(usageJSON), Valid: true}
		}
	}

	if interaction.Error != nil {
		errorType = sql.NullString{String: interaction.Error.Type, Valid: true}
		if interaction.Error.Code != "" {
			errorCode = sql.NullString{String: interaction.Error.Code, Valid: true}
		}
		errorMessage = sql.NullString{String: interaction.Error.Message, Valid: true}
	}

	streaming := 0
	if interaction.Streaming {
		streaming = 1
	}

	query := `INSERT INTO interactions (
		id, tenant_id, frontdoor, provider, app_name, requested_model, served_model, provider_model,
		streaming, status, duration_ns,
		request_raw, request_canonical, request_unmapped_fields, request_provider,
		response_raw, response_canonical, response_unmapped_fields, response_client,
		response_finish_reason, response_usage,
		error_type, error_code, error_message,
		metadata, request_headers, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		interaction.ID, interaction.TenantID, string(interaction.Frontdoor), interaction.Provider,
		interaction.AppName, interaction.RequestedModel, interaction.ServedModel, interaction.ProviderModel,
		streaming, string(interaction.Status), int64(interaction.Duration),
		requestRaw, requestCanonical, requestUnmappedFields, requestProvider,
		responseRaw, responseCanonical, responseUnmappedFields, responseClient,
		responseFinishReason, responseUsage,
		errorType, errorCode, errorMessage,
		string(metadata), string(requestHeaders),
		interaction.CreatedAt, interaction.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save interaction: %w", err)
	}

	return nil
}

// GetInteraction retrieves an interaction by ID
func (s *Store) GetInteraction(ctx context.Context, id string) (*domain.Interaction, error) {
	query := `SELECT 
		id, tenant_id, frontdoor, provider, app_name, requested_model, served_model, provider_model,
		streaming, status, duration_ns,
		request_raw, request_canonical, request_unmapped_fields, request_provider,
		response_raw, response_canonical, response_unmapped_fields, response_client,
		response_finish_reason, response_usage,
		error_type, error_code, error_message,
		metadata, request_headers, created_at, updated_at
	FROM interactions WHERE id = ?`

	var interaction domain.Interaction
	var frontdoor, status string
	var streaming int
	var durationNs int64
	var requestRaw, requestCanonical, requestUnmappedFields, requestProvider sql.NullString
	var responseRaw, responseCanonical, responseUnmappedFields, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString
	var metadataStr, requestHeadersStr sql.NullString
	var appName, servedModel, providerModel sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&interaction.ID, &interaction.TenantID, &frontdoor, &interaction.Provider,
		&appName, &interaction.RequestedModel, &servedModel, &providerModel,
		&streaming, &status, &durationNs,
		&requestRaw, &requestCanonical, &requestUnmappedFields, &requestProvider,
		&responseRaw, &responseCanonical, &responseUnmappedFields, &responseClient,
		&responseFinishReason, &responseUsage,
		&errorType, &errorCode, &errorMessage,
		&metadataStr, &requestHeadersStr,
		&interaction.CreatedAt, &interaction.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("interaction %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction: %w", err)
	}

	interaction.Frontdoor = domain.APIType(frontdoor)
	interaction.Status = domain.InteractionStatus(status)
	interaction.Streaming = streaming == 1
	interaction.Duration = time.Duration(durationNs)

	if appName.Valid {
		interaction.AppName = appName.String
	}
	if servedModel.Valid {
		interaction.ServedModel = servedModel.String
	}
	if providerModel.Valid {
		interaction.ProviderModel = providerModel.String
	}

	// Unmarshal request
	if requestRaw.Valid || requestCanonical.Valid || requestUnmappedFields.Valid || requestProvider.Valid {
		interaction.Request = &domain.InteractionRequest{}
		if requestRaw.Valid {
			interaction.Request.Raw = json.RawMessage(requestRaw.String)
		}
		if requestCanonical.Valid {
			interaction.Request.CanonicalJSON = json.RawMessage(requestCanonical.String)
		}
		if requestUnmappedFields.Valid {
			json.Unmarshal([]byte(requestUnmappedFields.String), &interaction.Request.UnmappedFields)
		}
		if requestProvider.Valid {
			interaction.Request.ProviderRequest = json.RawMessage(requestProvider.String)
		}
	}

	// Unmarshal response
	if responseRaw.Valid || responseCanonical.Valid || responseUnmappedFields.Valid || responseClient.Valid || responseFinishReason.Valid || responseUsage.Valid {
		interaction.Response = &domain.InteractionResponse{}
		if responseRaw.Valid {
			interaction.Response.Raw = json.RawMessage(responseRaw.String)
		}
		if responseCanonical.Valid {
			interaction.Response.CanonicalJSON = json.RawMessage(responseCanonical.String)
		}
		if responseUnmappedFields.Valid {
			json.Unmarshal([]byte(responseUnmappedFields.String), &interaction.Response.UnmappedFields)
		}
		if responseClient.Valid {
			interaction.Response.ClientResponse = json.RawMessage(responseClient.String)
		}
		if responseFinishReason.Valid {
			interaction.Response.FinishReason = responseFinishReason.String
		}
		if responseUsage.Valid {
			interaction.Response.Usage = &domain.Usage{}
			json.Unmarshal([]byte(responseUsage.String), interaction.Response.Usage)
		}
	}

	// Unmarshal error
	if errorType.Valid {
		interaction.Error = &domain.InteractionError{
			Type:    errorType.String,
			Message: errorMessage.String,
		}
		if errorCode.Valid {
			interaction.Error.Code = errorCode.String
		}
	}

	// Unmarshal metadata
	if metadataStr.Valid && metadataStr.String != "" {
		json.Unmarshal([]byte(metadataStr.String), &interaction.Metadata)
	}

	// Unmarshal request headers
	if requestHeadersStr.Valid && requestHeadersStr.String != "" {
		json.Unmarshal([]byte(requestHeadersStr.String), &interaction.RequestHeaders)
	}

	return &interaction, nil
}

// UpdateInteraction updates an existing interaction
func (s *Store) UpdateInteraction(ctx context.Context, interaction *domain.Interaction) error {
	interaction.UpdatedAt = time.Now()

	metadata, err := json.Marshal(interaction.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var responseRaw, responseCanonical, responseUnmappedFields, responseClient sql.NullString
	var responseFinishReason, responseUsage sql.NullString
	var errorType, errorCode, errorMessage sql.NullString

	if interaction.Response != nil {
		if len(interaction.Response.Raw) > 0 {
			responseRaw = sql.NullString{String: string(interaction.Response.Raw), Valid: true}
		}
		if len(interaction.Response.CanonicalJSON) > 0 {
			responseCanonical = sql.NullString{String: string(interaction.Response.CanonicalJSON), Valid: true}
		}
		if len(interaction.Response.UnmappedFields) > 0 {
			fieldsJSON, _ := json.Marshal(interaction.Response.UnmappedFields)
			responseUnmappedFields = sql.NullString{String: string(fieldsJSON), Valid: true}
		}
		if len(interaction.Response.ClientResponse) > 0 {
			responseClient = sql.NullString{String: string(interaction.Response.ClientResponse), Valid: true}
		}
		if interaction.Response.FinishReason != "" {
			responseFinishReason = sql.NullString{String: interaction.Response.FinishReason, Valid: true}
		}
		if interaction.Response.Usage != nil {
			usageJSON, _ := json.Marshal(interaction.Response.Usage)
			responseUsage = sql.NullString{String: string(usageJSON), Valid: true}
		}
	}

	if interaction.Error != nil {
		errorType = sql.NullString{String: interaction.Error.Type, Valid: true}
		if interaction.Error.Code != "" {
			errorCode = sql.NullString{String: interaction.Error.Code, Valid: true}
		}
		errorMessage = sql.NullString{String: interaction.Error.Message, Valid: true}
	}

	query := `UPDATE interactions SET
		served_model = ?, provider_model = ?,
		status = ?, duration_ns = ?,
		response_raw = ?, response_canonical = ?, response_unmapped_fields = ?, response_client = ?,
		response_finish_reason = ?, response_usage = ?,
		error_type = ?, error_code = ?, error_message = ?,
		metadata = ?, updated_at = ?
	WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		interaction.ServedModel, interaction.ProviderModel,
		string(interaction.Status), int64(interaction.Duration),
		responseRaw, responseCanonical, responseUnmappedFields, responseClient,
		responseFinishReason, responseUsage,
		errorType, errorCode, errorMessage,
		string(metadata), interaction.UpdatedAt, interaction.ID)

	if err != nil {
		return fmt.Errorf("failed to update interaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("interaction %s not found", interaction.ID)
	}

	return nil
}

// ListInteractions lists interactions with pagination and optional filtering
func (s *Store) ListInteractions(ctx context.Context, opts storage.InteractionListOptions) ([]*domain.InteractionSummary, error) {
	query := `SELECT 
		id, tenant_id, frontdoor, provider, app_name, requested_model, served_model,
		streaming, status, duration_ns, metadata, created_at, updated_at
	FROM interactions WHERE 1=1`

	var args []interface{}

	if opts.TenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, opts.TenantID)
	}
	if opts.Frontdoor != "" {
		query += " AND frontdoor = ?"
		args = append(args, string(opts.Frontdoor))
	}
	if opts.Provider != "" {
		query += " AND provider = ?"
		args = append(args, opts.Provider)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}

	query += " ORDER BY updated_at DESC"

	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query interactions: %w", err)
	}
	defer rows.Close()

	var interactions []*domain.InteractionSummary
	for rows.Next() {
		var summary domain.InteractionSummary
		var frontdoor, status string
		var streaming int
		var durationNs int64
		var metadataStr sql.NullString
		var appName, servedModel sql.NullString

		if err := rows.Scan(
			&summary.ID, &summary.TenantID, &frontdoor, &summary.Provider,
			&appName, &summary.RequestedModel, &servedModel,
			&streaming, &status, &durationNs,
			&metadataStr, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
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
		if metadataStr.Valid && metadataStr.String != "" {
			json.Unmarshal([]byte(metadataStr.String), &summary.Metadata)
		}

		interactions = append(interactions, &summary)
	}

	return interactions, rows.Err()
}
