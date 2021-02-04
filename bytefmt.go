// Package bytefmt contains provides tools to parse, format, and manipulate byte
// quantities. This package emphasizes accuracy over performance and implements
// both binary and decimal International System of Units (SI) conventions.
package bytefmt

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Commonly used values; do not change.
var (
	ten      = big.NewInt(10)
	tenPow3  = big.NewInt(1000)
	twoPow10 = big.NewInt(1024)
)

// New returns a new size from a count of bytes.
func New(bytes int64, base Base) *Size {
	return &Size{bytes, base}
}

// Size is a count of bytes with human-friendly unit scaling.
//
// Size offers control over how byte quantities are formatted through either
// automatic or explicit scaling to a byte quantity.
type Size struct {
	bytes int64
	Base  Base
}

// IsZero returns whether a size is exactly zero bytes.
func (s Size) IsZero() bool { return s.bytes == 0 }

// Equal returns whether two sizes represent the same number of bytes.
func (s Size) Equal(t Size) bool { return s.bytes == t.bytes }

// Cmp compares s and t and returns:
//   -1 if s <  y
//    0 if s == y
//   +1 if s >  y
func (s Size) Cmp(y Size) int {
	switch {
	case s.bytes == y.bytes:
		return 0
	case s.bytes < y.bytes:
		return -1
	default:
		return 1
	}
}

// Add adds size y to the current value.
func (s *Size) Add(y Size) { s.bytes += y.bytes }

// Sub subtracts size y from the current value.
func (s *Size) Sub(y Size) { s.bytes += y.bytes }

// Neg sets the current value to -s.
func (s *Size) Neg() { s.bytes = -1 }

// SetInt64 overrides a size's byte count while leaving its unit scale unchanged.
func (s *Size) SetInt64(bytes int64) { s.bytes = bytes }

// Int64 returns a size's representation as an absolute number of bytes.
func (s Size) Int64() int64 { return s.bytes }

// Parse converts a string representation of a byte quantity to a Size.
// Fractional values are truncated to the nearest byte, rounding toward zero.
//
// Parsed values retain their base format, defaulting to Metric if the suffix is
// missing. Unit prefixes are permissive for Metric scales ("K" = "kB"), but
// strict for Binary scales ("KiB").
//
//    Parse("1024")     = 1,024 B  = 1,024 bytes
//    Parse("1024k")    = 1,024 kB = 1,024,000 bytes
//    Parse("1.1gb")    = 1100 MB  = 1,100,000,000 bytes
//    Parse("1.25 GiB") = 1.25 GiB = 1,342,177,280 bytes
func Parse(s string) (Size, error) {
	size, err := parse(s)
	if err != nil {
		return Size{}, fmt.Errorf("can't convert %q to size: %w", s, err)
	}
	return size, nil
}

func parse(s string) (Size, error) {
	if len(s) == 0 {
		return Size{}, errors.New("empty string")
	}

	pos, end := 0, len(s)

	// Skip the sign. This is included by default.
	if len(s) != 0 && s[0] == '-' {
		pos++
	}

	// Parse the whole number part.
	var whole string
	for ; pos < end; pos++ {
		if s[pos] < '0' || s[pos] > '9' {
			break
		}
	}
	whole = s[:pos]

	// Parse the fractional number part.
	var frac string
	if pos < end && s[pos] == '.' {
		pos++
		fracStart := pos
		for ; pos < end; pos++ {
			if s[pos] < '0' || s[pos] > '9' {
				break
			}
		}
		frac = s[fracStart:pos]
	}

	// Normalize whole and fractional parts.
	if len(whole) == 0 && len(frac) == 0 {
		return Size{}, errors.New("must start with a number")
	}
	if len(whole) == 0 {
		whole = "0"
	}
	frac = strings.TrimRight(frac, "0")

	// Trim optional whitespace between number and unit suffix.
	if pos < end && s[pos] == ' ' {
		pos++
	}

	// Everything remaining must be the unit suffix.
	exp, base, err := parseSuffix(s[pos:end])
	if err != nil {
		return Size{}, err
	}

	// To avoid precision loss for large numbers, calculate size in big decimal.
	// value = (whole * 10**len(frac) + frac) * scale / 10**len(frac)

	var val, scale big.Int
	val.SetString(whole, 10)

	// Calculate the scalar. Base is guaranteed valid by parseSuffix.
	scale.SetInt64(int64(exp))
	switch base {
	case Metric:
		scale.Exp(tenPow3, &scale, nil)
	case Binary:
		scale.Exp(twoPow10, &scale, nil)
	}

	// Scale the number.
	if len(frac) != 0 {
		var prec, f big.Int
		prec.SetInt64(int64(len(frac))).Exp(ten, &prec, nil)
		f.SetString(frac, 10)
		val.Mul(&val, &prec).Add(&val, &f).Mul(&val, &scale).Quo(&val, &prec)
	} else {
		// For whole numbers we can skip all the precision math.
		val.Mul(&val, &scale)
	}

	if !val.IsInt64() {
		return Size{}, errors.New("value exceeds 64 bits")
	}

	return Size{bytes: val.Int64(), Base: base}, nil
}

// String returns the formatted quantity scaled to the largest exact base unit.
func (s Size) String() string {
	mant := s.bytes
	var exp int
	var suffix string

	switch s.Base {
	case 0, Metric:
		for (mant >= 1000 || mant <= -1000) && mant%1000 == 0 && exp < len(metricSuffixes) {
			exp++
			mant = mant / 1000
		}
		suffix = metricSuffixes[exp]
	case Binary:
		for (mant >= 1000 || mant <= -1000) && mant%1024 == 0 && exp < len(binarySuffixes) {
			exp++
			mant = mant / 1024
		}
		suffix = binarySuffixes[exp]
	default:
		panic("invalid base")
	}

	result := make([]byte, 0, 20) // Pre-allocate a size most numbers would fit within.
	result = strconv.AppendInt(result, mant, 10)
	result = append(result, ' ')
	result = append(result, suffix...)
	return string(result)
}

// MarshalJSON implements the json.Marshaler interface.
func (s Size) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(s.String())), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *Size) UnmarshalJSON(value []byte) error {
	if string(value) == "null" {
		return errors.New("can't decode null as bytefmt.Size")
	}

	// Strip quotes if present.
	str := string(value)
	if len(str) > 2 && str[0] == '"' {
		var err error
		if str, err = strconv.Unquote(str); err != nil {
			return fmt.Errorf("can't decode %q as bytefmt.Size: %w", value, err)
		}
	}

	size, err := Parse(str)
	*s = size
	if err != nil {
		return fmt.Errorf("can't decode %q as bytefmt.Size: %w", str, err)
	}
	return nil
}

// Value implements the sql.Valuer interface. It always produces a string.
func (s Size) Value() (driver.Value, error) {
	return s.String(), nil
}

// Scan implements the sql.Scanner interface. It accepts numeric and string values.
func (s *Size) Scan(value interface{}) error {
	switch v := value.(type) {
	case int64:
		*s = *New(v, Metric)
		return nil

	case string:
		var err error
		*s, err = Parse(v)
		return err

	case []byte:
		var err error
		*s, err = Parse(string(v))
		return err

	default: // Interpret as a string.
		return fmt.Errorf("could not convert value '%+v' of type '%T' to bytefmt.Size", value, value)
	}
}
