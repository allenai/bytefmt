// Package bytefmt contains provides tools to parse, format, and manipulate byte
// quantities. This package emphasizes accuracy over performance and implements
// both binary and decimal International System of Units (SI) conventions.
package bytefmt

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// FromInt64 creates a new size from a count of bytes and an optional scale.
// See SetUnit for more detailed information about units.
func FromInt64(i int64, u Unit) Size {
	mustValidateUnit(u)
	return Size{i, u}
}

// Size is a count of bytes with human-friendly unit representation.
//
// Size offers control over how byte quantities are formatted through either
// automatic or explicit scaling to a byte quantity.
type Size struct {
	bytes int64
	unit  Unit
}

// IsZero returns whether a size is exactly zero bytes.
func (s Size) IsZero() bool { return s.bytes == 0 }

// Equal returns whether two sizes represent the same number of bytes.
func (s Size) Equal(t Size) bool { return s.bytes == t.bytes }

// SetInt64 overrides a size's byte count while leaving its unit scale unchanged.
func (s *Size) SetInt64(bytes int64) { s.bytes = bytes }

// Int64 returns a size's representation as an absolute number of bytes.
func (s Size) Int64() int64 { return s.bytes }

// Unit returns the assigned unit size.
func (s Size) Unit() Unit { return s.unit }

// SetUnit overrides the assigned unit, or quanitity suffix.
//
// The values Metric and Binary will automatically scale formatted values to the
// most appropriate unit in their respective standards.
func (s *Size) SetUnit(u Unit) {
	mustValidateUnit(u)
	s.unit = u
}

// Parse converts a string representation of a byte quantity to a Size.
// Fractional values are truncated to the nearest byte, rounding toward zero.
//
// Parsed values retain their unit scale, defaulting to Byte if no unit is
// specified. Unit prefixes are permissive for metric SI units ("K" = "kB"), but
// strict for binary SI units ("KiB"). Units may be overridden with SetUnit.
//
//    Parse("1024")     = 1,024 B     = 1,024 bytes
//    Parse("1024k")    = 1,024 kB    = 1,024,000 bytes
//    Parse("1mb")      =    10 MB    = 10,000,000 bytes
//    Parse("1.25 GiB") =  1.25 GiB   = 1,342,177,280 bytes
func Parse(s string) (Size, error) {
	size, err := parse(s)
	if err != nil {
		return Size{}, fmt.Errorf("can't convert %q to size: %w", s, err)
	}
	return size, nil
}

var ten = big.NewInt(10)

func parse(s string) (Size, error) {
	if len(s) == 0 {
		return Size{}, errors.New("empty string")
	}
	if s[0] == '-' {
		return Size{}, errors.New("values must be non-negative")
	}

	pos, end := 0, len(s)

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

	// Trim optional whitespace between number and unit.
	if pos < end && s[pos] == ' ' {
		pos++
	}

	// Everything remaining must be the unit suffix.
	unit, err := parseUnit(s[pos:end])
	if err != nil {
		return Size{}, err
	}

	// To avoid precision loss for large numbers, calculate size in big decimal.
	// value = (whole * 10**len(frac) + frac) * unit / 10**len(frac)

	var val, u big.Int
	val.SetString(whole, 10)
	u.SetInt64(int64(unit))

	if len(frac) != 0 {
		var exp, f big.Int
		exp.SetInt64(int64(len(frac))).Exp(ten, &exp, nil)
		f.SetString(frac, 10)
		val.Mul(&val, &exp).Add(&val, &f).Mul(&val, &u).Quo(&val, &exp)
	} else {
		// In the common case we can skip all the above math, but we still need
		// to use bigint to check whether the value fits in 64 bits.
		val.Mul(&val, &u)
	}

	if !val.IsInt64() {
		return Size{}, errors.New("value exceeds 63 bits")
	}

	return Size{bytes: val.Int64(), unit: unit}, nil
}

// String returns the formatted quantity with unbounded precision.
func (s Size) String() string {
	return s.Format(-1)
}

// Format converts a byte quantity to a string, according to the given precision.
//
// If the size's unit is Byte, its value is formatted exactly with no rounding,
// ignoring precision. For other units, precision is limited to 53 significant
// bits, or just under 10 petabytes.
//
// Precision sets the number of digits after the decimal, similar to the %f
// verb. As a special case, setting precision to -1 uses the smallest number of
// digits necessary such that Parse will return the same value.
func (s Size) Format(prec int) string {
	unit := inferUnit(s.bytes, s.unit)

	// Special case: format byte units exactly to avoid precision loss.
	if unit == Byte {
		// TODO: Should this append the unit?
		// TODO: If prec >= 0, should we append "." + strings.Repeat("0", prec)?
		return strconv.FormatInt(s.bytes, 10)
	}

	// We accept loss of precision over 53 significant bits from casting.
	return strconv.FormatFloat(float64(s.bytes)/float64(unit), 'f', prec, 64) + " " + suffixes[unit]
}

// MarshalJSON implements the json.Marshaler interface. It does not guarantee
// full precision. Callers requiring exact values should call s.SetUnit(Byte)
// before marshaling.
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

// Scan implements the sql.Scanner interface for database deserialization.
func (s *Size) Scan(value interface{}) error {
	switch v := value.(type) {
	case int64:
		*s = FromInt64(v, Metric)
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
