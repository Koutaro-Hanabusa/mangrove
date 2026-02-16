package mangrove

import (
	"testing"
)

func TestParseLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "normal multiline input",
			input: "main\ndevelop\nfeature/foo\n",
			want:  []string{"main", "develop", "feature/foo"},
		},
		{
			name:  "empty lines filtered",
			input: "main\n\ndevelop\n\n",
			want:  []string{"main", "develop"},
		},
		{
			name:  "single line",
			input: "main\n",
			want:  []string{"main"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "only whitespace lines",
			input: "\n  \n\t\n",
			want:  nil,
		},
		{
			name:  "lines with extra whitespace",
			input: "  main  \n  develop  \n",
			want:  []string{"main", "develop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("parseLines(%q) returned %d lines, want %d\ngot: %v\nwant: %v",
					tt.input, len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
