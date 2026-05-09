package muhash

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/model"
	"golang.org/x/crypto/chacha20"
)

// MuHash3072 prime: 2^3072 - 1103717
// This is the same prime used by Bitcoin Core for UTXO set hashing.
var muHashPrime *big.Int

func init() {
	muHashPrime = new(big.Int).Lsh(big.NewInt(1), 3072)
	muHashPrime.Sub(muHashPrime, big.NewInt(1103717))
}

// MuHash3072 implements a multiplicative hash over a 3072-bit prime field.
// Elements are added via multiplication and removed via division (tracked
// separately as a denominator product). Finalize computes the quotient.
type MuHash3072 struct {
	numerator   *big.Int
	denominator *big.Int
}

func NewMuHash3072() *MuHash3072 {
	return &MuHash3072{
		numerator:   big.NewInt(1),
		denominator: big.NewInt(1),
	}
}

// dataToElement hashes arbitrary data into a 3072-bit field element.
// Steps: SHA256(data) → ChaCha20 expand to 384 bytes → interpret as LE big.Int → mod prime.
func dataToElement(data []byte) *big.Int {
	// SHA256 to get 32-byte key
	h := sha256.Sum256(data)

	// ChaCha20 with zero nonce, encrypt 384 zero bytes to get keystream
	nonce := make([]byte, chacha20.NonceSize) // 12 bytes of zeros
	cipher, err := chacha20.NewUnauthenticatedCipher(h[:], nonce)
	if err != nil {
		panic("chacha20 init failed: " + err.Error())
	}

	buf := make([]byte, 384)
	cipher.XORKeyStream(buf, buf) // encrypt zeros = keystream

	// interpret as little-endian 3072-bit integer
	elem := new(big.Int)
	// reverse to big-endian for big.Int.SetBytes
	reversed := make([]byte, 384)
	for i := 0; i < 384; i++ {
		reversed[383-i] = buf[i]
	}
	elem.SetBytes(reversed)
	elem.Mod(elem, muHashPrime)

	// element must not be zero in the multiplicative group
	if elem.Sign() == 0 {
		elem.SetInt64(1)
	}
	return elem
}

// Apply adds an element to the set (multiplies into numerator).
func (m *MuHash3072) Apply(data []byte) {
	elem := dataToElement(data)
	m.numerator.Mul(m.numerator, elem)
	m.numerator.Mod(m.numerator, muHashPrime)
}

// Remove removes an element from the set (multiplies into denominator).
func (m *MuHash3072) Remove(data []byte) {
	elem := dataToElement(data)
	m.denominator.Mul(m.denominator, elem)
	m.denominator.Mod(m.denominator, muHashPrime)
}

// Finalize computes the final 32-byte hash: SHA256(numerator/denominator mod prime).
// This is non-destructive — internal state is not modified.
func (m *MuHash3072) Finalize() [32]byte {
	// result = numerator * denominator^(-1) mod prime
	denInv := new(big.Int).ModInverse(m.denominator, muHashPrime)
	if denInv == nil {
		// denominator is 0 mod prime, should not happen
		panic("muhash: denominator has no inverse")
	}
	result := new(big.Int).Mul(m.numerator, denInv)
	result.Mod(result, muHashPrime)

	// serialize as 384 bytes little-endian
	resultBytes := result.Bytes() // big-endian
	buf := make([]byte, 384)
	for i, b := range resultBytes {
		buf[len(resultBytes)-1-i] = b
	}

	return sha256.Sum256(buf)
}

func (m *MuHash3072) DeepCopy() *MuHash3072 {
	return &MuHash3072{
		numerator:   new(big.Int).Set(m.numerator),
		denominator: new(big.Int).Set(m.denominator),
	}
}

// NumeratorBytes returns the numerator as big-endian bytes for persistence.
func (m *MuHash3072) NumeratorBytes() []byte {
	return m.numerator.Bytes()
}

// DenominatorBytes returns the denominator as big-endian bytes for persistence.
func (m *MuHash3072) DenominatorBytes() []byte {
	return m.denominator.Bytes()
}

// NewMuHash3072FromBytes restores a MuHash3072 from persisted big-endian byte slices.
func NewMuHash3072FromBytes(num, den []byte) *MuHash3072 {
	m := &MuHash3072{
		numerator:   new(big.Int).SetBytes(num),
		denominator: new(big.Int).SetBytes(den),
	}
	if m.numerator.Sign() == 0 {
		m.numerator.SetInt64(1)
	}
	if m.denominator.Sign() == 0 {
		m.denominator.SetInt64(1)
	}
	return m
}

// --- Balance serialization ---

// serializeBalance produces a deterministic byte representation of a balance entry.
// Format: len(ticker)[4B LE] + ticker[lowercase UTF-8] + len(pkscript)[4B LE] + pkscript[raw]
//   - len(available_str)[4B LE] + available_str + len(transferable_str)[4B LE] + transferable_str
func serializeBalance(balance *model.BRC20TokenBalance) []byte {
	ticker := strings.ToLower(balance.Ticker)
	pkscript := balance.PkScript
	availStr := balance.AvailableBalance.String()
	transStr := balance.TransferableBalance.String()

	size := 4 + len(ticker) + 4 + len(pkscript) + 4 + len(availStr) + 4 + len(transStr)
	buf := make([]byte, 0, size)

	var lenBuf [4]byte

	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(ticker)))
	buf = append(buf, lenBuf[:]...)
	buf = append(buf, []byte(ticker)...)

	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(pkscript)))
	buf = append(buf, lenBuf[:]...)
	buf = append(buf, []byte(pkscript)...)

	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(availStr)))
	buf = append(buf, lenBuf[:]...)
	buf = append(buf, []byte(availStr)...)

	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(transStr)))
	buf = append(buf, lenBuf[:]...)
	buf = append(buf, []byte(transStr)...)

	return buf
}

// ApplyBalanceHash adds a balance entry to the MuHash set.
// Zero-balance entries (both available and transferable are zero) are skipped.
func (m *MuHash3072) ApplyBalanceHash(balance *model.BRC20TokenBalance) {
	if balance.AvailableBalance.Sign() == 0 && balance.TransferableBalance.Sign() == 0 {
		return
	}
	m.Apply(serializeBalance(balance))
}

// RemoveBalanceHash removes a balance entry from the MuHash set.
// Zero-balance entries are skipped (they were never added).
func (m *MuHash3072) RemoveBalanceHash(balance *model.BRC20TokenBalance) {
	if balance.AvailableBalance.Sign() == 0 && balance.TransferableBalance.Sign() == 0 {
		return
	}
	m.Remove(serializeBalance(balance))
}

// --- BlockCommitment ---

type BlockCommitment struct {
	Version        uint8
	Height         uint64
	BlockHash      [32]byte
	Brc20StateHash [32]byte
}

// CommitmentHash computes SHA256(version + height + block_hash + brc20_state_hash).
func (b *BlockCommitment) CommitmentHash() [32]byte {
	var buf [1 + 8 + 32 + 32]byte
	buf[0] = b.Version
	binary.LittleEndian.PutUint64(buf[1:9], b.Height)
	copy(buf[9:41], b.BlockHash[:])
	copy(buf[41:73], b.Brc20StateHash[:])
	return sha256.Sum256(buf[:])
}
