package constants

// Shared status values are mirrored in frontend/src/types/status.ts. Keeping
// the lists explicit makes state-machine drift visible during code review.

type ManifestState string

const (
	ManifestStateDraft     ManifestState = "draft"
	ManifestStateSubmitted ManifestState = "submitted"
	ManifestStateInTransit ManifestState = "in_transit"
	ManifestStateReceived  ManifestState = "received"
	ManifestStateRejected  ManifestState = "rejected"
)

var AllManifestState = []string{"draft", "submitted", "in_transit", "received", "rejected"}

type CheckState string

const (
	CheckStatePending   CheckState = "pending"
	CheckStatePass      CheckState = "pass"
	CheckStateFail      CheckState = "fail"
	CheckStateEscalated CheckState = "escalated"
)

var AllCheckState = []string{"pending", "pass", "fail", "escalated"}

var WasteGeneratorTransitions = map[string]map[string]bool{
	"active":     {"restricted": true, "suspended": true, "expired": true},
	"restricted": {"active": true, "suspended": true, "expired": true},
	"suspended":  {"active": true, "expired": true},
	"expired":    {},
}

var CarrierProfileTransitions = map[string]map[string]bool{
	"pending":    {"verified": true, "restricted": true},
	"verified":   {"restricted": true, "expired": true},
	"restricted": {"pending": true, "expired": true},
	"expired":    {},
}

var TransferManifestTransitions = map[string]map[string]bool{
	"draft":      {"submitted": true},
	"submitted":  {"in_transit": true, "rejected": true},
	"in_transit": {"received": true, "rejected": true},
	"received":   {},
	"rejected":   {},
}

var ComplianceCheckTransitions = map[string]map[string]bool{
	"pending":   {"pass": true, "fail": true},
	"pass":      {},
	"fail":      {"escalated": true},
	"escalated": {},
}

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
