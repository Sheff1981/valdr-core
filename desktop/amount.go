package desktop

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Sheff1981/valdr-core/config"
)

var ErrInvalidVDRAmt = errors.New("invalid VDR amount")

func ParseVDR(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") ||
		strings.HasPrefix(value, "+") {
		return 0, ErrInvalidVDRAmt
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidVDRAmt
	}
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || whole > math.MaxUint64/config.AtomicUnitsPerVDR {
		return 0, ErrInvalidVDRAmt
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 8 {
			return 0, ErrInvalidVDRAmt
		}
		for _, ch := range fraction {
			if ch < '0' || ch > '9' {
				return 0, ErrInvalidVDRAmt
			}
		}
	}
	fraction += strings.Repeat("0", 8-len(fraction))
	frac := uint64(0)
	if fraction != "" {
		frac, err = strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalidVDRAmt
		}
	}
	base := whole * config.AtomicUnitsPerVDR
	if math.MaxUint64-base < frac {
		return 0, ErrInvalidVDRAmt
	}
	return base + frac, nil
}

func FormatVDR(value uint64) string {
	whole := value / config.AtomicUnitsPerVDR
	fraction := value % config.AtomicUnitsPerVDR
	return fmt.Sprintf("%d.%08d", whole, fraction)
}
