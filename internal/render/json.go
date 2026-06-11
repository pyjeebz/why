package render

import (
	"encoding/json"
	"io"

	"github.com/pyjeebz/why/internal/trail"
)

// JSON writes the trail as indented JSON: the stable contract for
// scripts and agents building on dig.
func JSON(w io.Writer, t trail.Trail) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(t)
}
