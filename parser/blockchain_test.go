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

func TestMetricReplayCommonBlockID(t *testing.T) {
	bc := &Blockchain{}
	bc.resetBlockIndexCaches()
	bc.applyBlockIndexInfos([]*loader.BlockIndexInfo{
		{Height: 799999, HashHex: "block799999"},
		{Height: 800000, HashHex: "block800000"},
	})

	got, ok := bc.metricReplayCommonBlockID(800000)
	if !ok {
		t.Fatal("expected common block")
	}
	if got != "block799999" {
		t.Fatalf("common block mismatch: got %q", got)
	}
}

func TestMetricReplayCommonBlockIDMissing(t *testing.T) {
	bc := &Blockchain{}
	bc.resetBlockIndexCaches()
	bc.applyBlockIndexInfos([]*loader.BlockIndexInfo{
		{Height: 800000, HashHex: "block800000"},
	})

	if got, ok := bc.metricReplayCommonBlockID(800000); ok || got != "" {
		t.Fatalf("unexpected common block: got=%q ok=%v", got, ok)
	}
}
