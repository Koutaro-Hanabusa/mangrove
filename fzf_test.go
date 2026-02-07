package mangrove

import (
	"reflect"
	"testing"
)

func TestReorderWithDefault(t *testing.T) {
	tests := []struct {
		name        string
		items       []string
		defaultItem string
		expect      []string
	}{
		{"デフォルト項目を先頭に移動", []string{"a", "b", "c"}, "b", []string{"b", "a", "c"}},
		{"既に先頭の場合は順序維持", []string{"a", "b", "c"}, "a", []string{"a", "b", "c"}},
		{"存在しない項目は順序変更なし", []string{"a", "b", "c"}, "z", []string{"a", "b", "c"}},
		{"デフォルト空文字はそのまま返す", []string{"a", "b", "c"}, "", []string{"a", "b", "c"}},
		{"空リストは空リストを返す", []string{}, "a", []string{}},
		{"要素1つでも正しく処理", []string{"single"}, "single", []string{"single"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderWithDefault(tt.items, tt.defaultItem)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("reorderWithDefault(%v, %q) = %v, want %v", tt.items, tt.defaultItem, got, tt.expect)
			}
		})
	}
}
