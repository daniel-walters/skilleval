package eval

import "testing"

func TestParseStringMatch(t *testing.T) {
	tests := []struct {
		in      string
		wantRaw string
		regex   bool
		pattern string
	}{
		{in: "literal", wantRaw: "literal", regex: false, pattern: "literal"},
		{in: "/foo/", wantRaw: "/foo/", regex: true, pattern: "foo"},
		{in: "/a/b/", wantRaw: "/a/b/", regex: true, pattern: "a/b"},
		{in: "/", wantRaw: "/", regex: false, pattern: "/"},
		{in: "//", wantRaw: "//", regex: true, pattern: ""},
		{in: "path/to/file", wantRaw: "path/to/file", regex: false, pattern: "path/to/file"},
		{in: "/unclosed", wantRaw: "/unclosed", regex: false, pattern: "/unclosed"},
	}
	for _, tt := range tests {
		got := parseStringMatch(tt.in)
		if got.raw != tt.wantRaw || got.regex != tt.regex || got.pattern != tt.pattern {
			t.Fatalf("parseStringMatch(%q) = raw=%q regex=%v pattern=%q, want raw=%q regex=%v pattern=%q",
				tt.in, got.raw, got.regex, got.pattern, tt.wantRaw, tt.regex, tt.pattern)
		}
	}
}

func TestStringMatchContainsAndEquals(t *testing.T) {
	lit := parseStringMatch("Foo")
	if err := compileStringMatch(&lit); err != nil {
		t.Fatal(err)
	}
	if !lit.MatchContains("func Foo()") || lit.MatchContains("bar") {
		t.Fatal("literal contains mismatch")
	}
	if !lit.MatchEquals("Foo") || lit.MatchEquals("func Foo()") {
		t.Fatal("literal equals mismatch")
	}

	re := parseStringMatch(`/Foo\d+/`)
	if err := compileStringMatch(&re); err != nil {
		t.Fatal(err)
	}
	if !re.MatchContains("id=Foo12!") || re.MatchContains("Foo") {
		t.Fatal("regex contains mismatch")
	}

	full := parseStringMatch(`/^Done\.$/`)
	if err := compileStringMatch(&full); err != nil {
		t.Fatal(err)
	}
	if !full.MatchEquals("Done.") || full.MatchEquals("Done. more") || full.MatchEquals("xDone.") {
		t.Fatal("regex equals full-match mismatch")
	}
}
