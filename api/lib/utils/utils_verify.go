package utils

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"strings"
)

func VerifyInscriptionId(inscriptionId string) (err error) {
	if len(inscriptionId) > 64+1+12 {
		return errors.New("inscriptionId too long")
	}

	parts := strings.Split(inscriptionId, "i")
	if len(parts) != 2 || len(parts[0]) != 64 {
		return errors.New("inscriptionId invalid, without 'i'")
	}

	if _, err := hex.DecodeString(parts[0]); err != nil {
		return errors.New("inscriptionId invalid, not hex")
	}

	if idx, err := strconv.Atoi(parts[1]); err != nil || idx < 0 {
		return errors.New("inscriptionId invalid, idx")
	}

	return nil
}

func VerifyInscriptionNumber(strNftNumber string) (err error) {
	// Check whether it is empty.
	if strNftNumber == "" {
		return errors.New("inscriptionId cannot be empty")
	}

	// Try converting the string to an integer; Atoi handles negative numbers automatically.
	_, err = strconv.Atoi(strNftNumber)
	if err != nil {
		return errors.New("inscriptionId invalid")
	}
	return nil
}

func fromCompact(buf []byte) (r, s *big.Int, err error) {
	if len(buf) != 65 {
		return r, s, errors.New("input buffer has incorrect length")
	}

	// Extract R and S values from the buffer.
	rBytes := buf[1:33]
	sBytes := buf[33:65]

	// Check if R and S are the correct length.
	if len(rBytes) != 32 || len(sBytes) != 32 {
		return r, s, errors.New("r and s must be 32 bytes each")
	}

	// Convert R and S to big.Int.
	r = new(big.Int).SetBytes(rBytes)
	s = new(big.Int).SetBytes(sBytes)

	return r, s, nil
}
