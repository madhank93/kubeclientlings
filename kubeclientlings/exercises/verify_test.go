package exercises

import "testing"

func TestStatusString(t *testing.T) {
	cases := map[Status]string{
		StatusPending:  "Pending",
		StatusFailing:  "Failing",
		StatusLintFail: "LintFail",
		StatusDone:     "Done",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Status(%d).String() = %q, want %q", int(s), got, want)
		}
	}
}

// A non-existent / non-compiling exercise must report StatusFailing (Verify
// always runs the exercise rather than short-circuiting on the marker).
func TestVerifyMissingFails(t *testing.T) {
	e := Exercise{Name: "ghost", Path: "does/not/exist/main.go", Mode: "compile"}
	status, _ := e.Verify()
	if status != StatusFailing {
		t.Errorf("want StatusFailing for missing file, got %v", status)
	}
}
