package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	defaultPerPage int32 = 20
	maxPerPage     int32 = 100
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type scanner interface {
	Scan(dest ...any) error
}

func paginate(pr *pb.PageRequest) (page, perPage int32) {
	page, perPage = 1, defaultPerPage
	if pr != nil {
		if pr.Page > 0 {
			page = pr.Page
		}
		if pr.PerPage > 0 {
			perPage = pr.PerPage
		}
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}

func pageInfo(page, perPage, total int32) *pb.PageInfo {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	return &pb.PageInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

func scanFlag(s scanner) (*pb.FlagDefinition, error) {
	var (
		f           pb.FlagDefinition
		desc        sql.NullString
		rulesJSON   []byte
		defaultJSON []byte
		createdBy   sql.NullString
		updatedBy   sql.NullString
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := s.Scan(
		&f.Key, &f.Type, &f.Semantics, &f.Enabled, &desc,
		&rulesJSON, &defaultJSON, &createdBy, &updatedBy,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.Description = desc.String
	f.CreatedBy = createdBy.String
	f.UpdatedBy = updatedBy.String
	f.CreatedAt = createdAt.Unix()
	f.UpdatedAt = updatedAt.Unix()
	_ = json.Unmarshal(rulesJSON, &f.Rules)
	var dr pb.EvalResult
	_ = json.Unmarshal(defaultJSON, &dr)
	f.DefaultResult = &dr
	return &f, nil
}

func scanExperiment(s scanner) (*pb.Experiment, error) {
	var (
		e         pb.Experiment
		varJSON   []byte
		allocJSON []byte
		createdAt time.Time
		updatedAt time.Time
	)
	err := s.Scan(
		&e.Key, &e.Name, &e.Status, &e.StickyAssignment,
		&varJSON, &allocJSON, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.CreatedAt = createdAt.Unix()
	e.UpdatedAt = updatedAt.Unix()
	_ = json.Unmarshal(varJSON, &e.Variants)
	_ = json.Unmarshal(allocJSON, &e.Allocation)
	return &e, nil
}

func scanAPIKey(s scanner) (*pb.APIKey, error) {
	var (
		k         pb.APIKey
		createdBy sql.NullString
		createdAt time.Time
		revokedAt sql.NullTime
	)
	err := s.Scan(
		&k.Id, &k.KeyHash, &k.KeyPrefix, &k.Scope, &k.Name,
		&createdBy, &createdAt, &revokedAt,
	)
	if err != nil {
		return nil, err
	}
	k.CreatedBy = createdBy.String
	k.CreatedAt = createdAt.Unix()
	if revokedAt.Valid {
		k.RevokedAt = revokedAt.Time.Unix()
	}
	return &k, nil
}

// marshalSlice ensures nil slices marshal as "[]" instead of "null".
func marshalSlice(v any) string {
	b, _ := json.Marshal(v)
	if len(b) == 0 || string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func marshalObj(v any) string {
	b, _ := json.Marshal(v)
	if len(b) == 0 || string(b) == "null" {
		return "{}"
	}
	return string(b)
}

// --- Flags ---

func (s *Store) GetFlag(ctx context.Context, req *pb.GetFlagRequest) (*pb.GetFlagResponse, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT key, type, semantics, enabled, description, rules, default_result, created_by, updated_by, created_at, updated_at
		 FROM flags WHERE tenant_id = $1 AND key = $2`,
		req.GetTenant().GetTenantId(), req.GetFlagKey(),
	)
	f, err := scanFlag(row)
	if err == sql.ErrNoRows {
		return &pb.GetFlagResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	return &pb.GetFlagResponse{Flag: f}, nil
}

func (s *Store) ListFlags(ctx context.Context, req *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	tid := req.GetTenant().GetTenantId()
	page, perPage := paginate(req.GetPagination())

	var total int32
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM flags WHERE tenant_id = $1`, tid,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("list flags count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, type, semantics, enabled, description, rules, default_result, created_by, updated_by, created_at, updated_at
		 FROM flags WHERE tenant_id = $1 ORDER BY key LIMIT $2 OFFSET $3`,
		tid, perPage, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list flags query: %w", err)
	}
	defer rows.Close()

	var flags []*pb.FlagDefinition
	for rows.Next() {
		f, err := scanFlag(rows)
		if err != nil {
			return nil, fmt.Errorf("list flags scan: %w", err)
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list flags rows: %w", err)
	}

	return &pb.ListFlagsResponse{
		Flags:      flags,
		Pagination: pageInfo(page, perPage, total),
	}, nil
}

func (s *Store) CreateFlag(ctx context.Context, req *pb.CreateFlagRequest) (*pb.CreateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()

	row := s.db.QueryRowContext(ctx,
		`INSERT INTO flags (tenant_id, key, type, semantics, enabled, description, rules, default_result, created_by, updated_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10)
		 RETURNING key, type, semantics, enabled, description, rules, default_result, created_by, updated_by, created_at, updated_at`,
		tid, f.GetKey(), f.GetType(), f.GetSemantics(), f.GetEnabled(),
		f.GetDescription(), marshalSlice(f.GetRules()), marshalObj(f.GetDefaultResult()),
		f.GetCreatedBy(), f.GetUpdatedBy(),
	)
	result, err := scanFlag(row)
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}
	return &pb.CreateFlagResponse{Flag: result}, nil
}

func (s *Store) UpdateFlag(ctx context.Context, req *pb.UpdateFlagRequest) (*pb.UpdateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()

	row := s.db.QueryRowContext(ctx,
		`UPDATE flags
		 SET type = $1, semantics = $2, enabled = $3, description = $4,
		     rules = $5::jsonb, default_result = $6::jsonb, updated_by = $7, updated_at = now()
		 WHERE tenant_id = $8 AND key = $9
		 RETURNING key, type, semantics, enabled, description, rules, default_result, created_by, updated_by, created_at, updated_at`,
		f.GetType(), f.GetSemantics(), f.GetEnabled(), f.GetDescription(),
		marshalSlice(f.GetRules()), marshalObj(f.GetDefaultResult()),
		f.GetUpdatedBy(), tid, f.GetKey(),
	)
	result, err := scanFlag(row)
	if err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}
	return &pb.UpdateFlagResponse{Flag: result}, nil
}

func (s *Store) DeleteFlag(ctx context.Context, req *pb.DeleteFlagRequest) (*pb.DeleteFlagResponse, error) {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM flags WHERE tenant_id = $1 AND key = $2`,
		req.GetTenant().GetTenantId(), req.GetFlagKey(),
	)
	if err != nil {
		return nil, fmt.Errorf("delete flag: %w", err)
	}
	return &pb.DeleteFlagResponse{}, nil
}

// --- Assignments ---

func (s *Store) GetAssignment(ctx context.Context, req *pb.GetAssignmentRequest) (*pb.GetAssignmentResponse, error) {
	tid := req.GetTenant().GetTenantId()

	var (
		a          pb.Assignment
		assignedAt time.Time
		expiresAt  sql.NullTime
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT subject_id, flag_key, enabled, variant, assigned_at, expires_at
		 FROM assignments
		 WHERE tenant_id = $1 AND subject_id = $2 AND flag_key = $3`,
		tid, req.GetSubjectId(), req.GetFlagKey(),
	).Scan(&a.SubjectId, &a.FlagKey, &a.Enabled, &a.Variant, &assignedAt, &expiresAt)

	if err == sql.ErrNoRows {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}

	a.AssignedAt = assignedAt.Unix()
	if expiresAt.Valid {
		a.ExpiresAt = expiresAt.Time.Unix()
	}
	return &pb.GetAssignmentResponse{Found: true, Assignment: &a}, nil
}

func (s *Store) SaveAssignment(ctx context.Context, req *pb.SaveAssignmentRequest) (*pb.SaveAssignmentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	a := req.GetAssignment()

	assignedAt := time.Now()
	if a.GetAssignedAt() > 0 {
		assignedAt = time.Unix(a.GetAssignedAt(), 0)
	}

	var expiresAt sql.NullTime
	if a.GetExpiresAt() > 0 {
		expiresAt = sql.NullTime{Time: time.Unix(a.GetExpiresAt(), 0), Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO assignments (tenant_id, subject_id, flag_key, enabled, variant, assigned_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, subject_id, flag_key)
		 DO UPDATE SET enabled = EXCLUDED.enabled, variant = EXCLUDED.variant,
		               assigned_at = EXCLUDED.assigned_at, expires_at = EXCLUDED.expires_at`,
		tid, a.GetSubjectId(), a.GetFlagKey(), a.GetEnabled(), a.GetVariant(), assignedAt, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save assignment: %w", err)
	}
	return &pb.SaveAssignmentResponse{}, nil
}

// --- Experiments ---

func (s *Store) GetExperiment(ctx context.Context, req *pb.GetExperimentRequest) (*pb.GetExperimentResponse, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT key, name, status, sticky_assignment, variants, allocation, created_at, updated_at
		 FROM experiments WHERE tenant_id = $1 AND key = $2`,
		req.GetTenant().GetTenantId(), req.GetExperimentKey(),
	)
	e, err := scanExperiment(row)
	if err == sql.ErrNoRows {
		return &pb.GetExperimentResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	return &pb.GetExperimentResponse{Experiment: e}, nil
}

func (s *Store) ListExperiments(ctx context.Context, req *pb.ListExperimentsRequest) (*pb.ListExperimentsResponse, error) {
	tid := req.GetTenant().GetTenantId()
	page, perPage := paginate(req.GetPagination())

	var total int32
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM experiments WHERE tenant_id = $1`, tid,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("list experiments count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, name, status, sticky_assignment, variants, allocation, created_at, updated_at
		 FROM experiments WHERE tenant_id = $1 ORDER BY key LIMIT $2 OFFSET $3`,
		tid, perPage, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list experiments query: %w", err)
	}
	defer rows.Close()

	var experiments []*pb.Experiment
	for rows.Next() {
		e, err := scanExperiment(rows)
		if err != nil {
			return nil, fmt.Errorf("list experiments scan: %w", err)
		}
		experiments = append(experiments, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list experiments rows: %w", err)
	}

	return &pb.ListExperimentsResponse{
		Experiments: experiments,
		Pagination:  pageInfo(page, perPage, total),
	}, nil
}

func (s *Store) CreateExperiment(ctx context.Context, req *pb.CreateExperimentRequest) (*pb.CreateExperimentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	e := req.GetExperiment()

	row := s.db.QueryRowContext(ctx,
		`INSERT INTO experiments (tenant_id, key, name, status, sticky_assignment, variants, allocation)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb)
		 RETURNING key, name, status, sticky_assignment, variants, allocation, created_at, updated_at`,
		tid, e.GetKey(), e.GetName(), e.GetStatus(), e.GetStickyAssignment(),
		marshalSlice(e.GetVariants()), marshalSlice(e.GetAllocation()),
	)
	result, err := scanExperiment(row)
	if err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	return &pb.CreateExperimentResponse{Experiment: result}, nil
}

func (s *Store) UpdateExperiment(ctx context.Context, req *pb.UpdateExperimentRequest) (*pb.UpdateExperimentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	e := req.GetExperiment()

	row := s.db.QueryRowContext(ctx,
		`UPDATE experiments
		 SET name = $1, status = $2, sticky_assignment = $3,
		     variants = $4::jsonb, allocation = $5::jsonb, updated_at = now()
		 WHERE tenant_id = $6 AND key = $7
		 RETURNING key, name, status, sticky_assignment, variants, allocation, created_at, updated_at`,
		e.GetName(), e.GetStatus(), e.GetStickyAssignment(),
		marshalSlice(e.GetVariants()), marshalSlice(e.GetAllocation()),
		tid, e.GetKey(),
	)
	result, err := scanExperiment(row)
	if err != nil {
		return nil, fmt.Errorf("update experiment: %w", err)
	}
	return &pb.UpdateExperimentResponse{Experiment: result}, nil
}

// --- API Keys ---

func (s *Store) GetAPIKeyByHash(ctx context.Context, req *pb.GetAPIKeyByHashRequest) (*pb.GetAPIKeyByHashResponse, error) {
	var (
		k         pb.APIKey
		tenantID  string
		createdBy sql.NullString
		createdAt time.Time
		revokedAt sql.NullTime
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, key_hash, key_prefix, scope, name, created_by, created_at, revoked_at
		 FROM api_keys WHERE key_hash = $1`,
		req.GetKeyHash(),
	).Scan(&k.Id, &tenantID, &k.KeyHash, &k.KeyPrefix, &k.Scope, &k.Name,
		&createdBy, &createdAt, &revokedAt)

	if err == sql.ErrNoRows {
		return &pb.GetAPIKeyByHashResponse{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}

	k.CreatedBy = createdBy.String
	k.CreatedAt = createdAt.Unix()
	if revokedAt.Valid {
		k.RevokedAt = revokedAt.Time.Unix()
	}

	return &pb.GetAPIKeyByHashResponse{
		Found:    true,
		ApiKey:   &k,
		TenantId: tenantID,
	}, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	tid := req.GetTenant().GetTenantId()
	page, perPage := paginate(req.GetPagination())

	var total int32
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1`, tid,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("list api keys count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, key_hash, key_prefix, scope, name, created_by, created_at, revoked_at
		 FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tid, perPage, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys query: %w", err)
	}
	defer rows.Close()

	var keys []*pb.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("list api keys scan: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list api keys rows: %w", err)
	}

	return &pb.ListAPIKeysResponse{
		ApiKeys:    keys,
		Pagination: pageInfo(page, perPage, total),
	}, nil
}

func (s *Store) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	tid := req.GetTenant().GetTenantId()
	k := req.GetApiKey()

	row := s.db.QueryRowContext(ctx,
		`INSERT INTO api_keys (id, tenant_id, key_hash, key_prefix, scope, name, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, key_hash, key_prefix, scope, name, created_by, created_at, revoked_at`,
		k.GetId(), tid, k.GetKeyHash(), k.GetKeyPrefix(), k.GetScope(), k.GetName(), k.GetCreatedBy(),
	)
	result, err := scanAPIKey(row)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &pb.CreateAPIKeyResponse{ApiKey: result}, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND tenant_id = $2`,
		req.GetKeyId(), req.GetTenant().GetTenantId(),
	)
	if err != nil {
		return nil, fmt.Errorf("revoke api key: %w", err)
	}
	return &pb.RevokeAPIKeyResponse{}, nil
}
