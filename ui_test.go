package mangrove

import (
	"testing"
)

func TestJoinParts(t *testing.T) {
	tests := []struct {
		name   string
		parts  []string
		expect string
	}{
		{"空スライスは空文字列", []string{}, ""},
		{"要素1つはそのまま返す", []string{"ahead"}, "ahead"},
		{"要素2つをandで結合", []string{"ahead", "behind"}, "ahead and behind"},
		{"要素3つ目以降は無視される", []string{"a", "b", "c"}, "a and b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinParts(tt.parts)
			if got != tt.expect {
				t.Errorf("joinParts(%v) = %q, want %q", tt.parts, got, tt.expect)
			}
		})
	}
}
