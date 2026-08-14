package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalid = errors.New("invalid money value")

func ParseMajor(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalid
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, ErrInvalid
	}

	whole := parts[0]
	if whole == "" || !allDigits(whole) {
		return 0, ErrInvalid
	}

	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
		if frac == "" || len(frac) > 2 || !allDigits(frac) {
			return 0, ErrInvalid
		}
	}

	intPart, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}

	fracPart := int64(0)
	switch len(frac) {
	case 1:
		fracPart = int64(frac[0]-'0') * 10
	case 2:
		fracPart = int64(frac[0]-'0')*10 + int64(frac[1]-'0')
	}

	if intPart > (1<<63-1-fracPart)/100 {
		return 0, ErrInvalid
	}

	return intPart*100 + fracPart, nil
}

func FormatMajor(sen int64) string {
	neg := sen < 0
	if neg {
		sen = -sen
	}

	s := fmt.Sprintf("%d.%02d", sen/100, sen%100)
	if neg {
		return "-" + s
	}
	return s
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
