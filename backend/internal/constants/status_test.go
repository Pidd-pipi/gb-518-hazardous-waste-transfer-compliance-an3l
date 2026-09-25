package constants

import "testing"

func TestWasteGeneratorTransitionGraph(t *testing.T) {
	if !CanTransition(WasteGeneratorTransitions, "active", "restricted") {
		t.Fatalf("expected active -> restricted transition to be allowed")
	}
	if CanTransition(WasteGeneratorTransitions, "active", "unknown") {
		t.Fatal("unknown status must never be accepted")
	}
}

func TestComplianceStateMachinesRejectBypassesAndReopen(t *testing.T) {
	tests := []struct {
		name  string
		graph map[string]map[string]bool
		from  string
		to    string
	}{
		{name: "manifest cannot skip submission", graph: TransferManifestTransitions, from: "draft", to: "in_transit"},
		{name: "received manifest is final", graph: TransferManifestTransitions, from: "received", to: "rejected"},
		{name: "passed check is final", graph: ComplianceCheckTransitions, from: "pass", to: "pending"},
		{name: "expired carrier cannot reactivate", graph: CarrierProfileTransitions, from: "expired", to: "verified"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if CanTransition(test.graph, test.from, test.to) {
				t.Fatalf("unexpected transition %s -> %s", test.from, test.to)
			}
		})
	}
}
