package util

import (
	"fmt"
	"strings"
	"time"
)

// Duration ...
type Duration time.Duration

// UnmarshalText ...
func (d *Duration) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("duration: cannot parse %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

// MarshalText ...
func (d *Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(*d).String()), nil
}
