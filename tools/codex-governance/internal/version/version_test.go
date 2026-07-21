package version

import "testing"

func TestParseAndNext(t *testing.T) {
	v, err := Parse("1.2.3-rc.1+build.4")
	if err != nil || v.String() != "1.2.3-rc.1+build.4" {
		t.Fatalf("Parse()=%v,%v", v, err)
	}
	n, err := v.Next("minor")
	if err != nil || n.String() != "1.3.0" {
		t.Fatalf("Next()=%v,%v", n, err)
	}
}
func TestParseRejectsInvalid(t *testing.T) {
	for _, value := range []string{"1.2", "01.2.3", "1.2.3-01", "1.2.3-"} {
		if _, err := Parse(value); err == nil {
			t.Fatalf("Parse(%q) succeeded", value)
		}
	}
}
