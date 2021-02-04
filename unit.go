package bytefmt

import (
	"fmt"
	"strings"
)

// Base is a radix by which byte quantities can be scaled.
type Base int

const (
	// Metric units define powers of 1000 using SI decimal prefixes per NIST.
	// https://physics.nist.gov/cuu/Units/prefixes.html
	Metric Base = 1000

	// Binary units define powers of 2^10 using SI binary prefixes per the IEC.
	// https://physics.nist.gov/cuu/Units/binary.html
	Binary Base = 1024
)

// Byte is the unscaled unit for bytes.
const Byte int64 = 1

// Metric suffixes scale quantities by powers of 1000.
const (
	KB = 1000 * Byte
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB
)

var metricSuffixes = [...]string{
	"B",
	"kB", // Intentionally lower-case per SI standard.
	"MB",
	"GB",
	"TB",
	"PB",
	"EB",
}

// Binary suffixes scale quantities by powers of 1024.
const (
	KiB = 1024 * Byte
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

var binarySuffixes = [...]string{
	"B",
	"KiB",
	"MiB",
	"GiB",
	"TiB",
	"PiB",
	"EiB",
}

func parseSuffix(s string) (int, Base, error) {
	switch strings.ToLower(s) {
	case "b", "":
		return 0, Metric, nil
	case "kb", "k":
		return 1, Metric, nil
	case "mb", "m":
		return 2, Metric, nil
	case "gb", "g":
		return 3, Metric, nil
	case "tb", "t":
		return 4, Metric, nil
	case "pb", "p":
		return 5, Metric, nil
	case "eb", "e":
		return 6, Metric, nil
	case "kib":
		return 1, Binary, nil
	case "mib":
		return 2, Binary, nil
	case "gib":
		return 3, Binary, nil
	case "tib":
		return 4, Binary, nil
	case "pib":
		return 5, Binary, nil
	case "eib":
		return 6, Binary, nil
	default:
		return 0, Metric, fmt.Errorf("%q is not a valid byte quantity", s)
	}
}
