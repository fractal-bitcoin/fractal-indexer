package parser

import (
	"fractal-indexer/loader"
	"testing"
)

func TestApplyBlockIndexInfosInitializesCaches(t *testing.T) {
	bc := &Blockchain{}
	bc.resetBlockIndexCaches()

	bc.applyBlockIndexInfos([]*loader.BlockIndexInfo{
		{Height: 100, HashHex: "block100"},
		{Height: 101, HashHex: "block101"},
	})

	if _, ok := bc.BlocksOfChainById["block100"]; !ok {
		t.Fatalf("block id cache missing first block")
	}
	if got := bc.BlocksOfChainByHeight[101]; got == nil || got.HashHex != "block101" {
		t.Fatalf("height cache mismatch: %#v", got)
	}
	if got := bc.Blocks["block101"]; got == nil || got.ParentHex != "block100" {
		t.Fatalf("block parent mismatch: %#v", got)
	}
	if bc.MainChainHeight != 101 || bc.MainChainBlockIdHex != "block101" {
		t.Fatalf("main chain tip mismatch: height=%d hash=%q", bc.MainChainHeight, bc.MainChainBlockIdHex)
	}
}
