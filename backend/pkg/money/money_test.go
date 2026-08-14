package money

import "testing"

func TestParseMajor(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{"0", 0, true},
		{"1", 100, true},
		{"600000", 60000000, true},
		{"600000.5", 60000050, true},
		{"600000.50", 60000050, true},
		{"0.01", 1, true},
		{"0.1", 10, true},
		{" 42 ", 4200, true},
		{"999999999999.99", 99999999999999, true},
		{"", 0, false},
		{"   ", 0, false},
		{"abc", 0, false},
		{"1.234", 0, false},
		{"1.", 0, false},
		{".5", 0, false},
		{"-5", 0, false},
		{"+5", 0, false},
		{"1,000", 0, false},
		{"1.2.3", 0, false},
		{"1e5", 0, false},
		{"100000000000000000", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMajor(tt.input)
			if tt.ok {
				if err != nil {
					t.Fatalf("expected success, got error %v", err)
				}
				if got != tt.want {
					t.Errorf("expected %d, got %d", tt.want, got)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got %d", got)
			}
		})
	}
}

func TestFormatMajor(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0.00"},
		{1, "0.01"},
		{100, "1.00"},
		{60000000, "600000.00"},
		{60000050, "600000.50"},
		{123456789, "1234567.89"},
		{-500, "-5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := FormatMajor(tt.input); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseFormatRoundTrip(t *testing.T) {
	for _, s := range []string{"0.01", "1.00", "600000.50", "999999999999.99"} {
		sen, err := ParseMajor(s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		if got := FormatMajor(sen); got != s {
			t.Errorf("round trip %q: got %q", s, got)
		}
	}
}
