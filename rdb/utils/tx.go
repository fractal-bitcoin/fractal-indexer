package utils

import (
	"context"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/rdb"
	"sort"
	"sync"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// UpdateUtxoInPika batch-updates Redis UTXOs.
func UpdateUtxoInPikaDel(utxoToRemove []string) bool {
	// logger.Log.Info("UpdateUtxoInPikaDel",
	// 	zap.Int("del", len(utxoToRemove)))

	ctx := context.Background()

	if len(utxoToRemove) > 0 {
		outpointKeys := utxoToRemove

		sort.Strings(outpointKeys)

		sliceLen := 10000
		for idx := 0; idx < (len(outpointKeys)-1)/sliceLen+1; idx++ {

			pikaPipe := rdb.RdbClient.Pipeline()
			n := 0
			for _, outpointKey := range outpointKeys[idx*sliceLen:] {
				if n == sliceLen {
					break
				}
				// Clear global Redis UTXO data.
				pikaPipe.Del(ctx, "u"+model.TrimOutpointKey(outpointKey))
				n++
			}
			if _, err := pikaPipe.Exec(ctx); err != nil && err != redis.Nil {
				logger.Log.Error("pika delete exec failed", zap.Error(err))
				pikaPipe.Close()
				return false
			}
			pikaPipe.Close()
		}
	}

	return true
}

// UpdateUtxoInPika batch-updates Redis UTXOs.
func UpdateUtxoInPikaAdd(utxoToRestore map[string]*model.TxoData) bool {
	// logger.Log.Info("UpdateUtxoInPikaAdd",
	// 	zap.Int("add", len(utxoToRestore)),
	// )

	type Pair struct {
		Outpoint string
		Utxo     []byte
	}

	// marshalBuf is a reusable scratch buffer; result is copied to an exact-size allocation
	// so that Pair.Utxo does not retain a reference to the (larger) scratch buffer.
	var marshalBuf []byte
	utxoBufToRestore := make([]*Pair, 0, len(utxoToRestore))
	for outpointKey, data := range utxoToRestore {
		// Standard dust UTXOs (P2WPKH=294, P2TR=330, P2PKH=546, no inscriptions) are
		// intentionally omitted from Pika. The read path infers satoshi from input scripts.
		if len(data.CreatePointOfNFTs) == 0 && scriptDecoder.ClassifyStandardDust(data.ScriptType, data.Satoshi) {
			continue
		}
		size := data.MarshalBufSize()
		if cap(marshalBuf) < size {
			marshalBuf = make([]byte, size)
		} else {
			marshalBuf = marshalBuf[:size]
		}
		zbuf, _ := data.Marshal(marshalBuf)
		result := make([]byte, len(zbuf))
		copy(result, zbuf)
		utxoBufToRestore = append(utxoBufToRestore, &Pair{
			Outpoint: outpointKey,
			Utxo:     result,
		})
	}

	if len(utxoBufToRestore) == 0 {
		return true
	}

	success := true

	const numWorkers = 8
	txChan := make(chan *Pair, 128)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()

			ctx := context.Background()

			maxSize := 10000000 // 10MB
			done := false
			for !done {
				done = true

				size := 0
				pikaPipe := rdb.RdbClient.Pipeline()
				for utxoPair := range txChan {
					// Add global Redis UTXO data so later spent inputs can be associated; record it whether or not the address is recognized.
					pikaPipe.Set(ctx, "u"+model.TrimOutpointKey(utxoPair.Outpoint), utxoPair.Utxo, 0)

					size += 36 + len(utxoPair.Utxo)
					if size >= maxSize {
						done = false
						break
					}
				}
				if _, err := pikaPipe.Exec(ctx); err != nil && err != redis.Nil {
					logger.Log.Error("pika utxo exec failed", zap.Int("size", size), zap.Error(err))
					success = false
					pikaPipe.Close()
				}
				pikaPipe.Close()
			}
		}()
	}

	for _, tx := range utxoBufToRestore {
		txChan <- tx
	}
	close(txChan)
	wg.Wait()

	if !success {
		return false
	}

	return true
}

// UpdateUtxoInPikaAddRaw writes pre-marshaled UTXO data to pika, skipping Marshal.
func UpdateUtxoInPikaAddRaw(utxoToRestore map[string][]byte) bool {
	if len(utxoToRestore) == 0 {
		return true
	}

	type Pair struct {
		Outpoint string
		Utxo     []byte
	}

	utxoBufToRestore := make([]*Pair, 0, len(utxoToRestore))
	for outpointKey, data := range utxoToRestore {
		utxoBufToRestore = append(utxoBufToRestore, &Pair{
			Outpoint: outpointKey,
			Utxo:     data,
		})
	}

	success := true

	const numWorkers = 8
	txChan := make(chan *Pair, 128)
	var wg2 sync.WaitGroup
	wg2.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg2.Done()

			ctx := context.Background()

			maxSize := 10000000 // 10MB
			done := false
			for !done {
				done = true

				size := 0
				pikaPipe := rdb.RdbClient.Pipeline()
				for utxoPair := range txChan {
					pikaPipe.Set(ctx, "u"+model.TrimOutpointKey(utxoPair.Outpoint), utxoPair.Utxo, 0)

					size += 36 + len(utxoPair.Utxo)
					if size >= maxSize {
						done = false
						break
					}
				}
				if _, err := pikaPipe.Exec(ctx); err != nil && err != redis.Nil {
					logger.Log.Error("pika utxo raw exec failed", zap.Error(err))
					success = false
					pikaPipe.Close()
				}
				pikaPipe.Close()
			}
		}()
	}

	for _, p := range utxoBufToRestore {
		txChan <- p
	}
	close(txChan)
	wg2.Wait()

	if !success {
		return false
	}
	return true
}
