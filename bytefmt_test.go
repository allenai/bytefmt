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

func TestFormat(t *testing.T) {
	tests := []struct {
		In     *Size
		Format string
		Expect string
	}{
		// Zero values
		{In: New(0, Metric), Format: "%v", Expect: "0 B"},
		{In: New(0, Binary), Format: "%v", Expect: "0 B"},

		// Minimum value representable by int64: -2**62
		{In: New(math.MinInt64, Metric), Format: "%g", Expect: "-9.223372036854776 EB"},
		{In: New(math.MinInt64, Binary), Format: "%g", Expect: "-8 EiB"},

		// Maximum value representable by int64: 2**63-1
		{In: New(math.MaxInt64, Metric), Format: "%g", Expect: "9.223372036854776 EB"},
		{In: New(math.MaxInt64, Binary), Format: "%g", Expect: "8 EiB"},

		// Thresholds between Metric suffixes
		{In: New(1*Byte, Metric), Format: "%v", Expect: "1 B"},
		{In: New(999*Byte, Metric), Format: "%v", Expect: "999 B"},
		{In: New(1*KB, Metric), Format: "%v", Expect: "1 kB"},
		{In: New(999*KB, Metric), Format: "%v", Expect: "999 kB"},
		{In: New(1*MB, Metric), Format: "%v", Expect: "1 MB"},
		{In: New(999*MB, Metric), Format: "%v", Expect: "999 MB"},
		{In: New(1*GB, Metric), Format: "%v", Expect: "1 GB"},
		{In: New(999*GB, Metric), Format: "%v", Expect: "999 GB"},
		{In: New(1*TB, Metric), Format: "%v", Expect: "1 TB"},
		{In: New(999*TB, Metric), Format: "%v", Expect: "999 TB"},
		{In: New(1*PB, Metric), Format: "%v", Expect: "1 PB"},
		{In: New(999*PB, Metric), Format: "%v", Expect: "999 PB"},
		{In: New(1000*PB, Metric), Format: "%v", Expect: "1 EB"},

		// Thresholds between Binary suffixes
		{In: New(1*Byte, Binary), Format: "%v", Expect: "1 B"},
		{In: New(1023*Byte, Binary), Format: "%v", Expect: "1023 B"},
		{In: New(1*KiB, Binary), Format: "%v", Expect: "1 KiB"},
		{In: New(1023*KiB, Binary), Format: "%v", Expect: "1023 KiB"},
		{In: New(1*MiB, Binary), Format: "%v", Expect: "1 MiB"},
		{In: New(1023*MiB, Binary), Format: "%v", Expect: "1023 MiB"},
		{In: New(1*GiB, Binary), Format: "%v", Expect: "1 GiB"},
		{In: New(1023*GiB, Binary), Format: "%v", Expect: "1023 GiB"},
		{In: New(1*TiB, Binary), Format: "%v", Expect: "1 TiB"},
		{In: New(1023*TiB, Binary), Format: "%v", Expect: "1023 TiB"},
		{In: New(1*PiB, Binary), Format: "%v", Expect: "1 PiB"},
		{In: New(1023*PiB, Binary), Format: "%v", Expect: "1023 PiB"},
		{In: New(1024*PiB, Binary), Format: "%v", Expect: "1 EiB"},

		// Loss of precision.
		{In: New(1001*Byte, Metric), Format: "%v", Expect: "1.001 kB"},
		{In: New(1025*Byte, Binary), Format: "%v", Expect: "1.001 KiB"},
		{In: New(123456*Byte, Metric), Format: "%v", Expect: "123.5 kB"},
		{In: New(123456*Byte, Binary), Format: "%v", Expect: "120.6 KiB"},
		{In: New(1500*Byte, Metric), Format: "%v", Expect: "1.5 kB"},
		{In: New(1501*Byte, Metric), Format: "%v", Expect: "1.501 kB"},
		{In: New(1499*Byte, Metric), Format: "%v", Expect: "1.499 kB"},

		// Rounding with Metric suffixes.
		{In: New(14995*Byte, Metric), Format: "%v", Expect: "14.99 kB"},
		{In: New(14996*Byte, Metric), Format: "%v", Expect: "15 kB"},
		{In: New(15000*Byte, Metric), Format: "%v", Expect: "15 kB"},
		{In: New(15004*Byte, Metric), Format: "%v", Expect: "15 kB"},
		{In: New(15005*Byte, Metric), Format: "%v", Expect: "15.01 kB"},

		// Rounding with Binary suffixes.
		{In: New(16378*Byte, Binary), Format: "%v", Expect: "15.99 KiB"},
		{In: New(16379*Byte, Binary), Format: "%v", Expect: "16 KiB"},
		{In: New(16384*Byte, Binary), Format: "%v", Expect: "16 KiB"},
		{In: New(16389*Byte, Binary), Format: "%v", Expect: "16 KiB"},
		{In: New(16390*Byte, Binary), Format: "%v", Expect: "16.01 KiB"},

		// 4 significant figures with Metric suffixes.
		{In: New(1*Byte, Metric), Format: "%v", Expect: "1 B"},
		{In: New(11*Byte, Metric), Format: "%v", Expect: "11 B"},
		{In: New(111*Byte, Metric), Format: "%v", Expect: "111 B"},
		{In: New(1111*Byte, Metric), Format: "%v", Expect: "1.111 kB"},
		{In: New(11111*Byte, Metric), Format: "%v", Expect: "11.11 kB"},
		{In: New(111111*Byte, Metric), Format: "%v", Expect: "111.1 kB"},
		{In: New(1111111*Byte, Metric), Format: "%v", Expect: "1.111 MB"},
		{In: New(11111111*Byte, Metric), Format: "%v", Expect: "11.11 MB"},
		{In: New(111111111*Byte, Metric), Format: "%v", Expect: "111.1 MB"},
		{In: New(1111111111*Byte, Metric), Format: "%v", Expect: "1.111 GB"},
		{In: New(11111111111*Byte, Metric), Format: "%v", Expect: "11.11 GB"},
		{In: New(111111111111*Byte, Metric), Format: "%v", Expect: "111.1 GB"},
		{In: New(1111111111111*Byte, Metric), Format: "%v", Expect: "1.111 TB"},
		{In: New(11111111111111*Byte, Metric), Format: "%v", Expect: "11.11 TB"},
		{In: New(111111111111111*Byte, Metric), Format: "%v", Expect: "111.1 TB"},
		{In: New(1111111111111111*Byte, Metric), Format: "%v", Expect: "1.111 PB"},
		{In: New(11111111111111111*Byte, Metric), Format: "%v", Expect: "11.11 PB"},
		{In: New(111111111111111111*Byte, Metric), Format: "%v", Expect: "111.1 PB"},
		{In: New(1111111111111111111*Byte, Metric), Format: "%v", Expect: "1.111 EB"},

		// 4 significant figures with Binary suffixes.
		{In: New(1*Byte, Binary), Format: "%v", Expect: "1 B"},
		{In: New(11*Byte, Binary), Format: "%v", Expect: "11 B"},
		{In: New(111*Byte, Binary), Format: "%v", Expect: "111 B"},
		{In: New(1111*Byte, Binary), Format: "%v", Expect: "1.085 KiB"},
		{In: New(11111*Byte, Binary), Format: "%v", Expect: "10.85 KiB"},
		{In: New(111111*Byte, Binary), Format: "%v", Expect: "108.5 KiB"},
		{In: New(1111111*Byte, Binary), Format: "%v", Expect: "1.06 MiB"},
		{In: New(11111111*Byte, Binary), Format: "%v", Expect: "10.6 MiB"},
		{In: New(111111111*Byte, Binary), Format: "%v", Expect: "106 MiB"},
		{In: New(1111111111*Byte, Binary), Format: "%v", Expect: "1.035 GiB"},
		{In: New(11111111111*Byte, Binary), Format: "%v", Expect: "10.35 GiB"},
		{In: New(111111111111*Byte, Binary), Format: "%v", Expect: "103.5 GiB"},
		{In: New(1111111111111*Byte, Binary), Format: "%v", Expect: "1.011 TiB"},
		{In: New(11111111111111*Byte, Binary), Format: "%v", Expect: "10.11 TiB"},
		{In: New(111111111111111*Byte, Binary), Format: "%v", Expect: "101.1 TiB"},
		{In: New(1111111111111111*Byte, Binary), Format: "%v", Expect: "1011 TiB"},
		{In: New(11111111111111111*Byte, Binary), Format: "%v", Expect: "9.869 PiB"},
		{In: New(111111111111111111*Byte, Binary), Format: "%v", Expect: "98.69 PiB"},
		{In: New(1111111111111111111*Byte, Binary), Format: "%v", Expect: "986.9 PiB"},

		// %f uses number of decimal places for precision.
		{In: New(1111*Byte, Metric), Format: "%.0f", Expect: "1 kB"},
		{In: New(1111*Byte, Metric), Format: "%.1f", Expect: "1.1 kB"},
		{In: New(1111*Byte, Metric), Format: "%.2f", Expect: "1.11 kB"},
		{In: New(1111*Byte, Metric), Format: "%.3f", Expect: "1.111 kB"},
		{In: New(1111*Byte, Metric), Format: "%.4f", Expect: "1.1110 kB"},

		// %f defaults to precision of 6.
		{In: New(1111*Byte, Metric), Format: "%f", Expect: "1.111000 kB"},

		// %g uses signficant figures for precision.
		{In: New(1111*Byte, Metric), Format: "%.0g", Expect: "1 kB"},
		{In: New(1111*Byte, Metric), Format: "%.1g", Expect: "1 kB"},
		{In: New(1111*Byte, Metric), Format: "%.2g", Expect: "1.1 kB"},
		{In: New(1111*Byte, Metric), Format: "%.3g", Expect: "1.11 kB"},
		{In: New(1111*Byte, Metric), Format: "%.4g", Expect: "1.111 kB"},
		{In: New(1111*Byte, Metric), Format: "%.5g", Expect: "1.111 kB"},

		// %g defaults to exact precision.
		{In: New(1111111111111*Byte, Metric), Format: "%g", Expect: "1.111111111111 TB"},

		// %v formats as %.4g.
		{In: New(1111*Byte, Metric), Format: "%v", Expect: "1.111 kB"},

		// Invalid verb.
		{In: New(Byte, Metric), Format: "%s", Expect: "%!s(size=1)"},
	}

	for _, test := range tests {
		str := fmt.Sprintf(test.Format, test.In)
		assertEqual(t, test.Expect, str, "Formatting (%d, %v) with format %q",
			test.In.Int64(), test.In.Base, test.Format)
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
