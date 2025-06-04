package circleci

import "time"

// FlakyTest represents a flaky test from CircleCI's API
type FlakyTest struct {
	TestName    string       `json:"test_name"`
	ClassName   string       `json:"classname"`
	TimesFlaky  int          `json:"times_flaky"`
	PipelineRun *PipelineRun `json:"pipeline_run,omitempty"`
}

// PipelineRun represents a pipeline run where the flaky test occurred
type PipelineRun struct {
	WorkflowID string    `json:"workflow_id"`
	PipelineID string    `json:"pipeline_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// FlakyTestResponse represents the response from CircleCI's flaky tests API
type FlakyTestResponse struct {
	FlakyTests    []FlakyTest `json:"flaky-tests"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

// FlakyTestMetric represents analyzed metrics for flaky tests
type FlakyTestMetric struct {
	TestName     string
	ClassName    string
	TimesFlaky   int
	LastOccurred *time.Time // When the test was last flaky
}
