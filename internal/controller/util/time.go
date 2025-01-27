package util

import "time"

// Define constants for the requeue timings
const (
	ErrorRequeueTime      = 30 * time.Second
	NotStartedRequeueTime = 30 * time.Second
	InProgressRequeueTime = 30 * time.Second
	CompletedRequeueTime  = 60 * time.Second
	SuccessfulRequeueTime = 120 * time.Second
	FailedRequeueTime     = 120 * time.Second
)
