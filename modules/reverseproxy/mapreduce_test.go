package reverseproxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSequentialMapReduceBasic tests basic sequential map/reduce with simple data
func TestSequentialMapReduceBasic(t *testing.T) {
	// Create source backend that returns a list of conversation IDs
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"conversations": []map[string]interface{}{
				{"id": "conv1", "title": "Conversation 1"},
				{"id": "conv2", "title": "Conversation 2"},
				{"id": "conv3", "title": "Conversation 3"},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer sourceServer.Close()

	// Create target backend that receives IDs and returns follow-up data
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify the request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		
		var requestData map[string]interface{}
		err = json.Unmarshal(body, &requestData)
		require.NoError(t, err)
		
		// Verify the IDs were extracted correctly
		ids, ok := requestData["conversation_ids"].([]interface{})
		require.True(t, ok)
		assert.Len(t, ids, 3)

		// Return follow-up data
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"followups": []map[string]interface{}{
				{"conversation_id": "conv1", "is_followup": true},
				{"conversation_id": "conv3", "is_followup": true},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer targetServer.Close()

	// Create backends
	backends := []*Backend{
		{ID: "conversations", URL: sourceServer.URL, Client: &http.Client{}},
		{ID: "followups", URL: targetServer.URL, Client: &http.Client{}},
	}

	// Create map/reduce config
	config := &MapReduceConfig{
		Type:          MapReduceTypeSequential,
		SourceBackend: "conversations",
		TargetBackend: "followups",
		MappingConfig: MappingConfig{
			ExtractPath:        "conversations",
			ExtractField:       "id",
			TargetRequestField: "conversation_ids",
			TargetRequestPath:  "/bulk",
			TargetRequestMethod: "POST",
			MergeIntoField:     "followup_data",
		},
		MergeStrategy:       MergeStrategyEnrich,
		AllowEmptyResponses: false,
	}

	// Create handler
	handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
	handler.SetMapReduceConfig(config)

	// Create test request
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()

	// Execute the handler
	handler.ServeHTTP(w, req)

	// Verify response
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	require.NoError(t, err)

	// Verify structure
	assert.NotNil(t, responseData["conversations"])
	assert.NotNil(t, responseData["followup_data"])
}

// TestSequentialMapReduceEmptySource tests handling of empty source response
func TestSequentialMapReduceEmptySource(t *testing.T) {
	// Create source backend that returns empty list
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"conversations": []interface{}{},
		})
	}))
	defer sourceServer.Close()

	// Target server shouldn't be called
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Target server should not be called with empty source data")
	}))
	defer targetServer.Close()

	backends := []*Backend{
		{ID: "source", URL: sourceServer.URL, Client: &http.Client{}},
		{ID: "target", URL: targetServer.URL, Client: &http.Client{}},
	}

	config := &MapReduceConfig{
		Type:          MapReduceTypeSequential,
		SourceBackend: "source",
		TargetBackend: "target",
		MappingConfig: MappingConfig{
			ExtractPath:        "conversations",
			ExtractField:       "id",
			TargetRequestField: "ids",
		},
		AllowEmptyResponses: true,
	}

	handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
	handler.SetMapReduceConfig(config)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Should return the source response as-is
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestParallelMapReduceWithJoin tests parallel execution with join strategy
func TestParallelMapReduceWithJoin(t *testing.T) {
	// Backend 1: Returns base conversation data
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := []map[string]interface{}{
			{"id": "1", "title": "Conv 1", "status": "open"},
			{"id": "2", "title": "Conv 2", "status": "open"},
			{"id": "3", "title": "Conv 3", "status": "closed"},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backend1.Close()

	// Backend 2: Returns follow-up info for some conversations
	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := []map[string]interface{}{
			{"id": "1", "is_followup": true, "parent_id": "orig1"},
			{"id": "3", "is_followup": false},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backend2.Close()

	backends := []*Backend{
		{ID: "conversations", URL: backend1.URL, Client: &http.Client{}},
		{ID: "followups", URL: backend2.URL, Client: &http.Client{}},
	}

	config := &MapReduceConfig{
		Type:     MapReduceTypeParallel,
		Backends: []string{"conversations", "followups"},
		MappingConfig: MappingConfig{
			JoinField:      "id",
			MergeIntoField: "followup_info",
		},
		MergeStrategy:       MergeStrategyJoin,
		AllowEmptyResponses: true,
		FilterOnEmpty:       false,
	}

	handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
	handler.SetMapReduceConfig(config)

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var responseData []map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	require.NoError(t, err)

	// Should have 3 items
	assert.Len(t, responseData, 3)

	// Find the item with id "1" and verify it has followup info
	var item1 map[string]interface{}
	for _, item := range responseData {
		if item["id"] == "1" {
			item1 = item
			break
		}
	}
	require.NotNil(t, item1)
	assert.Equal(t, "Conv 1", item1["title"])
	assert.NotNil(t, item1["followup_info"])
}

// TestParallelMapReduceWithFiltering tests filtering when ancillary data is missing
func TestParallelMapReduceWithFiltering(t *testing.T) {
	// Backend 1: Returns all conversations
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := []map[string]interface{}{
			{"id": "1", "title": "Conv 1"},
			{"id": "2", "title": "Conv 2"},
			{"id": "3", "title": "Conv 3"},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backend1.Close()

	// Backend 2: Returns data for only some conversations
	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := []map[string]interface{}{
			{"id": "1", "followup": true},
			// Missing id "2" intentionally
			{"id": "3", "followup": false},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backend2.Close()

	backends := []*Backend{
		{ID: "base", URL: backend1.URL, Client: &http.Client{}},
		{ID: "ancillary", URL: backend2.URL, Client: &http.Client{}},
	}

	config := &MapReduceConfig{
		Type:     MapReduceTypeParallel,
		Backends: []string{"base", "ancillary"},
		MappingConfig: MappingConfig{
			JoinField: "id",
		},
		MergeStrategy:       MergeStrategyJoin,
		AllowEmptyResponses: false,
		FilterOnEmpty:       true, // Filter out items without ancillary data
	}

	handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
	handler.SetMapReduceConfig(config)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var responseData []map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	require.NoError(t, err)

	// Should only have 2 items (id "2" filtered out)
	assert.Len(t, responseData, 2)

	// Verify we have id 1 and 3
	ids := make(map[string]bool)
	for _, item := range responseData {
		ids[item["id"].(string)] = true
	}
	assert.True(t, ids["1"])
	assert.False(t, ids["2"]) // Should be filtered out
	assert.True(t, ids["3"])
}

// TestComplexNestedResponse tests map/reduce with complex nested JSON structures
func TestComplexNestedResponse(t *testing.T) {
	// Source backend with nested data structure
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":   "item1",
						"name": "First Item",
						"metadata": map[string]interface{}{
							"created": "2024-01-01",
						},
					},
					{
						"id":   "item2",
						"name": "Second Item",
						"metadata": map[string]interface{}{
							"created": "2024-01-02",
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer sourceServer.Close()

	// Target backend
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return the enrichment data directly (not wrapped)
		response := map[string]interface{}{
			"processed": true,
			"count":     2,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer targetServer.Close()

	backends := []*Backend{
		{ID: "source", URL: sourceServer.URL, Client: &http.Client{}},
		{ID: "target", URL: targetServer.URL, Client: &http.Client{}},
	}

	config := &MapReduceConfig{
		Type:          MapReduceTypeSequential,
		SourceBackend: "source",
		TargetBackend: "target",
		MappingConfig: MappingConfig{
			ExtractPath:        "data.items",
			ExtractField:       "id",
			TargetRequestField: "item_ids",
			MergeIntoField:     "enrichment",
		},
		MergeStrategy:       MergeStrategyEnrich,
		AllowEmptyResponses: false,
	}

	handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
	handler.SetMapReduceConfig(config)

	req := httptest.NewRequest("GET", "/api/items", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	require.NoError(t, err)

	// Verify original data is present
	assert.NotNil(t, responseData["data"])
	
	// Verify enrichment was added
	assert.NotNil(t, responseData["enrichment"])
	enrichment := responseData["enrichment"].(map[string]interface{})
	assert.Equal(t, true, enrichment["processed"])
}

// TestMapReduceErrorHandling tests error scenarios
func TestMapReduceErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		sourceStatus   int
		sourceBody     string
		targetStatus   int
		allowEmpty     bool
		expectedStatus int
	}{
		{
			name:           "Source backend error",
			sourceStatus:   http.StatusInternalServerError,
			sourceBody:     `{"error": "internal error"}`,
			targetStatus:   http.StatusOK,
			allowEmpty:     false,
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "Source backend returns non-JSON",
			sourceStatus:   http.StatusOK,
			sourceBody:     "invalid json",
			targetStatus:   http.StatusOK,
			allowEmpty:     false,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Target backend error with AllowEmptyResponses",
			sourceStatus:   http.StatusOK,
			sourceBody:     `{"items": [{"id": "1"}]}`,
			targetStatus:   http.StatusInternalServerError,
			allowEmpty:     true,
			expectedStatus: http.StatusOK, // Should return source response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.sourceStatus)
				_, _ = w.Write([]byte(tt.sourceBody))
			}))
			defer sourceServer.Close()

			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.targetStatus)
				_, _ = w.Write([]byte(`{"data": "target"}`))
			}))
			defer targetServer.Close()

			backends := []*Backend{
				{ID: "source", URL: sourceServer.URL, Client: &http.Client{}},
				{ID: "target", URL: targetServer.URL, Client: &http.Client{}},
			}

			config := &MapReduceConfig{
				Type:          MapReduceTypeSequential,
				SourceBackend: "source",
				TargetBackend: "target",
				MappingConfig: MappingConfig{
					ExtractPath:        "items",
					ExtractField:       "id",
					TargetRequestField: "ids",
				},
				AllowEmptyResponses: tt.allowEmpty,
			}

			handler := NewCompositeHandler(backends, StrategyMapReduce, 30*time.Second)
			handler.SetMapReduceConfig(config)

			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestExtractDataFromResponse tests the data extraction utility
func TestExtractDataFromResponse(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		extractPath    string
		extractField   string
		expectedCount  int
		expectedValues []interface{}
		expectError    bool
	}{
		{
			name:           "Simple array extraction",
			body:           `{"items": [{"id": "1"}, {"id": "2"}]}`,
			extractPath:    "items",
			extractField:   "id",
			expectedCount:  2,
			expectedValues: []interface{}{"1", "2"},
			expectError:    false,
		},
		{
			name:           "Nested path extraction",
			body:           `{"data": {"items": [{"id": "a"}, {"id": "b"}, {"id": "c"}]}}`,
			extractPath:    "data.items",
			extractField:   "id",
			expectedCount:  3,
			expectedValues: []interface{}{"a", "b", "c"},
			expectError:    false,
		},
		{
			name:          "Empty array",
			body:          `{"items": []}`,
			extractPath:   "items",
			extractField:  "id",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Missing path",
			body:          `{"other": [{"id": "1"}]}`,
			extractPath:   "items",
			extractField:  "id",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "Invalid JSON",
			body:        `invalid json`,
			extractPath: "items",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MappingConfig{
				ExtractPath:  tt.extractPath,
				ExtractField: tt.extractField,
			}

			result, err := extractDataFromResponse([]byte(tt.body), config)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)

			if tt.expectedValues != nil {
				for i, expected := range tt.expectedValues {
					assert.Equal(t, expected, result[i])
				}
			}
		})
	}
}

// TestMergeStrategies tests different merge strategies
func TestMergeStrategies(t *testing.T) {
	sourceBody := []byte(`{"name": "test", "value": 123}`)
	targetBody := []byte(`{"extra": "data", "count": 456}`)

	tests := []struct {
		name          string
		strategy      MergeStrategy
		verifyFunc    func(t *testing.T, result map[string]interface{})
	}{
		{
			name:     "Nested strategy",
			strategy: MergeStrategyNested,
			verifyFunc: func(t *testing.T, result map[string]interface{}) {
				assert.NotNil(t, result["source"])
				assert.NotNil(t, result["target"])
			},
		},
		{
			name:     "Flat strategy",
			strategy: MergeStrategyFlat,
			verifyFunc: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "test", result["name"])
				assert.Equal(t, float64(123), result["value"])
				assert.Equal(t, "data", result["extra"])
				assert.Equal(t, float64(456), result["count"])
			},
		},
		{
			name:     "Enrich strategy",
			strategy: MergeStrategyEnrich,
			verifyFunc: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "test", result["name"])
				assert.Equal(t, float64(123), result["value"])
				assert.NotNil(t, result["enriched_data"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MapReduceConfig{
				SourceBackend: "source",
				TargetBackend: "target",
				MergeStrategy: tt.strategy,
				MappingConfig: MappingConfig{
					MergeIntoField: "enriched_data",
				},
			}

			result, err := mergeResponses(sourceBody, targetBody, config)
			require.NoError(t, err)

			var resultData map[string]interface{}
			err = json.Unmarshal(result, &resultData)
			require.NoError(t, err)

			tt.verifyFunc(t, resultData)
		})
	}
}
