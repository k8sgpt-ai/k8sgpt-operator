package util

import "time"

// Define constants for the requeue timings
const (
	ErrorRequeueTime      = 30 * time.Second
	NotStartedRequeueTime = 15 * time.Second
	InProgressRequeueTime = 30 * time.Second
	CompletedRequeueTime  = 30 * time.Second
	SuccessfulRequeueTime = 60 * time.Second
	PendingRequeueTime    = 30 * time.Second
	FailedRequeueTime     = 120 * time.Second
)
