package server

import (
	"context"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/modules/postgres/store"
)

type Server struct {
	pb.UnimplementedPersistenceServiceServer
	store *store.Store
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

func (s *Server) GetFlag(ctx context.Context, req *pb.GetFlagRequest) (*pb.GetFlagResponse, error) {
	return s.store.GetFlag(ctx, req)
}

func (s *Server) ListFlags(ctx context.Context, req *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	return s.store.ListFlags(ctx, req)
}

func (s *Server) CreateFlag(ctx context.Context, req *pb.CreateFlagRequest) (*pb.CreateFlagResponse, error) {
	return s.store.CreateFlag(ctx, req)
}

func (s *Server) UpdateFlag(ctx context.Context, req *pb.UpdateFlagRequest) (*pb.UpdateFlagResponse, error) {
	return s.store.UpdateFlag(ctx, req)
}

func (s *Server) DeleteFlag(ctx context.Context, req *pb.DeleteFlagRequest) (*pb.DeleteFlagResponse, error) {
	return s.store.DeleteFlag(ctx, req)
}

func (s *Server) GetAssignment(ctx context.Context, req *pb.GetAssignmentRequest) (*pb.GetAssignmentResponse, error) {
	return s.store.GetAssignment(ctx, req)
}

func (s *Server) SaveAssignment(ctx context.Context, req *pb.SaveAssignmentRequest) (*pb.SaveAssignmentResponse, error) {
	return s.store.SaveAssignment(ctx, req)
}

func (s *Server) GetExperiment(ctx context.Context, req *pb.GetExperimentRequest) (*pb.GetExperimentResponse, error) {
	return s.store.GetExperiment(ctx, req)
}

func (s *Server) ListExperiments(ctx context.Context, req *pb.ListExperimentsRequest) (*pb.ListExperimentsResponse, error) {
	return s.store.ListExperiments(ctx, req)
}

func (s *Server) CreateExperiment(ctx context.Context, req *pb.CreateExperimentRequest) (*pb.CreateExperimentResponse, error) {
	return s.store.CreateExperiment(ctx, req)
}

func (s *Server) UpdateExperiment(ctx context.Context, req *pb.UpdateExperimentRequest) (*pb.UpdateExperimentResponse, error) {
	return s.store.UpdateExperiment(ctx, req)
}

func (s *Server) GetAPIKeyByHash(ctx context.Context, req *pb.GetAPIKeyByHashRequest) (*pb.GetAPIKeyByHashResponse, error) {
	return s.store.GetAPIKeyByHash(ctx, req)
}

func (s *Server) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	return s.store.ListAPIKeys(ctx, req)
}

func (s *Server) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	return s.store.CreateAPIKey(ctx, req)
}

func (s *Server) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	return s.store.RevokeAPIKey(ctx, req)
}
