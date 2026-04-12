package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/engine"
	"google.golang.org/grpc"
)

const defaultTimeout = 5 * time.Second

// PersistenceClient wraps the generated PersistenceServiceClient and implements
// engine.FlagStore for use by bacon-core.
type PersistenceClient struct {
	client pb.PersistenceServiceClient
	conn   *grpc.ClientConn
}

// NewPersistenceClient creates a client connected to the given gRPC connection.
func NewPersistenceClient(conn *grpc.ClientConn) *PersistenceClient {
	return &PersistenceClient{
		client: pb.NewPersistenceServiceClient(conn),
		conn:   conn,
	}
}

// GetFlag implements engine.FlagStore.
func (c *PersistenceClient) GetFlag(tenantID, flagKey string) (*engine.FlagDefinition, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := c.client.GetFlag(ctx, &pb.GetFlagRequest{
		Tenant:  &pb.TenantScope{TenantId: tenantID},
		FlagKey: flagKey,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc GetFlag: %w", err)
	}
	if resp.Flag == nil {
		return nil, nil
	}
	return protoToFlag(resp.Flag), nil
}

// ListFlagKeys implements engine.FlagStore.
func (c *PersistenceClient) ListFlagKeys(tenantID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := c.client.ListFlags(ctx, &pb.ListFlagsRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Pagination: &pb.PageRequest{Page: 1, PerPage: 10000},
	})
	if err != nil {
		return nil, fmt.Errorf("grpc ListFlags: %w", err)
	}

	keys := make([]string, len(resp.Flags))
	for i, f := range resp.Flags {
		keys[i] = f.Key
	}
	return keys, nil
}

// getAssignmentProto retrieves a persisted assignment as a proto message.
func (c *PersistenceClient) getAssignmentProto(tenantID, subjectID, flagKey string) (*pb.Assignment, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := c.client.GetAssignment(ctx, &pb.GetAssignmentRequest{
		Tenant:    &pb.TenantScope{TenantId: tenantID},
		SubjectId: subjectID,
		FlagKey:   flagKey,
	})
	if err != nil {
		return nil, false, fmt.Errorf("grpc GetAssignment: %w", err)
	}
	return resp.Assignment, resp.Found, nil
}

// saveAssignmentProto persists an assignment given as a proto message.
func (c *PersistenceClient) saveAssignmentProto(tenantID string, assignment *pb.Assignment) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err := c.client.SaveAssignment(ctx, &pb.SaveAssignmentRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Assignment: assignment,
	})
	if err != nil {
		return fmt.Errorf("grpc SaveAssignment: %w", err)
	}
	return nil
}

// Close closes the underlying gRPC connection.
func (c *PersistenceClient) Close() error {
	return c.conn.Close()
}

// Raw returns the underlying generated client for direct access to RPCs
// not exposed through engine.FlagStore.
func (c *PersistenceClient) Raw() pb.PersistenceServiceClient {
	return c.client
}

func protoToFlag(pf *pb.FlagDefinition) *engine.FlagDefinition {
	f := &engine.FlagDefinition{
		Key:         pf.Key,
		Type:        engine.FlagType(pf.Type),
		Semantics:   engine.FlagSemantics(pf.Semantics),
		Enabled:     pf.Enabled,
		Description: pf.Description,
	}

	if pf.DefaultResult != nil {
		f.DefaultResult = engine.EvalResult{
			Enabled: pf.DefaultResult.Enabled,
			Variant: pf.DefaultResult.Variant,
		}
	}

	for _, pr := range pf.Rules {
		r := engine.Rule{
			RolloutPercentage: int(pr.RolloutPercentage),
			Variant:           pr.Variant,
		}
		for _, pc := range pr.Conditions {
			c := engine.Condition{
				Attribute: pc.Attribute,
				Operator:  engine.Operator(pc.Operator),
			}
			if pc.ValueJson != "" {
				var val any
				if err := json.Unmarshal([]byte(pc.ValueJson), &val); err == nil {
					c.Value = val
				} else {
					c.Value = pc.ValueJson
				}
			}
			r.Conditions = append(r.Conditions, c)
		}
		f.Rules = append(f.Rules, r)
	}
	return f
}

var _ engine.FlagStore = (*PersistenceClient)(nil)
