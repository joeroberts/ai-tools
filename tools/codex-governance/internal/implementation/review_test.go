package implementation

import "testing"

func TestReviewAndVerificationConverge(t *testing.T) {
	run := Run{State: StateReview}
	if err := ApplyReview(&run, Assessment{}); err != nil || run.State != StateVerification {
		t.Fatalf("review = %v, %s", err, run.State)
	}
	if err := ApplyVerification(&run, Assessment{}); err != nil || run.State != StateReadyToCommit {
		t.Fatalf("verification = %v, %s", err, run.State)
	}
}

func TestRemediationRequiresNamedActionableFinding(t *testing.T) {
	run := Run{State: StateRemediation}
	assessment := Assessment{Findings: []Finding{{ID: "R-1", Severity: "blocking", Summary: "out of scope"}}}
	if err := ApplyRemediation(&run, assessment, nil); err == nil {
		t.Fatal("unnamed remediation was accepted")
	}
	if err := ApplyRemediation(&run, assessment, []string{"R-2"}); err == nil {
		t.Fatal("unknown remediation finding was accepted")
	}
	if err := ApplyRemediation(&run, assessment, []string{"R-1"}); err != nil || run.State != StateReview {
		t.Fatalf("named remediation = %v, %s", err, run.State)
	}
}

func TestActionableFindingRequiresBoundedRemediation(t *testing.T) {
	run := Run{State: StateReview}
	finding := Assessment{Findings: []Finding{{ID: "R-1", Severity: "blocking", Summary: "out of scope"}}}
	if err := ApplyReview(&run, finding); err != nil || run.State != StateRemediation || run.ReviewCycles != 1 {
		t.Fatalf("first review = %v, %#v", err, run)
	}
	if err := run.Transition(StateReview); err != nil {
		t.Fatal(err)
	}
	run.ReviewCycles = 2
	if err := ApplyReview(&run, finding); err != nil || run.State != StateEscalated {
		t.Fatalf("bounded review = %v, %#v", err, run)
	}
}
