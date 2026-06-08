package model

import "sync/atomic"

var NeedStop atomic.Bool
var SkipMissingUTXO bool = false
var MissingUTXO bool

var EnableWAL bool = false // WAL revert data; only needed in once mode (live sync)

const (
	IndexModeFull       = "full"
	IndexModeMetricOnly = "metric_only"
)

var IndexMode = IndexModeFull
var MetricsEnabled bool = false

func IsMetricOnly() bool {
	return IndexMode == IndexModeMetricOnly
}

func ShouldIndexBusiness() bool {
	return !IsMetricOnly()
}

func ShouldIndexMetrics() bool {
	return MetricsEnabled || IsMetricOnly()
}
