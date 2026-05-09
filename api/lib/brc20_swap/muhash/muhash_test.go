package muhash

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
)

func makeBalance(ticker, pkscript string, avail, transfer string, precision int) *model.BRC20TokenBalance {
	a, err := uint128.FromString(avail, precision)
	if err != nil {
		panic(err)
	}
	t, err := uint128.FromString(transfer, precision)
	if err != nil {
		panic(err)
	}
	return &model.BRC20TokenBalance{
		Ticker:              ticker,
		PkScript:            pkscript,
		AvailableBalance:    a,
		TransferableBalance: t,
	}
}

// TestMuHashCommutativity verifies that applying elements in different orders
// produces the same final hash (the core MuHash property).
func TestMuHashCommutativity(t *testing.T) {
	a := []byte("element_a")
	b := []byte("element_b")
	c := []byte("element_c")

	// order 1: a, b, c
	m1 := NewMuHash3072()
	m1.Apply(a)
	m1.Apply(b)
	m1.Apply(c)
	h1 := m1.Finalize()

	// order 2: c, a, b
	m2 := NewMuHash3072()
	m2.Apply(c)
	m2.Apply(a)
	m2.Apply(b)
	h2 := m2.Finalize()

	// order 3: b, c, a
	m3 := NewMuHash3072()
	m3.Apply(b)
	m3.Apply(c)
	m3.Apply(a)
	h3 := m3.Finalize()

	if h1 != h2 || h2 != h3 {
		t.Fatalf("commutativity failed: h1=%x h2=%x h3=%x", h1, h2, h3)
	}
}

// TestMuHashApplyRemoveIdentity verifies that applying then removing the same
// element returns to the original state.
func TestMuHashApplyRemoveIdentity(t *testing.T) {
	m := NewMuHash3072()
	m.Apply([]byte("keep_this"))
	hBefore := m.Finalize()

	m.Apply([]byte("temporary"))
	hDuring := m.Finalize()

	m.Remove([]byte("temporary"))
	hAfter := m.Finalize()

	if hBefore == hDuring {
		t.Fatal("adding an element should change the hash")
	}
	if hBefore != hAfter {
		t.Fatalf("apply+remove should be identity: before=%x after=%x", hBefore, hAfter)
	}
}

// TestMuHashRemoveCommutativity verifies that applying and removing elements
// in different orders produces the same final hash.
func TestMuHashRemoveCommutativity(t *testing.T) {
	elems := [][]byte{
		[]byte("e1"), []byte("e2"), []byte("e3"), []byte("e4"), []byte("e5"),
	}

	// build a base with all 5 elements
	base := NewMuHash3072()
	for _, e := range elems {
		base.Apply(e)
	}

	// remove e2 then e4
	m1 := base.DeepCopy()
	m1.Remove(elems[1])
	m1.Remove(elems[3])
	h1 := m1.Finalize()

	// remove e4 then e2 (different order)
	m2 := base.DeepCopy()
	m2.Remove(elems[3])
	m2.Remove(elems[1])
	h2 := m2.Finalize()

	// build from scratch with only e1, e3, e5
	m3 := NewMuHash3072()
	m3.Apply(elems[0])
	m3.Apply(elems[2])
	m3.Apply(elems[4])
	h3 := m3.Finalize()

	if h1 != h2 {
		t.Fatalf("remove commutativity failed: h1=%x h2=%x", h1, h2)
	}
	if h1 != h3 {
		t.Fatalf("remove result should match building from scratch: remove=%x scratch=%x", h1, h3)
	}
}

// TestMuHashEmptyState verifies the empty MuHash produces a deterministic hash.
func TestMuHashEmptyState(t *testing.T) {
	m1 := NewMuHash3072()
	m2 := NewMuHash3072()

	if m1.Finalize() != m2.Finalize() {
		t.Fatal("two empty MuHash instances should produce the same hash")
	}
}

// TestMuHashFinalizeNonDestructive verifies Finalize does not alter internal state.
func TestMuHashFinalizeNonDestructive(t *testing.T) {
	m := NewMuHash3072()
	m.Apply([]byte("data1"))
	m.Apply([]byte("data2"))

	h1 := m.Finalize()
	h2 := m.Finalize()

	if h1 != h2 {
		t.Fatal("Finalize should be non-destructive")
	}

	// apply more after finalize
	m.Apply([]byte("data3"))
	h3 := m.Finalize()
	if h1 == h3 {
		t.Fatal("hash should change after applying more data")
	}
}

// TestMuHashDeepCopy verifies that DeepCopy creates an independent copy.
func TestMuHashDeepCopy(t *testing.T) {
	m := NewMuHash3072()
	m.Apply([]byte("shared"))
	cp := m.DeepCopy()

	hOrig := m.Finalize()
	hCopy := cp.Finalize()
	if hOrig != hCopy {
		t.Fatal("deep copy should have same hash")
	}

	// mutate original, copy should be unaffected
	m.Apply([]byte("only_original"))
	if m.Finalize() == cp.Finalize() {
		t.Fatal("mutating original should not affect copy")
	}
}

// TestMuHashPersistRestore verifies round-trip through NumeratorBytes/DenominatorBytes.
func TestMuHashPersistRestore(t *testing.T) {
	m := NewMuHash3072()
	m.Apply([]byte("a"))
	m.Apply([]byte("b"))
	m.Remove([]byte("c"))

	numBytes := m.NumeratorBytes()
	denBytes := m.DenominatorBytes()

	m2 := NewMuHash3072FromBytes(numBytes, denBytes)

	if m.Finalize() != m2.Finalize() {
		t.Fatal("restored MuHash should produce the same hash")
	}
}

// --- Balance-level tests ---

// TestBalanceSerializationDeterminism verifies that the same balance always
// produces the same serialized bytes.
func TestBalanceSerializationDeterminism(t *testing.T) {
	b := makeBalance("ordi", "\x00\x14pkscript_bytes_here!", "100.5", "50.25", 18)

	s1 := serializeBalance(b)
	s2 := serializeBalance(b)

	if len(s1) != len(s2) {
		t.Fatal("serialization length should be deterministic")
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			t.Fatalf("serialization differs at byte %d", i)
		}
	}
}

// TestBalanceTickerCaseInsensitive verifies that ticker case does not matter.
func TestBalanceTickerCaseInsensitive(t *testing.T) {
	b1 := makeBalance("ORDI", "pk1", "100", "0", 18)
	b2 := makeBalance("ordi", "pk1", "100", "0", 18)
	b3 := makeBalance("Ordi", "pk1", "100", "0", 18)

	s1 := serializeBalance(b1)
	s2 := serializeBalance(b2)
	s3 := serializeBalance(b3)

	if string(s1) != string(s2) || string(s2) != string(s3) {
		t.Fatal("ticker should be case-insensitive in serialization")
	}
}

// TestBalanceHashZeroSkipped verifies that zero-balance entries are no-ops.
func TestBalanceHashZeroSkipped(t *testing.T) {
	m := NewMuHash3072()
	hBefore := m.Finalize()

	zero := makeBalance("ordi", "pk1", "0", "0", 18)
	m.ApplyBalanceHash(zero)
	hAfter := m.Finalize()

	if hBefore != hAfter {
		t.Fatal("applying a zero-balance entry should be a no-op")
	}

	m.RemoveBalanceHash(zero)
	hAfterRemove := m.Finalize()
	if hBefore != hAfterRemove {
		t.Fatal("removing a zero-balance entry should be a no-op")
	}
}

// TestBalanceHashOrderIndependent verifies that applying balance entries in
// different orders produces the same hash (the key property for MuHash).
func TestBalanceHashOrderIndependent(t *testing.T) {
	balances := []*model.BRC20TokenBalance{
		makeBalance("ordi", "pk_alice", "1000.5", "200", 18),
		makeBalance("ordi", "pk_bob", "500", "100.25", 18),
		makeBalance("sats", "pk_alice", "999999", "0", 18),
		makeBalance("sats", "pk_charlie", "1", "1", 18),
		makeBalance("meme", "pk_bob", "42.123456789", "0", 18),
	}

	// forward order
	m1 := NewMuHash3072()
	for _, b := range balances {
		m1.ApplyBalanceHash(b)
	}
	h1 := m1.Finalize()

	// reverse order
	m2 := NewMuHash3072()
	for i := len(balances) - 1; i >= 0; i-- {
		m2.ApplyBalanceHash(balances[i])
	}
	h2 := m2.Finalize()

	// shuffled order (deterministic seed)
	m3 := NewMuHash3072()
	rng := rand.New(rand.NewSource(12345))
	perm := rng.Perm(len(balances))
	for _, idx := range perm {
		m3.ApplyBalanceHash(balances[idx])
	}
	h3 := m3.Finalize()

	if h1 != h2 {
		t.Fatalf("forward vs reverse: %x != %x", h1, h2)
	}
	if h1 != h3 {
		t.Fatalf("forward vs shuffled: %x != %x", h1, h3)
	}
}

// TestBalanceHashUpdateSimulation simulates the overlay merge flow:
// start with a set of balances, update some, and verify the result matches
// building from scratch with the final state.
func TestBalanceHashUpdateSimulation(t *testing.T) {
	// initial state: 3 balance entries
	initial := []*model.BRC20TokenBalance{
		makeBalance("ordi", "pk_alice", "1000", "200", 18),
		makeBalance("ordi", "pk_bob", "500", "100", 18),
		makeBalance("sats", "pk_alice", "9999", "0", 18),
	}

	m := NewMuHash3072()
	for _, b := range initial {
		m.ApplyBalanceHash(b)
	}

	// simulate overlay merge: alice's ordi balance changes
	oldAliceOrdi := initial[0]
	newAliceOrdi := makeBalance("ordi", "pk_alice", "800", "400", 18)
	m.RemoveBalanceHash(oldAliceOrdi)
	m.ApplyBalanceHash(newAliceOrdi)

	// simulate overlay merge: bob's ordi balance goes to zero (removed from set)
	oldBobOrdi := initial[1]
	newBobOrdi := makeBalance("ordi", "pk_bob", "0", "0", 18)
	m.RemoveBalanceHash(oldBobOrdi)
	m.ApplyBalanceHash(newBobOrdi) // zero balance = no-op

	// simulate overlay merge: new entry for charlie
	newCharlie := makeBalance("sats", "pk_charlie", "50", "10", 18)
	m.ApplyBalanceHash(newCharlie)

	hIncremental := m.Finalize()

	// build from scratch with final state
	final := []*model.BRC20TokenBalance{
		makeBalance("ordi", "pk_alice", "800", "400", 18),
		// bob removed (zero balance)
		makeBalance("sats", "pk_alice", "9999", "0", 18),
		makeBalance("sats", "pk_charlie", "50", "10", 18),
	}

	mScratch := NewMuHash3072()
	for _, b := range final {
		mScratch.ApplyBalanceHash(b)
	}
	hScratch := mScratch.Finalize()

	if hIncremental != hScratch {
		t.Fatalf("incremental update should match from-scratch: %x != %x", hIncremental, hScratch)
	}
}

// TestBlockCommitmentHash verifies BlockCommitment produces deterministic output.
func TestBlockCommitmentHash(t *testing.T) {
	bc := &BlockCommitment{
		Version:        0x02,
		Height:         840000,
		BlockHash:      [32]byte{0x01, 0x02, 0x03},
		Brc20StateHash: [32]byte{0xaa, 0xbb, 0xcc},
	}

	h1 := bc.CommitmentHash()
	h2 := bc.CommitmentHash()
	if h1 != h2 {
		t.Fatal("commitment hash should be deterministic")
	}

	// changing any field should change the hash
	bc2 := *bc
	bc2.Height = 840001
	if bc.CommitmentHash() == bc2.CommitmentHash() {
		t.Fatal("different height should produce different commitment hash")
	}
}

// TestMuHashLargeSetOrderIndependent applies 100 random balance entries in
// multiple random orders and verifies all produce the same hash.
func TestMuHashLargeSetOrderIndependent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tickers := []string{"ordi", "sats", "meme", "punk", "pepe"}
	n := 100

	balances := make([]*model.BRC20TokenBalance, n)
	for i := 0; i < n; i++ {
		ticker := tickers[rng.Intn(len(tickers))]
		pk := make([]byte, 20)
		rng.Read(pk)
		avail := uint128.Decimal{Precision: 18, Lo: rng.Uint64()%1e15 + 1}
		trans := uint128.Decimal{Precision: 18, Lo: rng.Uint64() % 1e15}
		balances[i] = &model.BRC20TokenBalance{
			Ticker:              ticker,
			PkScript:            string(pk),
			AvailableBalance:    avail,
			TransferableBalance: trans,
		}
	}

	// canonical order
	m1 := NewMuHash3072()
	for _, b := range balances {
		m1.ApplyBalanceHash(b)
	}
	h1 := m1.Finalize()

	// 5 random permutations
	for trial := 0; trial < 5; trial++ {
		m := NewMuHash3072()
		perm := rng.Perm(n)
		for _, idx := range perm {
			m.ApplyBalanceHash(balances[idx])
		}
		h := m.Finalize()
		if h != h1 {
			t.Fatalf("trial %d: shuffled hash differs from canonical: %x != %x", trial, h, h1)
		}
	}
}

// --- Benchmarks ---

// generateBalances creates n random balance entries for benchmarking.
// Simulates realistic data: multiple tickers, varied pkscript lengths, varied amounts.
func generateBalances(n int, seed int64) []*model.BRC20TokenBalance {
	rng := rand.New(rand.NewSource(seed))
	tickers := []string{"ordi", "sats", "meme", "punk", "pepe", "cats", "rats", "bear", "fish", "btcs"}
	balances := make([]*model.BRC20TokenBalance, n)
	for i := 0; i < n; i++ {
		ticker := tickers[rng.Intn(len(tickers))]
		// realistic pkscript: 22-34 bytes
		pkLen := 22 + rng.Intn(13)
		pk := make([]byte, pkLen)
		rng.Read(pk)
		avail := uint128.Decimal{Precision: 18, Lo: rng.Uint64()%1e15 + 1}
		trans := uint128.Decimal{Precision: 18, Lo: rng.Uint64() % 1e15}
		balances[i] = &model.BRC20TokenBalance{
			Ticker:              ticker,
			PkScript:            string(pk),
			AvailableBalance:    avail,
			TransferableBalance: trans,
		}
	}
	return balances
}

// BenchmarkRebuildMuHash simulates RebuildMuHashFromBalances at various scales.
// Reports: total time, per-entry time, and entries/sec.
func BenchmarkRebuildMuHash(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, n := range sizes {
		balances := generateBalances(n, 42)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := NewMuHash3072()
				for _, bal := range balances {
					m.ApplyBalanceHash(bal)
				}
				m.Finalize()
			}
		})
	}
}

// BenchmarkApplyBalanceHash measures the cost of a single ApplyBalanceHash call.
func BenchmarkApplyBalanceHash(b *testing.B) {
	bal := makeBalance("ordi", "pk_alice_address_bytes", "123456789.123456789", "987654321.987654321", 18)
	m := NewMuHash3072()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ApplyBalanceHash(bal)
	}
}

// BenchmarkSerializeBalance measures the serialization cost alone.
func BenchmarkSerializeBalance(b *testing.B) {
	bal := makeBalance("ordi", "pk_alice_address_bytes", "123456789.123456789", "987654321.987654321", 18)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = serializeBalance(bal)
	}
}

// BenchmarkDataToElement measures the SHA256→ChaCha20→big.Int field mapping cost.
func BenchmarkDataToElement(b *testing.B) {
	data := []byte("benchmark_element_data_sample")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dataToElement(data)
	}
}

// BenchmarkFinalize measures the cost of Finalize (ModInverse + SHA256).
func BenchmarkFinalize(b *testing.B) {
	m := NewMuHash3072()
	// add some state so Finalize does real work
	for i := 0; i < 100; i++ {
		m.Apply([]byte(fmt.Sprintf("element_%d", i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Finalize()
	}
}
