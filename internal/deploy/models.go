package deploy

import "time"

// DeploymentMetric represents the commit-to-deploy latency for a single deployment
type DeploymentMetric struct {
	ReleaseID             string
	ReleaseName           string
	CommitSHA             string
	PRNumber              string // PR number from pull-<number>_<SHA> format, empty if not a PR deployment
	CommitTime            time.Time
	ReleaseStartTime      time.Time
	ReleaseFinishTime     time.Time // Time when the last rollout completed
	CommitToDeployLatency time.Duration
	DeploymentSuccessful  bool
}

// PRDeploymentStats represents statistics for deployments of a specific PR
type PRDeploymentStats struct {
	PRNumber         string
	DeploymentCount  int
	FirstCommitTime  time.Time
	LastFinishTime   time.Time
	FirstToLastDelta time.Duration
	CommitSHAs       []string           // All unique commit SHAs deployed for this PR
	Deployments      []DeploymentMetric // All deployments for this PR
}
