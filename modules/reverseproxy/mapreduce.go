// Package reverseproxy provides map/reduce composite response functionality
package reverseproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	// StrategyMapReduce executes backends in a map/reduce pattern where one backend's
	// response can feed into another backend's request, enabling complex data aggregation
	// and enrichment scenarios.
	StrategyMapReduce CompositeStrategy = "map-reduce"
)

// MapReduceConfig defines the configuration for map/reduce composite routes.
// It supports two main patterns:
// 1. Sequential: Backend A → extract data → Backend B with extracted data → merge
// 2. Parallel: Both backends in parallel → map by common field → merge results
type MapReduceConfig struct {
	// Type specifies the map/reduce pattern to use
	Type MapReduceType `json:"type" yaml:"type" toml:"type" env:"TYPE" desc:"Map/reduce pattern type: sequential or parallel"`

	// SourceBackend is the first backend to query (for sequential pattern)
	SourceBackend string `json:"source_backend" yaml:"source_backend" toml:"source_backend" env:"SOURCE_BACKEND" desc:"Source backend for sequential pattern"`

	// TargetBackend is the backend to query with mapped data (for sequential pattern)
	TargetBackend string `json:"target_backend" yaml:"target_backend" toml:"target_backend" env:"TARGET_BACKEND" desc:"Target backend for sequential pattern"`

	// Backends is the list of backends for parallel pattern
	Backends []string `json:"backends" yaml:"backends" toml:"backends" env:"BACKENDS" desc:"List of backends for parallel pattern"`

	// MappingConfig defines how to extract and map data between backends
	MappingConfig MappingConfig `json:"mapping" yaml:"mapping" toml:"mapping" desc:"Configuration for data extraction and mapping"`

	// MergeStrategy defines how to combine the responses
	MergeStrategy MergeStrategy `json:"merge_strategy" yaml:"merge_strategy" toml:"merge_strategy" env:"MERGE_STRATEGY" desc:"Strategy for merging responses"`

	// AllowEmptyResponses controls whether empty responses from backends are acceptable
	AllowEmptyResponses bool `json:"allow_empty_responses" yaml:"allow_empty_responses" toml:"allow_empty_responses" env:"ALLOW_EMPTY_RESPONSES" desc:"Allow empty responses from backends"`

	// FilterOnEmpty when true, filters out results when ancillary backend returns empty
	FilterOnEmpty bool `json:"filter_on_empty" yaml:"filter_on_empty" toml:"filter_on_empty" env:"FILTER_ON_EMPTY" desc:"Filter results when ancillary data is empty"`
}

// MapReduceType defines the type of map/reduce pattern
type MapReduceType string

const (
	// MapReduceTypeSequential executes backends sequentially, feeding data from one to another
	MapReduceTypeSequential MapReduceType = "sequential"

	// MapReduceTypeParallel executes backends in parallel and maps results by common fields
	MapReduceTypeParallel MapReduceType = "parallel"
)

// MappingConfig defines how data is extracted and mapped between backends
type MappingConfig struct {
	// ExtractPath is the JSON path to extract from the source response
	// Examples: "conversations", "data.items", "results[*].id"
	ExtractPath string `json:"extract_path" yaml:"extract_path" toml:"extract_path" env:"EXTRACT_PATH" desc:"JSON path to extract from source response"`

	// ExtractField is the field name to extract from each item
	// Example: "id", "conversation_id"
	ExtractField string `json:"extract_field" yaml:"extract_field" toml:"extract_field" env:"EXTRACT_FIELD" desc:"Field name to extract from each item"`

	// TargetRequestField is where to place extracted data in the target request
	// Examples: "ids", "conversation_ids"
	TargetRequestField string `json:"target_request_field" yaml:"target_request_field" toml:"target_request_field" env:"TARGET_REQUEST_FIELD" desc:"Field in target request for extracted data"`

	// TargetRequestPath is the path to send the request to on the target backend
	// Example: "/api/followups/bulk"
	TargetRequestPath string `json:"target_request_path" yaml:"target_request_path" toml:"target_request_path" env:"TARGET_REQUEST_PATH" desc:"Path for target backend request"`

	// TargetRequestMethod is the HTTP method for the target request (default: POST)
	TargetRequestMethod string `json:"target_request_method" yaml:"target_request_method" toml:"target_request_method" env:"TARGET_REQUEST_METHOD" default:"POST" desc:"HTTP method for target request"`

	// JoinField is the common field to use for joining parallel responses
	// Example: "id", "conversation_id"
	JoinField string `json:"join_field" yaml:"join_field" toml:"join_field" env:"JOIN_FIELD" desc:"Common field for joining parallel responses"`

	// MergeIntoField is the field name in the final response where merged data appears
	// Example: "followups", "ancillary_data"
	MergeIntoField string `json:"merge_into_field" yaml:"merge_into_field" toml:"merge_into_field" env:"MERGE_INTO_FIELD" desc:"Field name for merged data in final response"`
}

// MergeStrategy defines how responses should be merged
type MergeStrategy string

const (
	// MergeStrategyNested creates a nested structure with backend responses as separate fields
	MergeStrategyNested MergeStrategy = "nested"

	// MergeStrategyFlat merges all backend responses into a single flat structure
	MergeStrategyFlat MergeStrategy = "flat"

	// MergeStrategyEnrich enriches the source response with data from target backend
	MergeStrategyEnrich MergeStrategy = "enrich"

	// MergeStrategyJoin joins responses by a common field (for parallel pattern)
	MergeStrategyJoin MergeStrategy = "join"
)

// executeMapReduce handles the map/reduce strategy execution
func (h *CompositeHandler) executeMapReduce(ctx context.Context, w http.ResponseWriter, r *http.Request, bodyBytes []byte, config *MapReduceConfig) {
	if config == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Map/reduce configuration is missing"))
		return
	}

	switch config.Type {
	case MapReduceTypeSequential:
		h.executeSequentialMapReduce(ctx, w, r, bodyBytes, config)
	case MapReduceTypeParallel:
		h.executeParallelMapReduce(ctx, w, r, bodyBytes, config)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Unknown map/reduce type: %s", config.Type)
	}
}

// executeSequentialMapReduce executes the sequential pattern: A → extract → B → merge
func (h *CompositeHandler) executeSequentialMapReduce(ctx context.Context, w http.ResponseWriter, r *http.Request, bodyBytes []byte, config *MapReduceConfig) {
	// Step 1: Query source backend
	sourceBackend := h.getBackendByID(config.SourceBackend)
	if sourceBackend == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Source backend not found: %s", config.SourceBackend)
		return
	}

	sourceResp, err := h.executeBackendRequest(ctx, sourceBackend, r, bodyBytes)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Failed to query source backend: %v", err)
		return
	}
	defer sourceResp.Body.Close()

	// Check if source response is successful
	if sourceResp.StatusCode >= 400 {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Source backend returned error: %d", sourceResp.StatusCode)
		return
	}

	// Step 2: Extract data from source response
	sourceBody, err := io.ReadAll(sourceResp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Failed to read source response"))
		return
	}

	extractedData, err := extractDataFromResponse(sourceBody, &config.MappingConfig)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to extract data: %v", err)
		return
	}

	// Handle empty extracted data
	if len(extractedData) == 0 {
		if config.AllowEmptyResponses {
			// Return source response as-is
			w.WriteHeader(sourceResp.StatusCode)
			for k, v := range sourceResp.Header {
				for _, val := range v {
					w.Header().Add(k, val)
				}
			}
			_, _ = w.Write(sourceBody)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Step 3: Query target backend with extracted data
	targetBackend := h.getBackendByID(config.TargetBackend)
	if targetBackend == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Target backend not found: %s", config.TargetBackend)
		return
	}

	targetResp, err := h.executeTargetBackendRequest(ctx, targetBackend, &config.MappingConfig, extractedData)
	if err != nil {
		if config.AllowEmptyResponses {
			// Return source response if target fails and empty responses allowed
			w.WriteHeader(sourceResp.StatusCode)
			for k, v := range sourceResp.Header {
				for _, val := range v {
					w.Header().Add(k, val)
				}
			}
			_, _ = w.Write(sourceBody)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Failed to query target backend: %v", err)
		return
	}
	defer targetResp.Body.Close()

	targetBody, err := io.ReadAll(targetResp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Failed to read target response"))
		return
	}

	// Step 4: Merge responses
	mergedResponse, err := mergeResponses(sourceBody, targetBody, config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to merge responses: %v", err)
		return
	}

	// Write merged response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mergedResponse)
}

// executeParallelMapReduce executes the parallel pattern: parallel requests → map → merge
func (h *CompositeHandler) executeParallelMapReduce(ctx context.Context, w http.ResponseWriter, r *http.Request, bodyBytes []byte, config *MapReduceConfig) {
	// Execute all backends in parallel
	responses := make(map[string][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, backendID := range config.Backends {
		backend := h.getBackendByID(backendID)
		if backend == nil {
			continue
		}

		wg.Add(1)
		go func(b *Backend, bid string) {
			defer wg.Done()

			resp, err := h.executeBackendRequest(ctx, b, r, bodyBytes)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			mu.Lock()
			responses[bid] = body
			mu.Unlock()
		}(backend, backendID)
	}

	wg.Wait()

	// Check if we have responses
	if len(responses) == 0 {
		if config.AllowEmptyResponses {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("No successful responses from backends"))
		return
	}

	// Merge parallel responses
	mergedResponse, err := mergeParallelResponses(responses, config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to merge responses: %v", err)
		return
	}

	// Write merged response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mergedResponse)
}

// getBackendByID returns a backend by its ID
func (h *CompositeHandler) getBackendByID(id string) *Backend {
	for _, backend := range h.backends {
		if backend.ID == id {
			return backend
		}
	}
	return nil
}

// extractDataFromResponse extracts data from a JSON response based on mapping config
func extractDataFromResponse(body []byte, mapping *MappingConfig) ([]interface{}, error) {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Navigate to the extract path
	current := data
	if mapping.ExtractPath != "" {
		parts := strings.Split(mapping.ExtractPath, ".")
		for _, part := range parts {
			if part == "" {
				continue
			}

			switch v := current.(type) {
			case map[string]interface{}:
				current = v[part]
			default:
				return nil, fmt.Errorf("%w at path %s: %s", ErrInvalidJSONPath, mapping.ExtractPath, part)
			}

			if current == nil {
				return []interface{}{}, nil // Path doesn't exist, return empty
			}
		}
	}

	// Extract the field from each item
	var extracted []interface{}
	switch v := current.(type) {
	case []interface{}:
		// Array of items
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if val, exists := itemMap[mapping.ExtractField]; exists {
					extracted = append(extracted, val)
				}
			}
		}
	case map[string]interface{}:
		// Single object
		if val, exists := v[mapping.ExtractField]; exists {
			extracted = append(extracted, val)
		}
	default:
		return nil, ErrExtractedDataInvalidType
	}

	return extracted, nil
}

// executeTargetBackendRequest sends a request to the target backend with extracted data
func (h *CompositeHandler) executeTargetBackendRequest(ctx context.Context, backend *Backend, mapping *MappingConfig, extractedData []interface{}) (*http.Response, error) {
	// Build the request body
	requestBody := make(map[string]interface{})
	requestBody[mapping.TargetRequestField] = extractedData

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Determine the request path
	requestPath := mapping.TargetRequestPath
	if requestPath == "" {
		requestPath = "/"
	}

	// Determine the HTTP method
	method := mapping.TargetRequestMethod
	if method == "" {
		method = http.MethodPost
	}

	// Build the URL
	backendURL := backend.URL + requestPath

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, backendURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := backend.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// mergeResponses merges source and target responses based on merge strategy
func mergeResponses(sourceBody, targetBody []byte, config *MapReduceConfig) ([]byte, error) {
	var sourceData interface{}
	var targetData interface{}

	if err := json.Unmarshal(sourceBody, &sourceData); err != nil {
		return nil, fmt.Errorf("failed to parse source response: %w", err)
	}

	if err := json.Unmarshal(targetBody, &targetData); err != nil {
		// If target response is not JSON and we allow empty, just return source
		if config.AllowEmptyResponses {
			return sourceBody, nil
		}
		return nil, fmt.Errorf("failed to parse target response: %w", err)
	}

	var result interface{}

	switch config.MergeStrategy {
	case MergeStrategyNested:
		// Create nested structure
		result = map[string]interface{}{
			config.SourceBackend: sourceData,
			config.TargetBackend: targetData,
		}

	case MergeStrategyFlat:
		// Merge into flat structure
		result = make(map[string]interface{})
		if sourceMap, ok := sourceData.(map[string]interface{}); ok {
			for k, v := range sourceMap {
				result.(map[string]interface{})[k] = v
			}
		}
		if targetMap, ok := targetData.(map[string]interface{}); ok {
			for k, v := range targetMap {
				result.(map[string]interface{})[k] = v
			}
		}

	case MergeStrategyEnrich:
		// Enrich source with target data
		result = sourceData
		if sourceMap, ok := sourceData.(map[string]interface{}); ok {
			mergeIntoField := config.MappingConfig.MergeIntoField
			if mergeIntoField == "" {
				mergeIntoField = "enriched_data"
			}
			sourceMap[mergeIntoField] = targetData
			result = sourceMap
		}

	case MergeStrategyJoin:
		// Join strategy is only supported in parallel mode
		return nil, fmt.Errorf("%w: join strategy is only supported in parallel mode", ErrMergeResponseFailed)

	default:
		// Default to nested
		result = map[string]interface{}{
			config.SourceBackend: sourceData,
			config.TargetBackend: targetData,
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMergeResponseFailed, err)
	}
	return encoded, nil
}

// mergeParallelResponses merges responses from parallel backend requests
func mergeParallelResponses(responses map[string][]byte, config *MapReduceConfig) ([]byte, error) {
	if config.MergeStrategy == MergeStrategyJoin {
		return mergeByJoinField(responses, config)
	}

	// For non-join strategies, use simpler merge
	result := make(map[string]interface{})

	for backendID, body := range responses {
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			if config.AllowEmptyResponses {
				continue
			}
			return nil, fmt.Errorf("failed to parse response from %s: %w", backendID, err)
		}

		switch config.MergeStrategy {
		case MergeStrategyFlat:
			if dataMap, ok := data.(map[string]interface{}); ok {
				for k, v := range dataMap {
					result[k] = v
				}
			}
		case MergeStrategyNested, MergeStrategyEnrich, MergeStrategyJoin:
			// For these strategies in parallel mode, treat as nested
			result[backendID] = data
		default:
			result[backendID] = data
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMergeResponseFailed, err)
	}
	return encoded, nil
}

// mergeByJoinField merges responses by joining on a common field
func mergeByJoinField(responses map[string][]byte, config *MapReduceConfig) ([]byte, error) {
	joinField := config.MappingConfig.JoinField
	if joinField == "" {
		return nil, ErrJoinFieldRequired
	}

	// Parse all responses into maps keyed by join field
	backendData := make(map[string]map[interface{}]interface{})

	for backendID, body := range responses {
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			if config.AllowEmptyResponses {
				continue
			}
			return nil, fmt.Errorf("failed to parse response from %s: %w", backendID, err)
		}

		// Extract items and index by join field
		items := make(map[interface{}]interface{})
		switch v := data.(type) {
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if joinValue, exists := itemMap[joinField]; exists {
						items[joinValue] = item
					}
				}
			}
		case map[string]interface{}:
			// Check if there's a data/items field
			if itemsArray, ok := v["items"].([]interface{}); ok {
				for _, item := range itemsArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if joinValue, exists := itemMap[joinField]; exists {
							items[joinValue] = item
						}
					}
				}
			} else if itemsArray, ok := v["data"].([]interface{}); ok {
				for _, item := range itemsArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if joinValue, exists := itemMap[joinField]; exists {
							items[joinValue] = item
						}
					}
				}
			}
		}

		backendData[backendID] = items
	}

	// Merge by join field
	result := make([]interface{}, 0)
	processedKeys := make(map[interface{}]bool)

	// Use the first backend from config as the base (deterministic)
	var baseBackendID string
	if len(config.Backends) > 0 {
		baseBackendID = config.Backends[0]
	} else {
		// Configuration error - backends list is required for deterministic merge
		return nil, fmt.Errorf("%w: config.Backends must not be empty for deterministic merge", ErrMergeResponseFailed)
	}

	// Iterate through base backend items
	for joinValue, baseItem := range backendData[baseBackendID] {
		if processedKeys[joinValue] {
			continue
		}
		processedKeys[joinValue] = true

		// Create merged item
		merged := make(map[string]interface{})

		// Add base item fields
		if baseMap, ok := baseItem.(map[string]interface{}); ok {
			for k, v := range baseMap {
				merged[k] = v
			}
		}

		// Merge fields from other backends
		for backendID, items := range backendData {
			if backendID == baseBackendID {
				continue
			}

			if otherItem, exists := items[joinValue]; exists {
				if otherMap, ok := otherItem.(map[string]interface{}); ok {
					mergeIntoField := config.MappingConfig.MergeIntoField
					if mergeIntoField == "" {
						// Merge directly into the item
						for k, v := range otherMap {
							if k != joinField { // Don't duplicate join field
								merged[k] = v
							}
						}
					} else {
						// Merge into a nested field
						merged[mergeIntoField] = otherItem
					}
				}
			} else if config.FilterOnEmpty {
				// Filter out this item if ancillary data is missing
				merged = nil
				break
			}
		}

		if merged != nil {
			result = append(result, merged)
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMergeResponseFailed, err)
	}
	return encoded, nil
}
