package utils

import "testing"

func TestSplitStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"empty", "", nil, true},
		{"single", "200", []int{200}, false},
		{"multi", "200502503", []int{200, 502, 503}, false},
		{"zero-status", "000", []int{0}, false},
		{"non-digit", "abc", nil, true},
		{"partial-digit", "200abc500", nil, true},
		{"bad-len", "2005", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SplitStatus(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SplitStatus(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("SplitStatus(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		got := MergeMaps(map[string]int{}, map[string]int{})
		if len(got) != 0 {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("one empty", func(t *testing.T) {
		got := MergeMaps(map[string]int{"a": 1}, map[string]int{})
		if got["a"] != 1 || len(got) != 1 {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("key conflict m2 wins", func(t *testing.T) {
		got := MergeMaps(map[string]int{"a": 1, "b": 2}, map[string]int{"a": 99})
		if got["a"] != 99 || got["b"] != 2 || len(got) != 2 {
			t.Fatalf("got %v", got)
		}
	})
}
