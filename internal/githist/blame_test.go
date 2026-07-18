package githist

import "testing"

func TestParseBlame_shaAndSubjectPerLine(t *testing.T) {
	const a = "1111111111111111111111111111111111111111"
	const b = "2222222222222222222222222222222222222222"
	// git blame --line-porcelain repeats the full commit block for every line.
	out := a + " 1 1 1\nauthor Jane\nsummary fix: race in loop\nfilename f.go\n\tfirst line\n" +
		a + " 2 2 1\nauthor Jane\nsummary fix: race in loop\nfilename f.go\n\tsecond line\n" +
		b + " 3 3 1\nauthor Sam\nsummary refactor thing\nfilename f.go\n\tthird line\n"

	got := parseBlame(out)
	if len(got) != 3 {
		t.Fatalf("want 3 blame lines, got %d: %+v", len(got), got)
	}
	if got[0].SHA != a || got[0].N != 1 || got[0].Subject != "fix: race in loop" {
		t.Errorf("line 1 = %+v", got[0])
	}
	if got[2].SHA != b || got[2].N != 3 || got[2].Subject != "refactor thing" {
		t.Errorf("line 3 = %+v", got[2])
	}
}
