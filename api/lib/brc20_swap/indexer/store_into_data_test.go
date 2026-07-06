package indexer

import (
	"bytes"
	"encoding/gob"
	"testing"

	"fractal-indexer/api/lib/brc20_swap/model"
)

func TestBRC20ModuleStoreGobLegacyHistoryData(t *testing.T) {
	store := &BRC20ModuleIndexerStore{
		AllModulesWithdrawHistory: []*model.BRC20ModuleHistory{
			{Data: &model.BRC20SwapHistoryWithdrawData{Tick: "ordi", Amount: "12"}},
			{Data: &model.BRC20SwapHistoryCommitData{FunctionsTotal: 3, InvalidList: []int{1}}},
		},
	}

	registerBRC20ModuleStoreGobTypes()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(store); err != nil {
		t.Fatalf("encode store: %v", err)
	}

	var decoded BRC20ModuleIndexerStore
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode store: %v", err)
	}

	if _, ok := decoded.AllModulesWithdrawHistory[0].Data.(model.BRC20SwapHistoryWithdrawData); !ok {
		t.Fatalf("withdraw data type = %T", decoded.AllModulesWithdrawHistory[0].Data)
	}
	if _, ok := decoded.AllModulesWithdrawHistory[1].Data.(model.BRC20SwapHistoryCommitData); !ok {
		t.Fatalf("commit data type = %T", decoded.AllModulesWithdrawHistory[1].Data)
	}
}
