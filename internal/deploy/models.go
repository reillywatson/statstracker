package deploy

import "time"

// DeploymentMetric represents the deployment-to-log latency for a single deployment
type DeploymentMetric struct {
	ReleaseID            string
	ReleaseName          string
	CommitSHA            string
	CommitTime           time.Time
	ReleaseStartTime     time.Time
	FirstLogTime         time.Time
	CommitToLogLatency   time.Duration
	DeploymentSuccessful bool
}
