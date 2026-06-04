package loader

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"fractal-indexer/logger"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"github.com/ybbus/jsonrpc"
	"go.uber.org/zap"
)

var ChainConf string = "conf/chain.yaml"

var rpcClient jsonrpc.RPCClient

const (
	rpcClientTimeout = 2 * time.Minute
	rpcRetryAttempts = 3
	rpcRetrySleep    = 5 * time.Second
)

func InitRpc() {
	viper.SetConfigFile(ChainConf)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	rpcAddress := viper.GetString("rpc")
	rpcAuth := viper.GetString("rpc_auth")
	rpcClient = jsonrpc.NewClientWithOpts(rpcAddress, &jsonrpc.RPCClientOpts{
		HTTPClient: &http.Client{
			Timeout: rpcClientTimeout,
		},
		CustomHeaders: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(rpcAuth)),
		},
	})
}

func callRPCWithRetry(method string, params ...interface{}) (*jsonrpc.RPCResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= rpcRetryAttempts; attempt++ {
		response, err := rpcClient.Call(method, params...)
		if err == nil {
			if response != nil && response.Error == nil {
				return response, nil
			}
			if response != nil {
				err = response.Error
			} else {
				err = fmt.Errorf("nil RPC response")
			}
		}
		lastErr = err
		if attempt == rpcRetryAttempts {
			break
		}
		logger.Log.Warn("rpc call failed, retrying",
			zap.String("method", method),
			zap.Int("attempt", attempt),
			zap.Int("maxAttempts", rpcRetryAttempts),
			zap.Duration("sleep", rpcRetrySleep),
			zap.Error(err))
		time.Sleep(rpcRetrySleep)
	}
	return nil, lastErr
}

// get height
func GetBlockCountRPC() uint32 {
	response, err := callRPCWithRetry("getblockcount", []string{})
	if err != nil {
		logger.Log.Info("call failed", zap.Error(err))
		return 0
	}

	blockCountString, ok := response.Result.(json.Number)
	if !ok {
		logger.Log.Info("block count not string",
			zap.Any("result", response.Result),
		)
		return 0
	}

	blockCount, err := blockCountString.Int64()
	if err != nil {
		logger.Log.Info("block count not int", zap.Any("count", blockCountString))
		return 0
	}

	logger.Log.Debug("get block count", zap.Int64("count", blockCount))
	return uint32(blockCount)
}

func GetRawBlock(blockHash string) ([]byte, error) {
	result, err := GetBlockDetail(blockHash, 0)
	if err != nil {
		logger.Log.Error("GetRawBlock err", zap.Error(err))
		return nil, err
	}

	rawBlockString, ok := result.(string)
	if !ok {
		logger.Log.Error("Invalid block data format")
		return nil, fmt.Errorf("invalid block data format")
	}

	rawBlockSize := hex.DecodedLen(len(rawBlockString))
	rawBlock := GetRawBlockBuffer(rawBlockSize)
	n, err := hex.Decode(rawBlock, []byte(rawBlockString))
	if err != nil {
		logger.Log.Info("rawBlock hex err")
		PutRawBlock(rawBlock)
		return nil, err
	}

	return rawBlock[:n], nil
}

// blockHash: block hash value
// verbose: detail level (0: only return hexadecimal raw data, 1: return basic JSON data, 2: return JSON data with transaction details)
func GetBlockDetail(blockHash string, verbose int) (interface{}, error) {
	params := []interface{}{blockHash, verbose}

	response, err := callRPCWithRetry("getblock", params)
	if err != nil {
		logger.Log.Error("RPC call failed", zap.Error(err))
		return nil, err
	}

	return response.Result, nil
}

func GetRawBlockByHeight(height int) ([]byte, error) {
	if height < 0 {
		return nil, fmt.Errorf("invalid negative block height %d", height)
	}

	blockHash, err := getBlockHashRPC(uint32(height))
	if err != nil {
		return nil, err
	}

	return GetRawBlock(blockHash)
}

func getBlockHashRPC(height uint32) (string, error) {
	response, err := callRPCWithRetry("getblockhash", []interface{}{height})
	if err != nil {
		return "", fmt.Errorf("failed to get block hash at height %d: %w", height, err)
	}

	blockHash, ok := response.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid response format for block hash at height %d", height)
	}

	return blockHash, nil
}

// BlockIndexInfo represents the index information for a single block.
type BlockIndexInfo struct {
	Height  uint32
	HashHex string
}

// GetBlockIndexRangeStandardRPC uses public RPC methods and walks backward
// from the target height so the returned headers are from one linked chain.
func GetBlockIndexRangeStandardRPC(startHeight, endHeight, stopHeight uint32, stopHash string) ([]*BlockIndexInfo, bool) {
	if endHeight < startHeight {
		logger.Log.Info("invalid: end height must be greater than or equal to start height",
			zap.Uint32("startHeight", startHeight),
			zap.Uint32("endHeight", endHeight))
		return nil, false
	}

	if endHeight-startHeight > 512 {
		logger.Log.Info("invalid: range too large, maximum 512 blocks",
			zap.Uint32("startHeight", startHeight),
			zap.Uint32("endHeight", endHeight))
		return nil, false
	}

	for retry := 0; retry < 3; retry++ {
		blockCount := GetBlockCountRPC()
		if blockCount == 0 && endHeight > 1 {
			if _, err := getBlockHashRPC(1); err == nil {
				return nil, false
			}
		}

		clampedEndHeight := endHeight
		chainEndHeight := blockCount + 1
		if clampedEndHeight > chainEndHeight {
			clampedEndHeight = chainEndHeight
		}
		if startHeight >= clampedEndHeight {
			return []*BlockIndexInfo{}, true
		}

		blockIndexInfos, err := getBlockIndexRangeStandardRPCOnce(startHeight, clampedEndHeight-1, stopHeight, stopHash)
		if err == nil {
			return blockIndexInfos, true
		}

		logger.Log.Info("standard block index range inconsistent, retrying",
			zap.Error(err),
			zap.Int("retry", retry+1),
			zap.Uint32("startHeight", startHeight),
			zap.Uint32("endHeight", clampedEndHeight))
	}

	return nil, false
}

func getBlockIndexRangeStandardRPCOnce(startHeight, targetHeight, stopHeight uint32, stopHash string) ([]*BlockIndexInfo, error) {
	targetHash, err := getBlockHashRPC(targetHeight)
	if err != nil {
		return nil, err
	}

	blockIndexInfos := make([]*BlockIndexInfo, 0, targetHeight-startHeight+1)
	blockHash := targetHash
	for height := targetHeight; ; height-- {
		response, err := callRPCWithRetry("getblockheader", []interface{}{blockHash, true})
		if err != nil {
			return nil, err
		}
		var header struct {
			Height            uint32 `json:"height"`
			Confirmations     int64  `json:"confirmations"`
			PreviousBlockHash string `json:"previousblockhash"`
		}
		js, err := json.Marshal(response.Result)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(js, &header); err != nil {
			return nil, err
		}
		if header.Height != height {
			return nil, fmt.Errorf("block header height mismatch at %d: got %d", height, header.Height)
		}
		if header.Confirmations < 0 {
			return nil, fmt.Errorf("block %s at height %d is not on active chain", blockHash, height)
		}
		blockIndexInfos = append(blockIndexInfos, &BlockIndexInfo{
			Height:  height,
			HashHex: blockHash,
		})

		if height == startHeight || (stopHash != "" && height == stopHeight && blockHash == stopHash) {
			break
		}
		if height == 0 {
			return nil, fmt.Errorf("block range reached genesis before start height %d", startHeight)
		}
		blockHash = header.PreviousBlockHash
	}

	endHash, err := getBlockHashRPC(targetHeight)
	if err != nil {
		return nil, err
	}
	if endHash != targetHash {
		return nil, fmt.Errorf("block range target changed at %d: got %s want %s", targetHeight, endHash, targetHash)
	}

	for i, j := 0, len(blockIndexInfos)-1; i < j; i, j = i+1, j-1 {
		blockIndexInfos[i], blockIndexInfos[j] = blockIndexInfos[j], blockIndexInfos[i]
	}

	return blockIndexInfos, nil
}
