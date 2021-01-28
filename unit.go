package bytefmt

import (
	"fmt"
	"strconv"
	"strings"
)

// Unit defines SI scales at which bytes can be rendered.
type Unit int64

func (u Unit) String() string {
	if s, ok := suffixes[u]; ok {
		return s
	}
	return strconv.FormatInt(int64(u), 64)
}

// Byte is the base unit for binary size
const Byte Unit = 1

// Metric units define powers of 1000 using SI decimal prefixes per NIST.
// https://physicnist.gov/cuu/Units/prefixehtml
const (
	KB = 1000 * Byte
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB
	EB = 1000 * PB
)

// Binary units define powers of 2^10 using SI binary prefixes per the IEC.
// https://physicnist.gov/cuu/Units/binary.html
const (
	KiB = 1024 * Byte
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
	EiB = 1024 * PiB
)

// Automatic unit sizes detect the best-fit unit for a given value.
const (
	Metric Unit = 0
	Binary Unit = -1
)

var suffixes = map[Unit]string{
	Byte: "B",
	KB:   "kB", // Intentionally lower-case per SI standard.
	MB:   "MB",
	GB:   "GB",
	TB:   "TB",
	PB:   "PB",
	EB:   "EB",
	KiB:  "KiB",
	MiB:  "MiB",
	GiB:  "GiB",
	TiB:  "TiB",
	PiB:  "PiB",
	EiB:  "EiB",
}

func mustValidateUnit(u Unit) {
	switch u {
	case Metric, Binary:
		// Auto-detect unit
	case Byte, KB, MB, GB, TB, PB, EB, KiB, MiB, GiB, TiB, PiB, EiB:
		// Specific unit size
	default:
		panic("bytefmt: invalid unit size")
	}
}

func parseUnit(s string) (Unit, error) {
	switch strings.ToLower(s) {
	case "b", "":
		return Byte, nil
	case "kb", "k":
		return KB, nil
	case "mb", "m":
		return MB, nil
	case "gb", "g":
		return GB, nil
	case "tb", "t":
		return TB, nil
	case "pb", "p":
		return PB, nil
	case "eb", "e":
		return EB, nil
	case "kib":
		return KiB, nil
	case "mib":
		return MiB, nil
	case "gib":
		return GiB, nil
	case "tib":
		return TiB, nil
	case "pib":
		return PiB, nil
	case "eib":
		return EiB, nil
	default:
		return 0, fmt.Errorf("%q is not a valid byte quantity", s)
	}
}

func inferUnit(bytes int64, unit Unit) Unit {
	if bytes < 0 {
		bytes = -bytes
	}

	switch unit {
	case Metric:
		switch {
		case bytes < int64(KB):
			return Byte
		case bytes < int64(MB):
			return KB
		case bytes < int64(GB):
			return MB
		case bytes < int64(TB):
			return GB
		case bytes < int64(PB):
			return TB
		case bytes < int64(EB):
			return PB
		default:
			return EB
		}

	case Binary:
		switch {
		case bytes < int64(KiB):
			return Byte
		case bytes < int64(MiB):
			return KiB
		case bytes < int64(GiB):
			return MiB
		case bytes < int64(TiB):
			return GiB
		case bytes < int64(PiB):
			return TiB
		case bytes < int64(EiB):
			return PiB
		default:
			return EiB
		}

	default:
		return unit
	}
}
