package circleci

import (
	"testing"
	"time"
)

func TestProcessFlakyTests(t *testing.T) {
	// Create test data
	now := time.Now()

	tests := []FlakyTest{
		{
			TestName:   "TestAlwaysFails",
			ClassName:  "com.example.MyTestClass",
			TimesFlaky: 5,
			PipelineRun: &PipelineRun{
				WorkflowID: "workflow-1",
				PipelineID: "pipeline-1",
				CreatedAt:  now.Add(-1 * time.Hour),
			},
		},
		{
			TestName:   "TestSometimesFails",
			ClassName:  "com.example.AnotherTestClass",
			TimesFlaky: 2,
			PipelineRun: &PipelineRun{
				WorkflowID: "workflow-2",
				PipelineID: "pipeline-2",
				CreatedAt:  now.Add(-2 * time.Hour),
			},
		},
		{
			TestName:    "TestRarelyFails",
			ClassName:   "com.example.StableTestClass",
			TimesFlaky:  1,
			PipelineRun: nil, // No pipeline run information
		},
	}

	metrics := ProcessFlakyTests(tests)

	// Should return the same number of metrics as input tests
	if len(metrics) != len(tests) {
		t.Errorf("Expected %d metrics, got %d", len(tests), len(metrics))
	}

	// Should be sorted by TimesFlaky in descending order
	if len(metrics) >= 2 {
		if metrics[0].TimesFlaky < metrics[1].TimesFlaky {
			t.Errorf("Expected metrics to be sorted by TimesFlaky in descending order")
		}
	}

	// Check the first metric (should be the most flaky)
	if metrics[0].TestName != "TestAlwaysFails" {
		t.Errorf("Expected first metric to be 'TestAlwaysFails', got '%s'", metrics[0].TestName)
	}

	if metrics[0].TimesFlaky != 5 {
		t.Errorf("Expected first metric to have TimesFlaky=5, got %d", metrics[0].TimesFlaky)
	}

	if metrics[0].LastOccurred == nil {
		t.Errorf("Expected first metric to have LastOccurred set")
	}

	// Check the last metric (should have no pipeline run info)
	lastMetric := metrics[len(metrics)-1]
	if lastMetric.TestName != "TestRarelyFails" {
		t.Errorf("Expected last metric to be 'TestRarelyFails', got '%s'", lastMetric.TestName)
	}

	if lastMetric.LastOccurred != nil {
		t.Errorf("Expected last metric to have LastOccurred=nil")
	}
}
