package parallel

import (
	"context"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/rdb"

	redis "github.com/go-redis/redis/v8"
)

// ParseTxFirst first analyzes transactions in parallel: different blocks run in parallel, while work within the same block is serial.
func ParseTxFirst(tx *model.Tx, isCoinbase bool, block *model.ProcessBlock) {
	// batch-allocate all outpoint keys in one slab: 1 alloc instead of M
	outBuf := make([]byte, len(tx.TxOuts)*model.OutpointKeySize)
	for idx := range tx.TxOuts {
		off := idx * model.OutpointKeySize
		model.MakeOutpointKey(outBuf[off:], tx.TxId, uint32(idx))
		tx.TxOuts[idx].OutpointKey = string(outBuf[off : off+model.OutpointKeySize])

		tx.TxOuts[idx].ScriptType = scriptDecoder.GetLockingScriptType(tx.TxOuts[idx].PkScript)

		if scriptDecoder.IsOpreturn(tx.TxOuts[idx].ScriptType) {
			tx.TxOuts[idx].LockingScriptUnspendable = true
		}
	}
}

// ParseUpdateTxoSpendByTxParallel marks UTXOs as spent.
func ParseUpdateTxoSpendByTxParallel(tx *model.Tx, isCoinbase bool, block *model.ProcessBlock) {
	if isCoinbase {
		return
	}
	for _, input := range tx.TxIns {
		block.SpentUtxoKeysMap[input.InputOutpointKey] = struct{}{}
	}
}

// ParseGetSpentUtxoDataFromRedisParallel fetches, deserializes, and caches spent UTXO data
// from Redis for all keys not satisfied by block-internal self-spend. Runs inside each
// block's parallel goroutine so multiple blocks pipeline concurrently.
//
// When Pika returns nil for an outpoint, the function attempts to infer the satoshi from
// the spending input's scriptSig/witness (standard dust: P2WPKH=294, P2TR=330, P2PKH=546).
// Inferred entries are marked with IsStandardDustStub=true and never trigger a Pika DEL.
func ParseGetSpentUtxoDataFromRedisParallel(block *model.Block) {
	parseData := block.ParseData
	n := len(parseData.SpentUtxoKeysMap)

	type inputMeta struct {
		outpointKey string
		scriptSig   []byte
		witness     []byte
	}

	cmds := make([]*redis.StringCmd, 0, n)
	metas := make([]inputMeta, 0, n)

	pipe := rdb.RdbClient.Pipeline()
	ctx := context.Background()

	for txIdx := range block.Txs {
		if txIdx == 0 {
			continue // coinbase inputs spend nothing
		}
		tx := &block.Txs[txIdx]
		for i := range tx.TxIns {
			input := &tx.TxIns[i]
			outpointKey := input.InputOutpointKey

			if _, ok := parseData.NewUtxoDataMap[outpointKey]; ok {
				continue // block-internal self-spend, resolved in serial stage
			}
			cmds = append(cmds, pipe.Get(ctx, "u"+model.TrimOutpointKey(outpointKey)))
			metas = append(metas, inputMeta{outpointKey, input.ScriptSig, input.ScriptWitness})
		}
	}

	if len(cmds) == 0 {
		pipe.Close()
		return
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		pipe.Close()
		panic(err)
	}
	pipe.Close()

	// Deserialize hits; for misses, attempt standard-dust inference from input scripts.
	slab := make([]model.TxoData, len(cmds))
	fetched := make(map[string]*model.TxoData, len(cmds))
	for i, cmd := range cmds {
		res, err := cmd.Result()
		if err == redis.Nil {
			// Not in Pika: infer satoshi from spending input's scriptSig/witness.
			// This succeeds only for standard dust UTXOs intentionally omitted at write time.
			sat, ok := scriptDecoder.InferDustFromInput(metas[i].scriptSig, metas[i].witness)
			if ok {
				d := &slab[i]
				d.Satoshi = sat
				d.IsStandardDustStub = true
				fetched[metas[i].outpointKey] = d
			}
			// If inference fails, the entry is absent from fetched; serial stage will error.
			continue
		}
		if err != nil {
			continue
		}
		d := &slab[i]
		d.Unmarshal([]byte(res))
		d.ScriptType = scriptDecoder.GetLockingScriptType(d.PkScript)
		fetched[metas[i].outpointKey] = d
	}
	parseData.SpentUtxoPrefetch = fetched
}

// ParseUpdateNewUtxoInTxParallel handles UTXO data; output initialization lacks NFT data, which is filled after querying UTXOs and before writing input DB data.
func ParseUpdateNewUtxoInTxParallel(txIdx uint32, tx *model.Tx, block *model.ProcessBlock, txoSlab []model.TxoData) {
	slabIdx := 0
	for _, output := range tx.TxOuts {
		if output.LockingScriptUnspendable && output.Satoshi == 0 {
			slabIdx++
			continue
		}

		// use pre-allocated slab element instead of new(TxoData)
		d := &txoSlab[slabIdx]
		slabIdx++
		d.BlockHeight = block.Height
		d.TxIdx = txIdx
		d.Satoshi = output.Satoshi
		d.ScriptType = output.ScriptType
		d.PkScript = output.PkScript

		block.NewUtxoDataMap[output.OutpointKey] = d
	}
}
