package cache

import "testing"

type payload struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

func TestRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var missing payload
	if Get("github/o__r/abc.json", &missing) {
		t.Fatal("Get on empty cache reported a hit")
	}

	want := payload{Name: "x", N: 7}
	if err := Put("github/o__r/abc.json", want); err != nil {
		t.Fatal(err)
	}

	var got payload
	if !Get("github/o__r/abc.json", &got) {
		t.Fatal("Get missed after Put")
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
