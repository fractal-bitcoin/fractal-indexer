package model

import (
	"errors"
)

var (
	ErrLockUtxoLimitExceeded   = errors.New("utxo: exceeded maximum manual lock limit of 1000")
	ErrUnlockUtxoLimitExceeded = errors.New("utxo: exceeded maximum unlock limit of 1000, Please send or merge some of the unlocked utxos first")
	ErrRunesUtxoLimitExceeded  = errors.New("runes utxo over 100000")
)

// ErrorMsg accepts an error and returns the error message.
func GetErrorResponse(err error, defaultMsg string) Response {
	if err == ErrRunesUtxoLimitExceeded {
		return Response{Code: -1002, Msg: ErrRunesUtxoLimitExceeded.Error()}
	} else if err == ErrLockUtxoLimitExceeded {
		return Response{Code: -1001, Msg: ErrLockUtxoLimitExceeded.Error()}
	} else if err == ErrUnlockUtxoLimitExceeded {
		return Response{Code: -1001, Msg: ErrUnlockUtxoLimitExceeded.Error()}
	}
	return Response{Code: -1, Msg: defaultMsg}
}
