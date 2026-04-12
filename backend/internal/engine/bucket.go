package engine

import (
	"fmt"

	"github.com/twmb/murmur3"
)

// Bucket computes a deterministic bucket in [0, 99] for a given tenant, flag, and subject.
// Hash input: "tenantId:flagKey:subjectId" → MurmurHash3 (32-bit) → mod 100.
func Bucket(tenantID, flagKey, subjectID string) int {
	input := fmt.Sprintf("%s:%s:%s", tenantID, flagKey, subjectID)
	h := murmur3.Sum32([]byte(input))
	return int(h % 100)
}

// InRollout returns true if the subject's bucket falls within the rollout percentage [0, 100].
func InRollout(tenantID, flagKey, subjectID string, rolloutPercentage int) bool {
	if rolloutPercentage <= 0 {
		return false
	}
	if rolloutPercentage >= 100 {
		return true
	}
	return Bucket(tenantID, flagKey, subjectID) < rolloutPercentage
}
