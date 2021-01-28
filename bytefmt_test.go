package bytefmt

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		In          string
		ExpectBytes int64
		ExpectUnit  Unit
		ExpectErr   string
	}{
		// Invalid values should produce errors.
		{In: "", ExpectErr: "empty string"},
		{In: "-1B", ExpectErr: "values must be non-negative"},
		{In: " B", ExpectErr: "must start with a number"},
		{In: ". B", ExpectErr: "must start with a number"},
		{In: "9223372036854775808", ExpectErr: "value exceeds 63 bits"},
		{In: "8.0 EiB", ExpectErr: "value exceeds 63 bits"},
		{In: "1 tUb", ExpectErr: `"tUb" is not a valid byte quantity`},

		// Missing leading or trailing digits is OK.
		{In: ".1 kB", ExpectBytes: 100, ExpectUnit: KB},
		{In: "1. kB", ExpectBytes: 1000, ExpectUnit: KB},

		// Zero parses correctly.
		{In: "0", ExpectBytes: 0, ExpectUnit: Byte},
		{In: "0 B", ExpectBytes: 0, ExpectUnit: Byte},
		{In: "0mib", ExpectBytes: 0, ExpectUnit: MiB},

		// Max values parse correctly, even with extreme precision.
		{In: "9223372036854775807", ExpectBytes: 9_223_372_036_854_775_807, ExpectUnit: Byte},
		{In: "9.223372036854775807eb", ExpectBytes: 9_223_372_036_854_775_807, ExpectUnit: EB},
		{In: "7.99999999999999999914 EiB", ExpectBytes: 9_223_372_036_854_775_807, ExpectUnit: EiB},

		// Metric and binary suffixes produce different results.
		{In: "123.456g", ExpectBytes: 123_456_000_000, ExpectUnit: GB},
		{In: "123.456 GB", ExpectBytes: 123_456_000_000, ExpectUnit: GB},
		{In: "123.456 GiB", ExpectBytes: 132_559_870_623, ExpectUnit: GiB},
	}

	for _, test := range tests {
		size, err := Parse(test.In)

		if test.ExpectErr != "" {
			expectErr := fmt.Sprintf("can't convert %q to size: %s", test.In, test.ExpectErr)
			assertEqualErr(t, expectErr, err, "Error for %q", test.In)
			continue
		}

		if assertNoErr(t, err, "Unxpected error for %q", test.In) {
			continue
		}
		assertEqual(t, test.ExpectBytes, size.Int64(), "Byte count for %q", test.In)
		assertEqual(t, test.ExpectUnit, size.Unit(), "Unit for %q", test.In)
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		InSize Size
		InPrec int64
		Expect string
	}{
		// Zero values
		{InSize: FromInt64(0, Byte), InPrec: -1, Expect: "0"},
		{InSize: FromInt64(0, Byte), InPrec: 2, Expect: "0"},
		{InSize: FromInt64(0, GiB), InPrec: 2, Expect: "0.00 GiB"},

		// Maximum values representable by float64: 2**53-1
		{InSize: FromInt64(9_007_199_254_740_991, Byte), InPrec: -1, Expect: "9007199254740991"},
		{InSize: FromInt64(9_007_199_254_740_991, PB), InPrec: -1, Expect: "9.007199254740991 PB"},
		{InSize: FromInt64(9_007_199_254_740_991, GiB), InPrec: -1, Expect: "8388607.999999999 GiB"},

		// Maximum values, variable precision; non-byte units are lossy.
		{InSize: FromInt64(9_223_372_036_854_775_807, Byte), InPrec: -1, Expect: "9223372036854775807"},
		{InSize: FromInt64(9_223_372_036_854_775_807, GB), InPrec: -1, Expect: "9223372036.854776 GB"},
		{InSize: FromInt64(9_223_372_036_854_775_807, GiB), InPrec: -1, Expect: "8589934592 GiB"},
		{InSize: FromInt64(9_223_372_036_854_775_807, EiB), InPrec: -1, Expect: "8 EiB"},

		// Variable precision
		{InSize: FromInt64(1_234_567, KB), InPrec: -1, Expect: "1234.567 kB"},
		{InSize: FromInt64(1_234_567, KB), InPrec: 0, Expect: "1235 kB"},
		{InSize: FromInt64(1_234_567, MB), InPrec: 3, Expect: "1.235 MB"},
		{InSize: FromInt64(1_234_567, MB), InPrec: 10, Expect: "1.2345670000 MB"},

		// Metric auto units
		{InSize: FromInt64(999, Metric), InPrec: -1, Expect: "999"},
		{InSize: FromInt64(1000, Metric), InPrec: -1, Expect: "1 kB"},
		{InSize: FromInt64(123_4567, Metric), InPrec: -1, Expect: "1.234567 MB"},

		// Binary auto units
		{InSize: FromInt64(1023, Binary), InPrec: 3, Expect: "1023"},
		{InSize: FromInt64(1024, Binary), InPrec: 3, Expect: "1.000 KiB"},
		{InSize: FromInt64(123_4567, Binary), InPrec: 3, Expect: "1.177 MiB"},
	}

	for _, test := range tests {
		str := test.InSize.Format(int(test.InPrec))
		assertEqual(t, test.Expect, str, "Formatting (%d, %v) with prec %d",
			test.InSize.Int64(), test.InSize.Unit(), test.InPrec)
	}
}

func assertNoErr(t *testing.T, err error, message string, args ...interface{}) bool {
	if err == nil {
		return true
	}
	t.Error(fmt.Sprintf(message, args...),
		"\n    Error:", err)
	return false
}

func assertEqualErr(
	t *testing.T,
	expect string,
	actual error,
	message string,
	args ...interface{},
) bool {
	if actual != nil {
		return assertEqual(t, expect, actual.Error(), message, args...)
	}
	return assertEqual(t, expect, actual, message, args...)
}

func assertEqual(
	t *testing.T,
	expect interface{},
	actual interface{},
	message string,
	args ...interface{},
) bool {
	if reflect.DeepEqual(expect, actual) {
		return true
	}
	t.Error(fmt.Sprintf(message, args...),
		"\n    Expected:", expect,
		"\n    Actual:  ", actual)
	return false
}
