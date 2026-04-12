package grpcclient

import (
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

// GetAssignment implements engine.AssignmentStore.
func (c *PersistenceClient) GetAssignment(tenantID, subjectID, flagKey string) (*engine.Assignment, bool, error) {
	pbAssign, found, err := c.getAssignmentProto(tenantID, subjectID, flagKey)
	if err != nil || !found {
		return nil, false, err
	}
	a := &engine.Assignment{
		SubjectID:  pbAssign.SubjectId,
		FlagKey:    pbAssign.FlagKey,
		Enabled:    pbAssign.Enabled,
		Variant:    pbAssign.Variant,
		AssignedAt: time.Unix(pbAssign.AssignedAt, 0),
	}
	if pbAssign.ExpiresAt > 0 {
		a.ExpiresAt = time.Unix(pbAssign.ExpiresAt, 0)
	}
	return a, true, nil
}

// SaveAssignment implements engine.AssignmentStore.
func (c *PersistenceClient) SaveAssignment(tenantID string, a *engine.Assignment) error {
	pbAssign := &pb.Assignment{
		SubjectId:  a.SubjectID,
		FlagKey:    a.FlagKey,
		Enabled:    a.Enabled,
		Variant:    a.Variant,
		AssignedAt: a.AssignedAt.Unix(),
	}
	if !a.ExpiresAt.IsZero() {
		pbAssign.ExpiresAt = a.ExpiresAt.Unix()
	}
	return c.saveAssignmentProto(tenantID, pbAssign)
}

var _ engine.AssignmentStore = (*PersistenceClient)(nil)
