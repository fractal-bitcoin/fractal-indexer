package model

var NeedStop bool
var SkipMissingUTXO bool = false
var MissingUTXO bool

var EnableWAL bool = false // WAL revert data; only needed in once mode (live sync)
