package main

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ByteSize is an int64 that unmarshals from human-readable strings like "2G", "500M", "1.5G".
// Plain integers (e.g. 2147483648) are also accepted.
type ByteSize int64

var byteSizeSuffixes = []struct {
	suffix string
	mult   int64
}{
	{"TB", 1024 * 1024 * 1024 * 1024},
	{"GB", 1024 * 1024 * 1024},
	{"MB", 1024 * 1024},
	{"KB", 1024},
	{"T", 1024 * 1024 * 1024 * 1024},
	{"G", 1024 * 1024 * 1024},
	{"M", 1024 * 1024},
	{"K", 1024},
	{"B", 1},
}

func parseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	for _, entry := range byteSizeSuffixes {
		if strings.HasSuffix(s, entry.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, entry.suffix))
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid byte size %q", s)
			}
			return int64(f * float64(entry.mult)), nil
		}
	}
	// No suffix — plain number
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q", s)
	}
	return int64(f), nil
}

func (b *ByteSize) UnmarshalYAML(value *yaml.Node) error {
	result, err := parseByteSize(value.Value)
	if err != nil {
		return err
	}
	*b = ByteSize(result)
	return nil
}

// humanizeBytes converts bytes to a human-readable format
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
