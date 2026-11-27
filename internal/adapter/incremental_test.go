package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func transcriptLine(tokens int) string {
	return fmt.Sprintf(`{"type":"assistant","sessionId":"s-inc","cwd":"/p/inc","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":%d,"output_tokens":5}}}`+"\n", tokens)
}

func statFor(t *testing.T, path string, stamp time.Time) os.FileInfo {
	t.Helper()
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// The summary is a left fold, so continuing from the cached offset must be
// indistinguishable from reparsing the whole file — that equivalence is
// what lets a 200MB transcript cost a few KB per poll instead of a reread.
func TestIncrementalAbsorbMatchesFullReparse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s1.jsonl")
	base := time.Now().Add(-time.Minute).Truncate(time.Second)

	if err := os.WriteFile(path, []byte(transcriptLine(10)+transcriptLine(20)), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClaudeSource(dir, nil).(*claudeSource)
	first, ok := c.summarize(path, statFor(t, path, base))
	if !ok || first.totals.InputTokens != 30 {
		t.Fatalf("first pass: ok=%v input=%d, want 30", ok, first.totals.InputTokens)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(transcriptLine(30) + transcriptLine(40)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	info := statFor(t, path, base.Add(5*time.Second))

	incremental, _ := c.summarize(path, info)
	full, _ := NewClaudeSource(dir, nil).(*claudeSource).summarize(path, info)
	if incremental != full {
		t.Fatalf("incremental summary diverged from full reparse:\n inc: %+v\nfull: %+v", incremental, full)
	}
	if got := c.cache[path].offset; got != info.Size() {
		t.Fatalf("cache offset = %d, want file size %d", got, info.Size())
	}
}

// A file that shrank was rewritten, not appended: the fold starts over.
func TestIncrementalRestartsWhenFileShrinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s2.jsonl")
	base := time.Now().Add(-time.Minute).Truncate(time.Second)

	big := transcriptLine(100) + transcriptLine(200) + transcriptLine(300)
	if err := os.WriteFile(path, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClaudeSource(dir, nil).(*claudeSource)
	c.summarize(path, statFor(t, path, base))

	if err := os.WriteFile(path, []byte(transcriptLine(7)), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, _ := c.summarize(path, statFor(t, path, base.Add(5*time.Second)))
	if sum.totals.InputTokens != 7 {
		t.Fatalf("after rewrite input=%d, want 7 (stale fold leaked through)", sum.totals.InputTokens)
	}
}

// A torn tail — a write caught mid-flight, not yet valid JSON — must not
// be counted as consumed, so the next poll absorbs the finished line
// exactly once.
func TestIncrementalLeavesTornTailForNextPoll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s3.jsonl")
	base := time.Now().Add(-time.Minute).Truncate(time.Second)

	whole := transcriptLine(10)
	torn := `{"type":"assistant","sessionId":"s-inc","cwd":"/p/inc","message":{"model":"claude-sonnet-4-5","usage":{"input_tok`
	if err := os.WriteFile(path, []byte(whole+torn), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClaudeSource(dir, nil).(*claudeSource)
	sum, _ := c.summarize(path, statFor(t, path, base))
	if sum.totals.InputTokens != 10 {
		t.Fatalf("torn tail leaked into totals: input=%d, want 10", sum.totals.InputTokens)
	}
	if got, want := c.cache[path].offset, int64(len(whole)); got != want {
		t.Fatalf("offset = %d, want %d (end of last complete line)", got, want)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`ens":90,"output_tokens":5}}}` + "\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	sum, _ = c.summarize(path, statFor(t, path, base.Add(5*time.Second)))
	if sum.totals.InputTokens != 100 {
		t.Fatalf("finished line absorbed wrong: input=%d, want 100", sum.totals.InputTokens)
	}
}
