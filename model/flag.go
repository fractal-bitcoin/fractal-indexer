package model

import "sync/atomic"

var NeedStop atomic.Bool
var SkipMissingUTXO bool = false
var MissingUTXO bool

var EnableWAL bool = false // WAL revert data; only needed in once mode (live sync)
