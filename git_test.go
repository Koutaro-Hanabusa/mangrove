package mangrove

import (
	"reflect"
	"testing"
)

func TestParseLines(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"空入力はnil", "", nil},
		{"1行のみ", "hello", []string{"hello"}},
		{"複数行の分割", "a\nb\nc", []string{"a", "b", "c"}},
		{"空行を除外", "a\n\nb\n\nc", []string{"a", "b", "c"}},
		{"前後の空白をトリム", "  a  \n  b  ", []string{"a", "b"}},
		{"末尾改行を処理", "a\n", []string{"a"}},
		{"全て空行はnil", "\n\n\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("parseLines(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}
