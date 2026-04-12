package grpcclient

import (
	"context"
	"net"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	fmtUnexpectedErr = "unexpected error: %v"
	tenantDefault    = "_default"
	flagDarkMode     = "dark-mode"
	subjectUser1     = "user-1"
)

type mockPersistenceServer struct {
	pb.UnimplementedPersistenceServiceServer
}

func (m *mockPersistenceServer) GetFlag(_ context.Context, req *pb.GetFlagRequest) (*pb.GetFlagResponse, error) {
	if req.FlagKey == flagDarkMode {
		return &pb.GetFlagResponse{
			Flag: &pb.FlagDefinition{
				Key:       flagDarkMode,
				Type:      "boolean",
				Semantics: "deterministic",
				Enabled:   true,
				DefaultResult: &pb.EvalResult{
					Enabled: true,
					Variant: "on",
				},
				Rules: []*pb.Rule{
					{
						Conditions: []*pb.Condition{
							{
								Attribute: "environment",
								Operator:  "equals",
								ValueJson: `"production"`,
							},
						},
						RolloutPercentage: 100,
						Variant:           "on",
					},
				},
			},
		}, nil
	}
	return &pb.GetFlagResponse{}, nil
}

func (m *mockPersistenceServer) ListFlags(_ context.Context, _ *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	return &pb.ListFlagsResponse{
		Flags: []*pb.FlagDefinition{
			{Key: flagDarkMode},
			{Key: "other-flag"},
		},
	}, nil
}

func (m *mockPersistenceServer) GetAssignment(_ context.Context, req *pb.GetAssignmentRequest) (*pb.GetAssignmentResponse, error) {
	if req.SubjectId == subjectUser1 && req.FlagKey == flagDarkMode {
		return &pb.GetAssignmentResponse{
			Found: true,
			Assignment: &pb.Assignment{
				SubjectId: subjectUser1,
				FlagKey:   flagDarkMode,
				Enabled:   true,
				Variant:   "on",
			},
		}, nil
	}
	return &pb.GetAssignmentResponse{Found: false}, nil
}

func (m *mockPersistenceServer) SaveAssignment(_ context.Context, _ *pb.SaveAssignmentRequest) (*pb.SaveAssignmentResponse, error) {
	return &pb.SaveAssignmentResponse{}, nil
}

func startMockServer(t *testing.T) *grpc.ClientConn {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterPersistenceServiceServer(srv, &mockPersistenceServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestPersistenceClient_GetFlag(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	flag, err := client.GetFlag(tenantDefault, flagDarkMode)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if flag == nil {
		t.Fatal("expected flag, got nil")
	}
	if flag.Key != flagDarkMode {
		t.Errorf("expected key %s, got %s", flagDarkMode, flag.Key)
	}
	if !flag.Enabled {
		t.Error("expected enabled = true")
	}
	if len(flag.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(flag.Rules))
	}
	if flag.Rules[0].Conditions[0].Value != "production" {
		t.Errorf("expected condition value 'production', got %v", flag.Rules[0].Conditions[0].Value)
	}
}

func TestPersistenceClient_GetFlag_NotFound(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	flag, err := client.GetFlag(tenantDefault, "nonexistent")
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if flag != nil {
		t.Error("expected nil for nonexistent flag")
	}
}

func TestPersistenceClient_ListFlagKeys(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	keys, err := client.ListFlagKeys(tenantDefault)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestPersistenceClient_GetAssignment(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	assignment, found, err := client.GetAssignment(tenantDefault, subjectUser1, flagDarkMode)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if !found {
		t.Fatal("expected assignment to be found")
	}
	if !assignment.Enabled {
		t.Error("expected enabled = true")
	}
}

func TestPersistenceClient_GetAssignment_NotFound(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	_, found, err := client.GetAssignment(tenantDefault, "user-999", flagDarkMode)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if found {
		t.Error("expected assignment to not be found")
	}
}

func TestPersistenceClient_SaveAssignment(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	err := client.SaveAssignment(tenantDefault, &pb.Assignment{
		SubjectId: subjectUser1,
		FlagKey:   flagDarkMode,
		Enabled:   true,
		Variant:   "on",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
}

func TestPersistenceClient_Raw(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)

	if client.Raw() == nil {
		t.Error("expected Raw() to return non-nil client")
	}
}

func TestDial_Insecure(t *testing.T) {
	conn, err := Dial("127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	_ = conn.Close()
}
