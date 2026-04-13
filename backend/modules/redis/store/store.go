package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pagination"
)

type Store struct {
	pb.UnimplementedPersistenceServiceServer
	client *redis.Client
}

func New(client *redis.Client) *Store {
	return &Store{client: client}
}

// --- Shared helpers ---

func buildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func (s *Store) marshalAndSet(ctx context.Context, key string, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return s.client.Set(ctx, key, data, 0).Err()
}

func (s *Store) getAndUnmarshal(ctx context.Context, key string, target any) (bool, error) {
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) paginateKeys(ctx context.Context, setKey string, pr *pb.PageRequest) ([]string, *pb.PageInfo, error) {
	page, perPage := pagination.Parse(pr)
	keys, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(keys)

	total := int32(len(keys))
	offset := (page - 1) * perPage
	end := offset + perPage
	if offset > total {
		offset = total
	}
	if end > total {
		end = total
	}
	return keys[offset:end], pagination.Info(page, perPage, total), nil
}

// --- Flags ---

func (s *Store) CreateFlag(ctx context.Context, req *pb.CreateFlagRequest) (*pb.CreateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()
	now := time.Now().Unix()
	f.CreatedAt = now
	f.UpdatedAt = now

	data, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("create flag marshal: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, buildKey(tid, "flags", f.GetKey()), data, 0)
	pipe.SAdd(ctx, buildKey(tid, "flag_keys"), f.GetKey())
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}
	return &pb.CreateFlagResponse{Flag: f}, nil
}

func (s *Store) GetFlag(ctx context.Context, req *pb.GetFlagRequest) (*pb.GetFlagResponse, error) {
	var f pb.FlagDefinition
	found, err := s.getAndUnmarshal(ctx, buildKey(req.GetTenant().GetTenantId(), "flags", req.GetFlagKey()), &f)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	if !found {
		return &pb.GetFlagResponse{}, nil
	}
	return &pb.GetFlagResponse{Flag: &f}, nil
}

func (s *Store) ListFlags(ctx context.Context, req *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	tid := req.GetTenant().GetTenantId()
	pageKeys, pi, err := s.paginateKeys(ctx, buildKey(tid, "flag_keys"), req.GetPagination())
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}

	flags := make([]*pb.FlagDefinition, 0, len(pageKeys))
	for _, k := range pageKeys {
		var f pb.FlagDefinition
		if found, err := s.getAndUnmarshal(ctx, buildKey(tid, "flags", k), &f); err == nil && found {
			flags = append(flags, &f)
		}
	}

	return &pb.ListFlagsResponse{
		Flags:      flags,
		Pagination: pi,
	}, nil
}

func (s *Store) UpdateFlag(ctx context.Context, req *pb.UpdateFlagRequest) (*pb.UpdateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()
	key := buildKey(tid, "flags", f.GetKey())

	var old pb.FlagDefinition
	found, err := s.getAndUnmarshal(ctx, key, &old)
	if err != nil {
		return nil, fmt.Errorf("update flag get: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("update flag get: %w", redis.Nil)
	}

	f.CreatedAt = old.CreatedAt
	f.UpdatedAt = time.Now().Unix()

	if err := s.marshalAndSet(ctx, key, f); err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}
	return &pb.UpdateFlagResponse{Flag: f}, nil
}

func (s *Store) DeleteFlag(ctx context.Context, req *pb.DeleteFlagRequest) (*pb.DeleteFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	pipe := s.client.Pipeline()
	pipe.Del(ctx, buildKey(tid, "flags", req.GetFlagKey()))
	pipe.SRem(ctx, buildKey(tid, "flag_keys"), req.GetFlagKey())
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete flag: %w", err)
	}
	return &pb.DeleteFlagResponse{}, nil
}

// --- Assignments ---

type assignmentData struct {
	SubjectID  string `json:"subject_id"`
	FlagKey    string `json:"flag_key"`
	Enabled    bool   `json:"enabled"`
	Variant    string `json:"variant"`
	AssignedAt int64  `json:"assigned_at"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
}

func (s *Store) SaveAssignment(ctx context.Context, req *pb.SaveAssignmentRequest) (*pb.SaveAssignmentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	a := req.GetAssignment()

	assignedAt := time.Now().Unix()
	if a.GetAssignedAt() > 0 {
		assignedAt = a.GetAssignedAt()
	}

	ad := assignmentData{
		SubjectID:  a.GetSubjectId(),
		FlagKey:    a.GetFlagKey(),
		Enabled:    a.GetEnabled(),
		Variant:    a.GetVariant(),
		AssignedAt: assignedAt,
		ExpiresAt:  a.GetExpiresAt(),
	}

	data, err := json.Marshal(ad)
	if err != nil {
		return nil, fmt.Errorf("save assignment marshal: %w", err)
	}

	rkey := buildKey(tid, "assignments", a.GetSubjectId(), a.GetFlagKey())

	var ttl time.Duration
	if a.GetExpiresAt() > 0 {
		ttl = time.Until(time.Unix(a.GetExpiresAt(), 0))
		if ttl <= 0 {
			ttl = 1 * time.Millisecond
		}
	}

	if err := s.client.Set(ctx, rkey, data, ttl).Err(); err != nil {
		return nil, fmt.Errorf("save assignment: %w", err)
	}
	return &pb.SaveAssignmentResponse{}, nil
}

func (s *Store) GetAssignment(ctx context.Context, req *pb.GetAssignmentRequest) (*pb.GetAssignmentResponse, error) {
	rkey := buildKey(req.GetTenant().GetTenantId(), "assignments", req.GetSubjectId(), req.GetFlagKey())

	var ad assignmentData
	found, err := s.getAndUnmarshal(ctx, rkey, &ad)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if !found {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}

	if ad.ExpiresAt > 0 && time.Unix(ad.ExpiresAt, 0).Before(time.Now()) {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}

	a := &pb.Assignment{
		SubjectId:  ad.SubjectID,
		FlagKey:    ad.FlagKey,
		Enabled:    ad.Enabled,
		Variant:    ad.Variant,
		AssignedAt: ad.AssignedAt,
		ExpiresAt:  ad.ExpiresAt,
	}
	return &pb.GetAssignmentResponse{Found: true, Assignment: a}, nil
}

// --- Experiments ---

func (s *Store) CreateExperiment(ctx context.Context, req *pb.CreateExperimentRequest) (*pb.CreateExperimentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	e := req.GetExperiment()
	now := time.Now().Unix()
	e.CreatedAt = now
	e.UpdatedAt = now

	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("create experiment marshal: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, buildKey(tid, "experiments", e.GetKey()), data, 0)
	pipe.SAdd(ctx, buildKey(tid, "experiment_keys"), e.GetKey())
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	return &pb.CreateExperimentResponse{Experiment: e}, nil
}

func (s *Store) GetExperiment(ctx context.Context, req *pb.GetExperimentRequest) (*pb.GetExperimentResponse, error) {
	var e pb.Experiment
	found, err := s.getAndUnmarshal(ctx, buildKey(req.GetTenant().GetTenantId(), "experiments", req.GetExperimentKey()), &e)
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	if !found {
		return &pb.GetExperimentResponse{}, nil
	}
	return &pb.GetExperimentResponse{Experiment: &e}, nil
}

func (s *Store) ListExperiments(ctx context.Context, req *pb.ListExperimentsRequest) (*pb.ListExperimentsResponse, error) {
	tid := req.GetTenant().GetTenantId()
	pageKeys, pi, err := s.paginateKeys(ctx, buildKey(tid, "experiment_keys"), req.GetPagination())
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}

	experiments := make([]*pb.Experiment, 0, len(pageKeys))
	for _, k := range pageKeys {
		var e pb.Experiment
		if found, err := s.getAndUnmarshal(ctx, buildKey(tid, "experiments", k), &e); err == nil && found {
			experiments = append(experiments, &e)
		}
	}

	return &pb.ListExperimentsResponse{
		Experiments: experiments,
		Pagination:  pi,
	}, nil
}

func (s *Store) UpdateExperiment(ctx context.Context, req *pb.UpdateExperimentRequest) (*pb.UpdateExperimentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	e := req.GetExperiment()
	key := buildKey(tid, "experiments", e.GetKey())

	var old pb.Experiment
	found, err := s.getAndUnmarshal(ctx, key, &old)
	if err != nil {
		return nil, fmt.Errorf("update experiment get: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("update experiment get: %w", redis.Nil)
	}

	e.CreatedAt = old.CreatedAt
	e.UpdatedAt = time.Now().Unix()

	if err := s.marshalAndSet(ctx, key, e); err != nil {
		return nil, fmt.Errorf("update experiment: %w", err)
	}
	return &pb.UpdateExperimentResponse{Experiment: e}, nil
}

// --- API Keys ---

type apiKeyData struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	KeyHash   string `json:"key_hash"`
	KeyPrefix string `json:"key_prefix"`
	Scope     string `json:"scope"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt int64  `json:"created_at"`
	RevokedAt int64  `json:"revoked_at,omitempty"`
}

func apiKeyFromData(d *apiKeyData) *pb.APIKey {
	return &pb.APIKey{
		Id:        d.ID,
		KeyHash:   d.KeyHash,
		KeyPrefix: d.KeyPrefix,
		Scope:     d.Scope,
		Name:      d.Name,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
		RevokedAt: d.RevokedAt,
	}
}

func (s *Store) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	tid := req.GetTenant().GetTenantId()
	k := req.GetApiKey()
	now := time.Now().Unix()

	ad := apiKeyData{
		ID:        k.GetId(),
		TenantID:  tid,
		KeyHash:   k.GetKeyHash(),
		KeyPrefix: k.GetKeyPrefix(),
		Scope:     k.GetScope(),
		Name:      k.GetName(),
		CreatedBy: k.GetCreatedBy(),
		CreatedAt: now,
	}

	data, err := json.Marshal(ad)
	if err != nil {
		return nil, fmt.Errorf("create api key marshal: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, buildKey(tid, "apikeys", k.GetId()), data, 0)
	pipe.SAdd(ctx, buildKey(tid, "apikey_ids"), k.GetId())
	pipe.Set(ctx, buildKey("apikey_by_hash", k.GetKeyHash()), buildKey(tid, k.GetId()), 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &pb.CreateAPIKeyResponse{ApiKey: apiKeyFromData(&ad)}, nil
}

func (s *Store) GetAPIKeyByHash(ctx context.Context, req *pb.GetAPIKeyByHashRequest) (*pb.GetAPIKeyByHashResponse, error) {
	ref, err := s.client.Get(ctx, buildKey("apikey_by_hash", req.GetKeyHash())).Result()
	if err == redis.Nil {
		return &pb.GetAPIKeyByHashResponse{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash lookup: %w", err)
	}

	tid, id, ok := strings.Cut(ref, ":")
	if !ok {
		return &pb.GetAPIKeyByHashResponse{Found: false}, nil
	}

	var ad apiKeyData
	found, err := s.getAndUnmarshal(ctx, buildKey(tid, "apikeys", id), &ad)
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	if !found {
		return &pb.GetAPIKeyByHashResponse{Found: false}, nil
	}

	return &pb.GetAPIKeyByHashResponse{
		Found:    true,
		ApiKey:   apiKeyFromData(&ad),
		TenantId: ad.TenantID,
	}, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	tid := req.GetTenant().GetTenantId()
	pageIDs, pi, err := s.paginateKeys(ctx, buildKey(tid, "apikey_ids"), req.GetPagination())
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	keys := make([]*pb.APIKey, 0, len(pageIDs))
	for _, id := range pageIDs {
		var ad apiKeyData
		if found, err := s.getAndUnmarshal(ctx, buildKey(tid, "apikeys", id), &ad); err == nil && found {
			keys = append(keys, apiKeyFromData(&ad))
		}
	}

	return &pb.ListAPIKeysResponse{
		ApiKeys:    keys,
		Pagination: pi,
	}, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	tid := req.GetTenant().GetTenantId()
	rkey := buildKey(tid, "apikeys", req.GetKeyId())

	var ad apiKeyData
	found, err := s.getAndUnmarshal(ctx, rkey, &ad)
	if err != nil {
		return nil, fmt.Errorf("revoke api key get: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("revoke api key get: %w", redis.Nil)
	}

	ad.RevokedAt = time.Now().Unix()
	if err := s.marshalAndSet(ctx, rkey, ad); err != nil {
		return nil, fmt.Errorf("revoke api key: %w", err)
	}
	return &pb.RevokeAPIKeyResponse{}, nil
}
