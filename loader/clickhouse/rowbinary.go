package clickhouse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	// HTTP client with connection pooling for RowBinary inserts
	httpClient *http.Client
	httpBase   string // "http://host:8123"
	httpUser   string
	httpPass   string
	httpDB     string

	fixedZeros [64]byte // pre-allocated zero padding for FixedString
)

// RowBinaryInserter encodes rows in ClickHouse RowBinary format
// and sends them via HTTP POST. Buffer is reused between flushes.
type RowBinaryInserter struct {
	buf        bytes.Buffer
	tableName  string
	rowCount   int
	totalCount int
	maxRows    int
	mu         sync.Mutex
	scratch    [8]byte // reused for encoding integers
}

// NewRowBinaryInserter creates a new inserter for the given table.
func NewRowBinaryInserter(tableName string) *RowBinaryInserter {
	r := &RowBinaryInserter{
		tableName: tableName,
		maxRows:   defaultMaxRowCount,
	}
	r.buf.Grow(defaultBufSize)
	return r
}

// ResetForTable resets the inserter for a new batch, reusing the underlying buffer.
func (r *RowBinaryInserter) ResetForTable(tableName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tableName = tableName
	r.buf.Reset()
	r.rowCount = 0
	r.totalCount = 0
}
func (r *RowBinaryInserter) Lock() {
	r.mu.Lock()
}

// EndRow increments the row count and auto-flushes if threshold is reached.
// Always unlocks the mutex.
func (r *RowBinaryInserter) EndRow() error {
	r.rowCount++
	r.totalCount++
	if r.rowCount >= r.maxRows {
		err := r.flushLocked()
		r.mu.Unlock()
		return err
	}
	r.mu.Unlock()
	return nil
}

// Flush sends any buffered data. Caller must NOT hold the mutex.
func (r *RowBinaryInserter) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.flushLocked()
}

// Count returns the total number of rows written (including flushed).
func (r *RowBinaryInserter) Count() int {
	return r.totalCount
}

// flushLocked sends the buffer via HTTP POST. Caller must hold the mutex.
func (r *RowBinaryInserter) flushLocked() error {
	if r.rowCount == 0 {
		return nil
	}

	query := fmt.Sprintf("INSERT INTO %s FORMAT RowBinary", r.tableName)
	reqURL := fmt.Sprintf("%s/?database=%s&query=%s",
		httpBase,
		url.QueryEscape(httpDB),
		url.QueryEscape(query),
	)

	req, err := http.NewRequest("POST", reqURL, &r.buf)
	if err != nil {
		return fmt.Errorf("rowbinary: create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if httpUser != "" {
		req.Header.Set("X-ClickHouse-User", httpUser)
		req.Header.Set("X-ClickHouse-Key", httpPass)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		// Reset buffer so data is lost on network error (same as driver behavior)
		r.buf.Reset()
		r.rowCount = 0
		return fmt.Errorf("rowbinary: http post failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.buf.Reset()
		r.rowCount = 0
		return fmt.Errorf("rowbinary: clickhouse error (status %d): %s", resp.StatusCode, string(body))
	}
	// Drain the response so net/http can return the connection to the idle pool.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		r.buf.Reset()
		r.rowCount = 0
		return fmt.Errorf("rowbinary: read response body failed: %w", err)
	}

	r.buf.Reset()
	r.rowCount = 0
	return nil
}

// --- RowBinary encoding methods ---
// All Put* methods write directly to the internal buffer.
// Caller must hold the mutex (via Lock).

func (r *RowBinaryInserter) PutUInt8(v uint8) {
	r.buf.WriteByte(v)
}

func (r *RowBinaryInserter) PutUInt16(v uint16) {
	binary.LittleEndian.PutUint16(r.scratch[:2], v)
	r.buf.Write(r.scratch[:2])
}

func (r *RowBinaryInserter) PutUInt32(v uint32) {
	binary.LittleEndian.PutUint32(r.scratch[:4], v)
	r.buf.Write(r.scratch[:4])
}

func (r *RowBinaryInserter) PutUInt64(v uint64) {
	binary.LittleEndian.PutUint64(r.scratch[:8], v)
	r.buf.Write(r.scratch[:8])
}

func (r *RowBinaryInserter) PutInt64(v int64) {
	binary.LittleEndian.PutUint64(r.scratch[:8], uint64(v))
	r.buf.Write(r.scratch[:8])
}

// PutString writes a String value: unsigned LEB128 length + raw bytes.
func (r *RowBinaryInserter) PutString(data []byte) {
	r.putVarUInt(uint64(len(data)))
	r.buf.Write(data)
}

// PutFixedString writes exactly `size` bytes, padding with zeros if needed.
func (r *RowBinaryInserter) PutFixedString(data []byte, size int) {
	n := len(data)
	if n >= size {
		r.buf.Write(data[:size])
		return
	}
	r.buf.Write(data)
	// pad with zeros
	pad := size - n
	for pad > len(fixedZeros) {
		r.buf.Write(fixedZeros[:])
		pad -= len(fixedZeros)
	}
	r.buf.Write(fixedZeros[:pad])
}

// putVarUInt writes an unsigned LEB128 varint.
func (r *RowBinaryInserter) putVarUInt(v uint64) {
	for v >= 0x80 {
		r.buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	r.buf.WriteByte(byte(v))
}

const defaultMaxRowCount = 500000
const defaultBufSize = 32 << 20 // 32MB initial buffer

// InitHTTP initializes the HTTP client for RowBinary inserts.
// Derives HTTP address from native address by replacing port with 8123.
func InitHTTP(nativeAddr, database, username, password string, timeout time.Duration) {
	host := nativeAddr
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	httpBase = "http://" + host + ":8123"
	httpUser = username
	httpPass = password
	httpDB = database
	httpClient = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        32,
			MaxIdleConnsPerHost: 32,
			MaxConnsPerHost:     64,
		},
	}
}
