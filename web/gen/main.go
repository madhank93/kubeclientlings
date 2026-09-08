// Command gen renders the KubeClientlings catalog data (catalog.ts, per-exercise
// detail markdown, and per-topic chapters) from info.toml + exercise sources +
// per-topic READMEs. Run from the repo root: `go run ./web/gen`.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
)

const (
	infoFile    = "info.toml"
	dataDir     = "web/src/data"                // catalog.ts for /catalog
	detailsDir  = "web/src/data/lesson-details" // per-exercise detail markdown
	chaptersDir = "web/src/data/chapters"       // per-topic chapter markdown for the catalog
	exerciseR   = "exercises"
	// Popups link to the worked solution instead of embedding it.
	repoBlob = "https://github.com/madhank93/kubeclientlings/blob/main"
)

// tier groups topics into a beginner→advanced section.
type tier struct {
	name   string
	topics []string
}

var tiers = []tier{
	{"Beginner · Setup", []string{"setup"}},
	{"Beginner · Core Workloads", []string{"pods", "deployments", "services"}},
	{"Intermediate · Requests & Watch", []string{"options", "watch"}},
	{"Intermediate · Dynamic & CRDs", []string{"dynamic", "crds"}},
	{"Advanced · Controller Machinery", []string{"informers", "workqueue", "controllers"}},
	{"Advanced · Advanced Writes", []string{"ssa", "subresources", "finalizers"}},
	{"Advanced · Webhooks & Testing", []string{"webhooks", "testing"}},
	{"Advanced · Operator", []string{"operator"}},
}

// tierColors gives every topic in a tier the same chip color on /catalog.
var tierColors = []string{
	"#4fa86d", "#3b9eff", "#7c6af5", "#d29922",
	"#9b5de5", "#c53030", "#e36f0e", "#1f8a9c",
}

// prettyNames overrides the default underscore→Title Case topic label.
var prettyNames = map[string]string{
	"crds": "CRDs", "ssa": "Server-Side Apply",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run() error {
	exs, err := exercises.List(infoFile)
	if err != nil {
		return err
	}
	byTopic := map[string][]exercises.Exercise{}
	for _, e := range exs {
		t := topicOf(e.Path)
		byTopic[t] = append(byTopic[t], e)
	}

	// Every topic must be placed in a tier, or its exercises silently vanish
	// from the site.
	if err := checkCoverage(byTopic); err != nil {
		return err
	}
	if err := writeCatalog(byTopic); err != nil {
		return err
	}
	return writeChapters()
}

// writeChapters renders each topic README as chapter markdown for the catalog
// modal. The README stays the single source: the TUI and GitHub read it
// directly, the site reads this rendering of it.
func writeChapters() error {
	if err := os.RemoveAll(chaptersDir); err != nil {
		return err
	}
	if err := os.MkdirAll(chaptersDir, 0o755); err != nil {
		return err
	}

	for _, ti := range tiers {
		for _, topic := range ti.topics {
			body := readme(topic)
			if body == "" {
				continue
			}
			var b strings.Builder
			fmt.Fprintf(&b, "---\ntitle: %s\n---\n\n", pretty(topic))
			fmt.Fprintf(&b, "%s\n", chapterHTML(exercises.StripFences(body, "ascii")))
			if err := os.WriteFile(filepath.Join(chaptersDir, topic+".md"), []byte(b.String()), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// chapterHTML prepares a chapter for the catalog modal: neighbour links become
// catalog deep links, and mermaid fences become the <pre class="mermaid"> the
// modal's lazy mermaid pass renders (a fence would stay a code block, since the
// modal injects its HTML after the page has loaded).
func chapterHTML(md string) string {
	md = topicLink.ReplaceAllString(md, "](/catalog/?chapter=$1)")
	return mermaidFence.ReplaceAllStringFunc(md, func(block string) string {
		src := mermaidFence.FindStringSubmatch(block)[1]
		return "<pre class=\"mermaid\">" + escapeHTML(strings.TrimRight(src, "\n")) + "</pre>"
	})
}

// escapeHTML escapes the three characters that would otherwise end the <pre>
// early or be read as markup. Mermaid parses the element's text content, so it
// sees the original characters back.
func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Link rewrites applied to chapters and notes on their way to the catalog.
var (
	topicLink    = regexp.MustCompile(`\]\(\.\./([a-z_]+)/\)`)
	readmeLink   = regexp.MustCompile(`\]\(\.\./README\.md\)`)
	mermaidFence = regexp.MustCompile("(?s)```mermaid\n(.*?)```")
)

// checkCoverage fails when info.toml has a topic the tiers table doesn't place,
// or the tiers table names a topic with no exercises.
func checkCoverage(byTopic map[string][]exercises.Exercise) error {
	placed := map[string]bool{}
	for _, ti := range tiers {
		for _, t := range ti.topics {
			placed[t] = true
			if len(byTopic[t]) == 0 {
				return fmt.Errorf("tier topic %q has no exercises in info.toml", t)
			}
		}
	}
	var missing []string
	for t := range byTopic {
		if !placed[t] {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("topics missing from the tiers table: %s", strings.Join(missing, ", "))
	}
	return nil
}

// writeCatalog emits the /catalog data: src/data/catalog.ts (typed entries the
// page renders at build time) and src/data/lesson-details/<name>.md (source and
// hint, fetched into the modal on demand so spoilers never load with the table).
func writeCatalog(byTopic map[string][]exercises.Exercise) error {
	if err := os.RemoveAll(detailsDir); err != nil {
		return err
	}
	if err := os.MkdirAll(detailsDir, 0o755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("// AUTO-GENERATED by `go run ./web/gen` from info.toml — do not hand-edit.\n\n")
	b.WriteString("export type CatalogEntry = {\n")
	b.WriteString("  topic: string;\n  slug: string;\n")
	b.WriteString("  description: string;\n  path: string;\n};\n\n")

	b.WriteString("export const TOPICS: Record<string, { label: string; tier: string; color: string; learn: string }> = {\n")
	for i, ti := range tiers {
		color := tierColors[i%len(tierColors)]
		for _, topic := range ti.topics {
			fmt.Fprintf(&b, "  %s: { label: %s, tier: %s, color: %s, learn: %s },\n",
				js(topic), js(pretty(topic)), js(ti.name), js(color), js(learnText(topic)))
		}
	}
	b.WriteString("};\n\n")

	b.WriteString("export const CATALOG: CatalogEntry[] = [\n")
	for _, ti := range tiers {
		for _, topic := range ti.topics {
			for _, e := range byTopic[topic] {
				fmt.Fprintf(&b, "  { topic: %s, slug: %s, description: %s, path: %s },\n",
					js(topic), js(e.Name), js(e.Description()), js(e.Path))
				if err := writeDetail(e); err != nil {
					return err
				}
			}
		}
	}
	b.WriteString("];\n")
	return os.WriteFile(filepath.Join(dataDir, "catalog.ts"), []byte(b.String()), 0o644)
}

// writeDetail renders one exercise's modal content as markdown: the
// broken-on-purpose source, then the hint and the teaching walk-through, each
// behind its own <details>. The worked solution is deliberately NOT embedded —
// the modal links to it on GitHub so the site never spoils the answer inline.
func writeDetail(e exercises.Exercise) error {
	files, err := filepath.Glob(filepath.Join(filepath.Dir(e.Path), "*.go"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %s\n---\n\n", e.Name)

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "```go title=%q\n%s\n```\n\n", filepath.Base(f), strings.TrimRight(string(src), "\n"))
	}

	if h := strings.TrimSpace(e.Hint); h != "" {
		fmt.Fprintf(&b, "<details>\n<summary>Show hint (spoiler)</summary>\n\n```text\n%s\n```\n\n</details>\n\n", h)
	}

	if n := notesBody(e); n != "" {
		fmt.Fprintf(&b, "<details>\n<summary>Show walk-through (spoiler)</summary>\n\n%s\n\n</details>\n\n", n)
	}

	solDir := "solutions/" + strings.TrimPrefix(filepath.ToSlash(filepath.Dir(e.Path)), "exercises/")
	if _, err := os.Stat(solDir); err != nil {
		return fmt.Errorf("%s: no solution dir %s", e.Name, solDir)
	}
	fmt.Fprintf(&b, "**Worked solution:** [view it on GitHub ↗](%s/%s/main.go) — try the hint first.\n", repoBlob, solDir)

	return os.WriteFile(filepath.Join(detailsDir, e.Name+".md"), []byte(b.String()), 0o644)
}

// notesBody returns the exercise's notes.md with its leading heading stripped
// — the modal already names the exercise — or "" when it has no walk-through.
func notesBody(e exercises.Exercise) string {
	md := strings.TrimSpace(e.Notes())
	if md == "" {
		return ""
	}
	lines := strings.Split(md, "\n")
	if strings.HasPrefix(lines[0], "#") {
		lines = lines[1:]
	}
	out := strings.TrimSpace(strings.Join(lines, "\n"))
	// The note's chapter backlink is a relative path on GitHub and in the TUI;
	// on the site the chapter is a catalog deep link.
	out = readmeLink.ReplaceAllString(out, "](/catalog/?chapter="+topicOf(e.Path)+")")
	return exercises.StripFences(out, "ascii")
}

// js renders s as a JavaScript string literal.
func js(s string) string {
	out, _ := json.Marshal(s)
	return string(out)
}

var mdLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

// learnText returns the topic's opening prose as popover-ready plain text:
// headings and link lists dropped, inline links unwrapped, paragraphs kept as
// blank-line breaks. Only the intro — the rest of the chapter opens in the
// modal, and a whole chapter would not fit a popover.
func learnText(topic string) string {
	body := intro(readme(topic))
	if body == "" {
		return ""
	}
	var paras []string
	for para := range strings.SplitSeq(body, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" || strings.HasPrefix(para, "#") || strings.HasPrefix(para, "-") {
			continue
		}
		para = strings.ReplaceAll(para, "\n", " ")
		paras = append(paras, mdLink.ReplaceAllString(para, "$1"))
	}
	return strings.Join(paras, "\n\n")
}

// intro returns a chapter's opening prose: everything before its first section
// heading. It has to stand on its own — the catalog popover shows only this.
func intro(body string) string {
	if i := strings.Index(body, "\n## "); i >= 0 {
		body = body[:i]
	}
	return strings.TrimSpace(body)
}

// readme returns the topic README with its leading H1 stripped (the popover
// has a title already), or "" if there is no README.
func readme(topic string) string {
	data, err := os.ReadFile(filepath.Join(exerciseR, topic, "README.md"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		lines = lines[1:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func topicOf(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return path
}

func pretty(topic string) string {
	if p, ok := prettyNames[topic]; ok {
		return p
	}
	words := strings.Split(topic, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
