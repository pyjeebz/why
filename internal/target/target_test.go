package target

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.go")
	if err := os.WriteFile(file, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	colonFile := filepath.Join(dir, "weird:name")
	if err := os.WriteFile(colonFile, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		arg     string
		want    Target
		wantErr bool
	}{
		{arg: file + ":142", want: Target{Path: file, Start: 142, End: 142}},
		{arg: file + ":120-160", want: Target{Path: file, Start: 120, End: 160}},
		{arg: file, want: Target{Path: file}},
		{arg: colonFile, want: Target{Path: colonFile}},
		{arg: file + ":0", wantErr: true},
		{arg: file + ":50-20", wantErr: true},
		{arg: filepath.Join(dir, "missing.go") + ":3", wantErr: true},
		{arg: dir, wantErr: true},
		{arg: "", wantErr: true},
	}

	for _, tc := range tests {
		got, err := Parse(tc.arg)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Parse(%q): expected error, got %+v", tc.arg, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", tc.arg, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tc.arg, got, tc.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		in   Target
		want string
	}{
		{Target{Path: "a.go"}, "a.go"},
		{Target{Path: "a.go", Start: 5, End: 5}, "a.go:5"},
		{Target{Path: "a.go", Start: 5, End: 9}, "a.go:5-9"},
	}
	for _, tc := range tests {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
}
