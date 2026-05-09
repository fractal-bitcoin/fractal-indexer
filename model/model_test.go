package model_test

import (
	"bytes"
	"encoding/json"
	"fractal-indexer/model"
	"math/rand"
	"testing"
	"time"
)

// Assume we have a function whose performance needs testing.
func randomSum(n int) int {
	rand.Seed(time.Now().UnixNano()) // ensures a different seed and random sequence on each run
	sum := 0
	for i := 0; i < n; i++ {
		sum += rand.Intn(100) // generates a random integer between 0 and 99
	}
	return sum
}

// Defines the benchmark function.
func BenchmarkRandomSum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		randomSum(100) // calls the function under test
	}
}

func BenchmarkTxo(b *testing.B) {
	var data *model.TxoData = &model.TxoData{
		BlockHeight: 21000,
		TxIdx:       256000,
		Satoshi:     256003,
		PkScript:    []byte("................................................................"),
		CreatePointOfNFTs: []model.NFTCreatePoint{
			{
				IsStrip:    true,
				Height:     21000,
				IdxInBlock: 130,
				Offset:     256000,
				IsText:     true,
			},
		},
	}

	buf := data.MakeMarshalBuf()
	zbuf, _ := data.Marshal(buf)
	d := &model.TxoData{}

	for i := 0; i < b.N; i++ {
		// data.Marshal(buf)
		model.DumpNFTCreatePoints(buf, data.CreatePointOfNFTs)
	}
	d.Unmarshal(zbuf)
}

func TestTxo(t *testing.T) {
	var data *model.TxoData = &model.TxoData{
		UTxid: []byte(",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,"),
		Vout:  3,

		BlockHeight:       1,
		TxIdx:             2,
		Satoshi:           3,
		PkScript:          []byte("0000000000000000000000000000000000000000000000000000000000000000"),
		CreatePointOfNFTs: []model.NFTCreatePoint{},
	}

	nSize := 4000 * 10000
	for idx := 0; idx < nSize; idx++ {
		data.CreatePointOfNFTs = append(data.CreatePointOfNFTs, model.NFTCreatePoint{
			IsStrip:    false,
			Height:     22001 + uint32(idx)%500000,
			IdxInBlock: 132 + uint32(idx)%1000,
			Offset:     100000 + uint64(idx)*(546+uint64(idx)%2),
			IsText:     true,
			IsBRC20:    true,
		})
	}

	buf := data.MakeMarshalBuf()
	zbuf, size := data.Marshal(buf)
	t.Log("len size: ", size)
	t.Log("len zbuf: ", len(zbuf))

	d := &model.TxoData{}
	ok := d.Unmarshal(zbuf)
	if !ok {
		t.Logf("unmarshal not ok")
	}

	if len(data.CreatePointOfNFTs) != len(d.CreatePointOfNFTs) {
		t.Errorf("nft not match %d != %d", len(data.CreatePointOfNFTs), len(d.CreatePointOfNFTs))
		return
	}

	for idx := 0; idx < nSize; idx++ {
		if data.CreatePointOfNFTs[idx].IsStrip != d.CreatePointOfNFTs[idx].IsStrip {
			t.Errorf("prune not match")
			break
		}

		if data.CreatePointOfNFTs[idx].Offset != d.CreatePointOfNFTs[idx].Offset {
			t.Errorf("offset not match")
			break
		}

		if data.CreatePointOfNFTs[idx].IsStrip {
			continue
		}
		if data.CreatePointOfNFTs[idx].Height != d.CreatePointOfNFTs[idx].Height {
			t.Errorf("height not match")
			break
		}

		if data.CreatePointOfNFTs[idx].IdxInBlock != d.CreatePointOfNFTs[idx].IdxInBlock {
			t.Errorf("idx not match")
			break
		}

		if data.CreatePointOfNFTs[idx].Sequence != d.CreatePointOfNFTs[idx].Sequence {
			t.Errorf("seq not match")
			break
		}

	}
}

type InscriptionNamePick struct {
	Proto     string `json:"p"`
	Operation string `json:"op"`

	BRC20Tick string `json:"tick"` // brc20
}

func isBRC20(nftContentBody []byte) (base, ext, mint, transfer bool) {
	if len(nftContentBody) < 40 {
		return
	}

	content := bytes.TrimSpace(nftContentBody)
	if !bytes.HasPrefix(content, []byte("{")) {
		return
	}
	if !bytes.HasSuffix(content, []byte("}")) {
		return
	}

	var namePick InscriptionNamePick
	if err := json.Unmarshal(nftContentBody, &namePick); err != nil {
		return
	}
	if namePick.Proto == "brc-20" {
		if namePick.BRC20Tick == "" {
			return
		}
		base = true
		if namePick.Operation == "mint" {
			mint = true
			return
		}
		if namePick.Operation == "transfer" {
			transfer = true
			return
		}
		return
	}

	if namePick.Proto == "brc20-module" {
		base = true
		return
	}

	if namePick.Proto == "brc20-swap" {
		base = true
		if namePick.Operation == "conditional-approve" {
			ext = true
			return
		}
		return
	}

	// Does not explicitly block content_encoding / delegate.
	return
}

func TestBrc20(t *testing.T) {
	base, ext, mint, transfer := isBRC20([]byte(`{"p":"brc-20","op":"mint","tick":"Nikola","amt":"3693"}`))

	t.Errorf("%v, %v, %v, %v", base, ext, mint, transfer)
}
