package exercises

import "testing"

func TestValidatePath(t *testing.T) {
	valid := []string{
		"exercises",
		"exercises/variables/variables1/main.go",
		"exercises/a/b/c/main_test.go",
	}
	for _, p := range valid {
		if err := validatePath(p); err != nil {
			t.Errorf("validatePath(%q) = %v; want nil", p, err)
		}
	}

	invalid := []string{
		"",                           // empty
		"/etc/passwd",                // absolute
		"../secrets/main.go",         // traversal up
		"exercises/../../etc/passwd", // traversal escaping root
		"exercises/./variables1",     // non-canonical
		"notexercises/main.go",       // outside root
		"foo/exercises/main.go",      // root not a prefix
	}
	for _, p := range invalid {
		if err := validatePath(p); err == nil {
			t.Errorf("validatePath(%q) = nil; want error", p)
		}
	}
}
