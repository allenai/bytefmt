package bytefmt

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestCmp(t *testing.T) {
	tests := []struct {
		A      *Size
		B      *Size
		Expect int
	}{
		// Zeroes
		{A: &Size{}, B: &Size{}, Expect: 0},
		{A: New(0, Metric), B: New(0, Binary), Expect: 0},

		// Different bases
		{A: New(1024*KB, Metric), B: New(1000*KiB, Binary), Expect: 0},
		{A: New(1000*KB, Metric), B: New(1024*KB, Binary), Expect: -1},
		{A: New(1024*KiB, Metric), B: New(1000*KiB, Binary), Expect: 1},

		// Equal and opposite
		{A: New(-1, Metric), B: New(1, Metric), Expect: -1},
		{A: New(1024*KiB, Metric), B: New(-1000*KiB, Metric), Expect: 1},

		// Extreme values
		{A: New(math.MaxInt64, Metric), B: New(math.MaxInt64, Metric), Expect: 0},
		{A: New(math.MinInt64, Metric), B: New(math.MaxInt64, Metric), Expect: -1},
		{A: New(math.MaxInt64, Metric), B: New(math.MinInt64, Metric), Expect: 1},
	}

	for _, test := range tests {
		result := test.A.Cmp(*test.B)
		assertEqual(t, test.Expect, result, "Comparing %v against %v", test.A, test.B)
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		A      *Size
		B      *Size
		Expect int64
	}{
		// Zeroes
		{A: &Size{}, B: &Size{}, Expect: 0},
		{A: New(0, Metric), B: New(0, Binary), Expect: 0},

		// Different bases
		{A: New(123, Metric), B: New(456, Binary), Expect: 579},

		// Extreme values
		{A: New(math.MinInt64, Metric), B: New(math.MaxInt64, Metric), Expect: -1},
		{A: New(math.MaxInt64, Metric), B: New(math.MinInt64, Metric), Expect: -1},
	}

	for _, test := range tests {
		result := New(0, Metric)
		result.Add(*test.A)
		assertEqual(t, test.A.Int64(), result.Int64(), "Adding %v to zero", test.A)

		result.Add(*test.B)
		assertEqual(t, test.Expect, result.Int64(), "Adding %v + %v", test.A, test.B)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		In          string
		ExpectBytes int64
		ExpectBase  Base
		ExpectErr   string
	}{
		// Invalid values should produce errors.
		{In: "", ExpectErr: "empty string"},
		{In: " B", ExpectErr: "must start with a number"},
		{In: "9223372036854775808", ExpectErr: "value exceeds 64 bits"},
		{In: "8.0 EiB", ExpectErr: "value exceeds 64 bits"},
		{In: "1 tUb", ExpectErr: `"tUb" is not a valid byte quantity`},

		// Zero parses correctly.
		{In: "0", ExpectBytes: 0, ExpectBase: Metric},
		{In: "-0", ExpectBytes: 0, ExpectBase: Metric},
		{In: "0 B", ExpectBytes: 0, ExpectBase: Metric},
		{In: "0mib", ExpectBytes: 0, ExpectBase: Binary},

		// Missing leading or trailing zeroes
		{In: ". B", ExpectErr: "must start with a number"},
		{In: "-. B", ExpectErr: "must start with a number"},
		{In: ".1 kB", ExpectBytes: 100, ExpectBase: Metric},
		{In: "-.1 kB", ExpectBytes: -100, ExpectBase: Metric},
		{In: "1. kB", ExpectBytes: 1000, ExpectBase: Metric},

		// Extra leading or trailing zeroes
		{In: ".10000 kB", ExpectBytes: 100, ExpectBase: Metric},
		{In: "0000.1 kB", ExpectBytes: 100, ExpectBase: Metric},
		{In: "-0000.1 kB", ExpectBytes: -100, ExpectBase: Metric},
		{In: "0001.0000 kB", ExpectBytes: 1000, ExpectBase: Metric},

		// Min values parse correctly.
		{In: "-9223372036854775808", ExpectBytes: math.MinInt64, ExpectBase: Metric},
		{In: "-9.223372036854775808eb", ExpectBytes: math.MinInt64, ExpectBase: Metric},
		{In: "-8 EiB", ExpectBytes: math.MinInt64, ExpectBase: Binary},

		// Max values parse correctly, even with extreme precision.
		{In: "9223372036854775807", ExpectBytes: math.MaxInt64, ExpectBase: Metric},
		{In: "9.223372036854775807eb", ExpectBytes: math.MaxInt64, ExpectBase: Metric},
		{In: "7.99999999999999999914 EiB", ExpectBytes: math.MaxInt64, ExpectBase: Binary},

		// Metric and binary suffixes produce different results.
		{In: "123.456g", ExpectBytes: 123_456_000_000, ExpectBase: Metric},
		{In: "123.456 GB", ExpectBytes: 123_456_000_000, ExpectBase: Metric},
		{In: "123.456 GiB", ExpectBytes: 132_559_870_623, ExpectBase: Binary},
	}

	for _, test := range tests {
		size, err := Parse(test.In)

		if test.ExpectErr != "" {
			expectErr := fmt.Sprintf("can't convert %q to size: %s", test.In, test.ExpectErr)
			assertEqualErr(t, expectErr, err, "Error for %q", test.In)
			continue
		}

		if !assertNoErr(t, err, "Unxpected error for %q", test.In) {
			continue
		}
		assertEqual(t, test.ExpectBytes, size.Int64(), "Byte count for %q", test.In)
		assertEqual(t, test.ExpectBase, size.Base, "Base for %q", test.In)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		In     *Size
		Expect string
	}{
		// Zero values
		{In: New(0, Metric), Expect: "0 B"},
		{In: New(0, Binary), Expect: "0 B"},

		// Minimum value representable by int64: -2**62
		{In: New(math.MinInt64, Metric), Expect: "-9223372036854775808 B"},
		{In: New(math.MinInt64, Binary), Expect: "-8 EiB"},

		// Maximum value representable by int64: 2**63-1
		{In: New(math.MaxInt64, Metric), Expect: "9223372036854775807 B"},
		{In: New(math.MaxInt64, Binary), Expect: "9223372036854775807 B"},

		// Thresholds between Metric suffixes
		{In: New(1*Byte, Metric), Expect: "1 B"},
		{In: New(999*Byte, Metric), Expect: "999 B"},
		{In: New(1*KB, Metric), Expect: "1 kB"},
		{In: New(999*KB, Metric), Expect: "999 kB"},
		{In: New(1*MB, Metric), Expect: "1 MB"},
		{In: New(999*MB, Metric), Expect: "999 MB"},
		{In: New(1*GB, Metric), Expect: "1 GB"},
		{In: New(999*GB, Metric), Expect: "999 GB"},
		{In: New(1*TB, Metric), Expect: "1 TB"},
		{In: New(999*TB, Metric), Expect: "999 TB"},
		{In: New(1*PB, Metric), Expect: "1 PB"},
		{In: New(999*PB, Metric), Expect: "999 PB"},
		{In: New(1000*PB, Metric), Expect: "1 EB"},

		// Thresholds between Binary suffixes
		{In: New(1*Byte, Binary), Expect: "1 B"},
		{In: New(1023*Byte, Binary), Expect: "1023 B"},
		{In: New(1*KiB, Binary), Expect: "1 KiB"},
		{In: New(1023*KiB, Binary), Expect: "1023 KiB"},
		{In: New(1*MiB, Binary), Expect: "1 MiB"},
		{In: New(1023*MiB, Binary), Expect: "1023 MiB"},
		{In: New(1*GiB, Binary), Expect: "1 GiB"},
		{In: New(1023*GiB, Binary), Expect: "1023 GiB"},
		{In: New(1*TiB, Binary), Expect: "1 TiB"},
		{In: New(1023*TiB, Binary), Expect: "1023 TiB"},
		{In: New(1*PiB, Binary), Expect: "1 PiB"},
		{In: New(1023*PiB, Binary), Expect: "1023 PiB"},
		{In: New(1024*PiB, Binary), Expect: "1 EiB"},
	}

	for _, test := range tests {
		str := test.In.String()
		assertEqual(t, test.Expect, str, "Formatting (%d, %v)",
			test.In.Int64(), test.In.Base)
	}
}

func assertNoErr(t *testing.T, err error, message string, args ...interface{}) bool {
	t.Helper()
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
	t.Helper()
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
	t.Helper()
	if reflect.DeepEqual(expect, actual) {
		return true
	}
	t.Error(fmt.Sprintf(message, args...),
		"\n    Expected:", expect,
		"\n    Actual:  ", actual)
	return false
}
