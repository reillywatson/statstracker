package circleci

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCircleCIClient_FetchFlakyTests(t *testing.T) {
	// Create a mock server that returns sample flaky test data
	mockResponse := FlakyTestResponse{
		FlakyTests: []FlakyTest{
			{
				TestName:   "TestFlaky1",
				ClassName:  "com.example.MyTestClass",
				TimesFlaky: 3,
				PipelineRun: &PipelineRun{
					WorkflowID: "workflow-123",
					PipelineID: "pipeline-456",
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				},
			},
			{
				TestName:   "TestFlaky2",
				ClassName:  "com.example.AnotherClass",
				TimesFlaky: 1,
				PipelineRun: &PipelineRun{
					WorkflowID: "workflow-789",
					PipelineID: "pipeline-012",
					CreatedAt:  time.Now().Add(-2 * time.Hour),
				},
			},
		},
		NextPageToken: "",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		expectedPath := "/insights/gh/test-org/test-repo/flaky-tests"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify auth header
		authHeader := r.Header.Get("Circle-Token")
		if authHeader != "test-token" {
			t.Errorf("Expected Circle-Token header to be 'test-token', got '%s'", authHeader)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewCircleCIClient("test-token")
	client.baseURL = server.URL

	// Test fetching flaky tests
	ctx := context.Background()
	tests, err := client.FetchFlakyTests(ctx, "test-org", "test-repo")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tests) != 2 {
		t.Fatalf("Expected 2 tests, got %d", len(tests))
	}

	// Verify first test
	if tests[0].TestName != "TestFlaky1" {
		t.Errorf("Expected first test name to be 'TestFlaky1', got '%s'", tests[0].TestName)
	}

	if tests[0].TimesFlaky != 3 {
		t.Errorf("Expected first test to be flaky 3 times, got %d", tests[0].TimesFlaky)
	}

	if tests[0].PipelineRun == nil {
		t.Errorf("Expected first test to have pipeline run information")
	}
}

func TestCircleCIClient_FetchFlakyTests_APIError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Project not found"))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewCircleCIClient("test-token")
	client.baseURL = server.URL

	// Test fetching flaky tests with error
	ctx := context.Background()
	tests, err := client.FetchFlakyTests(ctx, "nonexistent-org", "nonexistent-repo")

	if err == nil {
		t.Fatalf("Expected error, got none")
	}

	if tests != nil {
		t.Errorf("Expected nil tests on error, got %v", tests)
	}

	// Check that error message contains status code
	expectedError := "API returned status 404"
	if err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error to start with '%s', got '%s'", expectedError, err.Error())
	}
}
