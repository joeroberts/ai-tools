package implementation

import "testing"

func TestParseAssessment(t *testing.T) {
	assessment, err := parseAssessment([]byte(`{"findings":[{"id":"R-1","severity":"blocking","summary":"outside approved paths"}]}`))
	if err != nil || len(assessment.Findings) != 1 {
		t.Fatalf("parseAssessment() = %#v, %v", assessment, err)
	}
	if _, err := parseAssessment([]byte(`{"findings":[{"id":"","severity":"bad","summary":""}]}`)); err == nil {
		t.Fatal("invalid assessment was accepted")
	}
}
