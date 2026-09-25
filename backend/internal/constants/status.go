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
	"in_transit": {"rejected": true},
	// 签收 (in_transit -> received) is not a bare transition: it goes through
	// POST /manifests/:id/receive with the measured weight. Receipt over the
	// allowed tolerance lands in rejected instead. Both states are terminal.
	"received": {},
	"rejected": {},
}

var ComplianceCheckTransitions = map[string]map[string]bool{
	"pending":   {"pass": true, "fail": true, "escalated": true},
	"pass":      {},
	"fail":      {"escalated": true},
	"escalated": {},
}

// ManifestWeightTolerance caps how far the received weight may exceed the
// planned QuantityKg. Strictly above 5% turns 签收 into 驳回.
const ManifestWeightTolerance = 0.05

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
