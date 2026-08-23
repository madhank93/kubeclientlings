package exercises_test

import (
	"strings"
	"testing"

	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
)

func TestStripFencesDropsOnlyTaggedBlocks(t *testing.T) {
	md := strings.Join([]string{
		"# Title",
		"",
		"```mermaid",
		"graph LR",
		"  a --> b",
		"```",
		"",
		"```ascii",
		"a -> b",
		"```",
		"",
		"```go",
		"ch := make(chan int)",
		"```",
	}, "\n")

	got := exercises.StripFences(md, "mermaid")

	if strings.Contains(got, "graph LR") {
		t.Errorf("mermaid block survived:\n%s", got)
	}
	for _, want := range []string{"# Title", "```ascii", "a -> b", "```go", "ch := make(chan int)"} {
		if !strings.Contains(got, want) {
			t.Errorf("stripping mermaid removed %q:\n%s", want, got)
		}
	}
	if opens, closes := strings.Count(got, "```ascii"), strings.Count(got, "```"); opens != 1 || closes != 4 {
		t.Errorf("fences unbalanced after stripping: %d ascii opens, %d fence lines\n%s", opens, closes, got)
	}
}

func TestStripFencesIsANoOpWithoutTheTag(t *testing.T) {
	md := "text\n\n```go\nx := 1\n```\n"
	if got := exercises.StripFences(md, "mermaid"); got != md {
		t.Errorf("unrelated markdown changed:\nwant %q\ngot  %q", md, got)
	}
}
