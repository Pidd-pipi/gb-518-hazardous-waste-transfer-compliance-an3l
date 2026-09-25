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

func TestReceiptAndEscalationGraphEdges(t *testing.T) {
	if CanTransition(TransferManifestTransitions, "in_transit", "received") {
		t.Fatal("签收 must go through the receive endpoint with a measured weight")
	}
	if !CanTransition(TransferManifestTransitions, "in_transit", "rejected") {
		t.Fatal("in_transit manifests must still rejectable without a receipt")
	}
	if !CanTransition(ComplianceCheckTransitions, "pending", "escalated") {
		t.Fatal("a pending check on a rejected manifest must be able to escalate")
	}
	if !CanTransition(ComplianceCheckTransitions, "pending", "fail") {
		t.Fatal("a pending check must be able to record a fail outcome")
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
		{name: "receipt requires the dedicated receive operation", graph: TransferManifestTransitions, from: "in_transit", to: "received"},
		{name: "received manifest is final", graph: TransferManifestTransitions, from: "received", to: "rejected"},
		{name: "rejected manifest cannot be signed", graph: TransferManifestTransitions, from: "rejected", to: "received"},
		{name: "passed check is final", graph: ComplianceCheckTransitions, from: "pass", to: "pending"},
		{name: "passed check cannot be escalated", graph: ComplianceCheckTransitions, from: "pass", to: "escalated"},
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
