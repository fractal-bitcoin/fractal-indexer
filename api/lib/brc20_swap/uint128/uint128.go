package uint128

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
)

const MAX_PRECISION = 18

var MAX_PRECISION_STRING = "18"

var precisionFactor64 = [19]uint64{
	1e0,
	1e1,
	1e2,
	1e3,
	1e4,
	1e5,
	1e6,
	1e7,
	1e8,
	1e9,
	1e10,
	1e11,
	1e12,
	1e13,
	1e14,
	1e15,
	1e16,
	1e17,
	1e18,
}

// Zero is a zero-valued uint128.
var Zero Decimal

// A Decimal is an unsigned 128-bit number.
type Decimal struct {
	Precision uint8
	Lo, Hi    uint64
}

// Add returns u+v.
func (u Decimal) Add(v Decimal) Decimal {
	if v.IsZero() {
		return u
	}

	if u.IsZero() {
		return v
	}

	if u.Precision != v.Precision {
		panic("precision not match")
	}

	lo, carry := bits.Add64(u.Lo, v.Lo, 0)
	hi, carry := bits.Add64(u.Hi, v.Hi, carry)
	if carry != 0 {
		panic("overflow")
	}
	return Decimal{u.Precision, lo, hi}
}

// Sub returns u-v.
func (u Decimal) Sub(v Decimal) Decimal {
	if v.IsZero() {
		return u
	}

	if u.Precision != v.Precision {
		panic("precision not match")
	}

	lo, borrow := bits.Sub64(u.Lo, v.Lo, 0)
	hi, borrow := bits.Sub64(u.Hi, v.Hi, borrow)
	if borrow != 0 {
		panic("underflow")
	}
	return Decimal{u.Precision, lo, hi}
}

// QuoRem64 returns q = u/v and r = u%v.
func (u Decimal) QuoRem64(v uint64) (q Decimal, r uint64) {
	if u.Hi < v {
		q.Lo, r = bits.Div64(u.Hi, u.Lo, v)
	} else {
		q.Hi, r = bits.Div64(0, u.Hi, v)
		q.Lo, r = bits.Div64(r, u.Lo, v)
	}
	return
}

// IsZero returns true if u == 0.
func (u Decimal) IsZero() bool {
	// NOTE: we do not compare against Zero, because that is a global variable
	// that could be modified.

	return (u.Hi == u.Lo && u.Lo == 0)
}

func (u Decimal) Sign() int {
	if u.IsZero() {
		return 0
	}
	return 1
}

// Cmp compares u and v and returns:
//
//	-1 if u <  v
//	 0 if u == v
//	+1 if u >  v
func (u Decimal) Cmp(v Decimal) int {
	if u.IsZero() {
		if v.IsZero() {
			return 0
		}
		return -1
	}
	if v.IsZero() {
		return 1
	}

	if u.Precision != v.Precision {
		panic("precision not match")
	}

	if u == v {
		return 0
	} else if u.Hi < v.Hi || (u.Hi == v.Hi && u.Lo < v.Lo) {
		return -1
	} else {
		return 1
	}
}

// Cmp compares u and v and returns:
//
//	-1 if u <  v
//	 0 if u == v
//	+1 if u >  v
func (u Decimal) CmpAlign(v Decimal) int {
	if u.IsZero() {
		if v.IsZero() {
			return 0
		}
		return -1
	}
	if v.IsZero() {
		return 1
	}

	if u == v {
		return 0
	} else if u.Hi < v.Hi || (u.Hi == v.Hi && u.Lo < v.Lo) {
		return -1
	} else {
		return 1
	}
}

func (u Decimal) IsOverflowUint64() bool {
	m := u.GetMaxUint64()
	if u.Cmp(m) > 0 {
		return true
	}
	return false
}

func (u Decimal) GetMaxUint64() (v Decimal) {
	integerPart := new(big.Int).SetUint64(math.MaxUint64)
	precision := new(big.Int).SetUint64(precisionFactor64[u.Precision])
	value := new(big.Int).Mul(integerPart, precision)

	v.Precision = u.Precision
	v.Lo = value.Uint64()
	v.Hi = value.Rsh(value, 64).Uint64()

	return v
}

func (u Decimal) Float64() float64 {
	if u.IsZero() {
		return 0
	}
	if u.Precision == 0 {
		return float64(u.Lo)
	}

	quotient, remainder := u.QuoRem64(precisionFactor64[u.Precision])
	f := float64(quotient.Lo) + float64(remainder)/float64(precisionFactor64[u.Precision])
	return f
}

func (u Decimal) Uint64() uint64 {
	return u.Lo
}

// Big returns u as a *big.Int.
func (u Decimal) Big() *big.Int {
	i := new(big.Int).SetUint64(u.Hi)
	i = i.Lsh(i, 64)
	i = i.Xor(i, new(big.Int).SetUint64(u.Lo))
	return i
}

// New returns the Decimal value (lo,hi).
func New(v uint64, p uint8) Decimal {
	if p > MAX_PRECISION {
		p = MAX_PRECISION
	}
	return Decimal{p, v, 0}
}

// FromBig converts i to a Decimal value. It panics if i is negative or
// overflows 128 bits.
func FromBig(i *big.Int, p uint8) (u Decimal) {
	if i.Sign() < 0 {
		panic("value cannot be negative")
	} else if i.BitLen() > 128 {
		panic("value overflows Decimal")
	}
	u.Precision = p
	u.Lo = i.Uint64()
	u.Hi = new(big.Int).Rsh(i, 64).Uint64()
	return u
}

// FromString creates a Decimal instance from a string
func FromString(s string, maxPrecision int) (u Decimal, err error) {
	if s == "" || s[0] == '-' || s[0] == '+' {
		return u, errors.New("empty string")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return u, fmt.Errorf("invalid decimal format: %s", s)
	}

	integerPartStr := parts[0]
	if integerPartStr == "" || integerPartStr[0] == '+' {
		return u, errors.New("empty integer")
	}

	integerPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return u, fmt.Errorf("invalid integer format: %s", parts[0])
	}

	if integerPart.Sign() < 0 {
		return u, errors.New("value cannot be negative")
	}
	if integerPart.BitLen() > 64 {
		return u, errors.New("value overflows Uint64")
	}

	currPrecision := 0
	decimalPart := big.NewInt(0)
	if len(parts) == 2 {
		decimalPartStr := parts[1]
		if decimalPartStr == "" || decimalPartStr[0] == '-' || decimalPartStr[0] == '+' {
			return u, errors.New("empty decimal")
		}

		currPrecision = len(decimalPartStr)
		if currPrecision > maxPrecision {
			return u, fmt.Errorf("decimal exceeds maximum precision: %s", s)
		}
		n := maxPrecision - currPrecision
		for i := 0; i < n; i++ {
			decimalPartStr += "0"
		}
		decimalPart, ok = new(big.Int).SetString(decimalPartStr, 10)
		if !ok || decimalPart.Sign() < 0 {
			return u, fmt.Errorf("invalid decimal format: %s", parts[0])
		}
	}

	u.Precision = uint8(maxPrecision)
	precision := new(big.Int).SetUint64(precisionFactor64[maxPrecision])

	value := new(big.Int).Mul(integerPart, precision)
	value = value.Add(value, decimalPart)

	u.Lo = value.Uint64()
	u.Hi = value.Rsh(value, 64).Uint64()

	return u, nil
}

// String returns the base-10 representation of u as a string.
func (u Decimal) String() string {
	if u.IsZero() {
		return "0"
	}

	quotient, remainder := u.QuoRem64(precisionFactor64[u.Precision])
	if remainder == 0 {
		return strconv.FormatUint(quotient.Lo, 10)
	}

	var builder strings.Builder
	builder.Grow(20 + int(u.Precision))

	builder.WriteString(strconv.FormatUint(quotient.Lo, 10))
	builder.WriteByte('.')

	decimalStr := strconv.FormatUint(remainder, 10)
	padding := int(u.Precision) - len(decimalStr)
	for i := 0; i < padding; i++ {
		builder.WriteByte('0')
	}

	end := len(decimalStr)
	for end > 0 && decimalStr[end-1] == '0' {
		end--
	}
	builder.WriteString(decimalStr[:end])

	return builder.String()
}
