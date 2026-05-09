package service

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg"
)

var (
	ctx             = context.Background()
	GlobalNetParams = &chaincfg.MainNetParams
)
