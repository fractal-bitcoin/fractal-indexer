package service

import (
	"testing"

	indexerConstant "fractal-indexer/constant"
)

func TestGetBlockMetricsTimeEstimateConfig(t *testing.T) {
	tests := []struct {
		name          string
		chainType     string
		genesisUnixMs int64
		blockMs       int64
	}{
		{
			name:          "fractal",
			chainType:     indexerConstant.CHAIN_TYPE_FRACTAL,
			genesisUnixMs: 1725840000000,
			blockMs:       30000,
		},
		{
			name:          "btc",
			chainType:     indexerConstant.CHAIN_TYPE_BTC,
			genesisUnixMs: 1231006505000,
			blockMs:       600000,
		},
	}

	previousChainType := indexerConstant.CHAIN_TYPE
	t.Cleanup(func() {
		indexerConstant.CHAIN_TYPE = previousChainType
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexerConstant.CHAIN_TYPE = tt.chainType

			got := GetBlockMetricsTimeEstimateConfig()
			if got.GenesisUnixMs != tt.genesisUnixMs {
				t.Fatalf("GenesisUnixMs = %d, want %d", got.GenesisUnixMs, tt.genesisUnixMs)
			}
			if got.BlockMs != tt.blockMs {
				t.Fatalf("BlockMs = %d, want %d", got.BlockMs, tt.blockMs)
			}
		})
	}
}
