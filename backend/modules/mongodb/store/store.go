package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pagination"
)

const (
	colFlags       = "flags"
	colAssignments = "assignments"
	colExperiments = "experiments"
	colAPIKeys     = "apikeys"
	fieldID        = "_id"
)

type Store struct {
	pb.UnimplementedPersistenceServiceServer
	db *mongo.Database
}

func New(db *mongo.Database) *Store {
	return &Store{db: db}
}

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(colAssignments).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	if err != nil {
		return fmt.Errorf("create assignments TTL index: %w", err)
	}

	_, err = db.Collection(colAPIKeys).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "key_hash", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("create apikeys key_hash index: %w", err)
	}
	return nil
}

func docID(parts ...string) string {
	return strings.Join(parts, ":")
}

func tenantFilter(tenantID string) bson.M {
	return bson.M{fieldID: bson.M{"$regex": "^" + tenantID + ":"}}
}

func idFilter(id string) bson.M {
	return bson.M{fieldID: id}
}

func listDocuments[D any, P any](ctx context.Context, col *mongo.Collection, filter bson.M, pr *pb.PageRequest, convert func(*D) P) ([]P, *pb.PageInfo, error) {
	page, perPage := pagination.Parse(pr)

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	skip := int64((page - 1) * perPage)
	cursor, err := col.Find(ctx, filter,
		options.Find().SetSkip(skip).SetLimit(int64(perPage)).SetSort(bson.D{{Key: fieldID, Value: 1}}))
	if err != nil {
		return nil, nil, err
	}

	var docs []D
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, nil, err
	}

	items := make([]P, 0, len(docs))
	for i := range docs {
		items = append(items, convert(&docs[i]))
	}
	return items, pagination.Info(page, perPage, int32(total)), nil
}

// --- BSON document types ---

type conditionDoc struct {
	Attribute string `bson:"attribute"`
	Operator  string `bson:"operator"`
	ValueJSON string `bson:"value_json"`
}

type evalResultDoc struct {
	Enabled bool   `bson:"enabled"`
	Variant string `bson:"variant"`
}

type ruleDoc struct {
	Conditions        []conditionDoc `bson:"conditions,omitempty"`
	RolloutPercentage int32          `bson:"rollout_percentage"`
	Variant           string         `bson:"variant"`
}

type flagDoc struct {
	ID            string         `bson:"_id"`
	Key           string         `bson:"key"`
	Type          string         `bson:"type"`
	Semantics     string         `bson:"semantics"`
	Enabled       bool           `bson:"enabled"`
	Description   string         `bson:"description,omitempty"`
	Rules         []ruleDoc      `bson:"rules,omitempty"`
	DefaultResult *evalResultDoc `bson:"default_result,omitempty"`
	CreatedBy     string         `bson:"created_by,omitempty"`
	UpdatedBy     string         `bson:"updated_by,omitempty"`
	CreatedAt     int64          `bson:"created_at"`
	UpdatedAt     int64          `bson:"updated_at"`
}

type assignmentDoc struct {
	ID         string     `bson:"_id"`
	SubjectID  string     `bson:"subject_id"`
	FlagKey    string     `bson:"flag_key"`
	Enabled    bool       `bson:"enabled"`
	Variant    string     `bson:"variant"`
	AssignedAt int64      `bson:"assigned_at"`
	ExpiresAt  *time.Time `bson:"expires_at,omitempty"`
}

type variantDoc struct {
	Key         string `bson:"key"`
	Description string `bson:"description,omitempty"`
}

type allocationDoc struct {
	VariantKey string `bson:"variant_key"`
	Percentage int32  `bson:"percentage"`
}

type experimentDoc struct {
	ID               string          `bson:"_id"`
	Key              string          `bson:"key"`
	Name             string          `bson:"name"`
	Status           string          `bson:"status"`
	StickyAssignment bool            `bson:"sticky_assignment"`
	Variants         []variantDoc    `bson:"variants,omitempty"`
	Allocation       []allocationDoc `bson:"allocation,omitempty"`
	CreatedAt        int64           `bson:"created_at"`
	UpdatedAt        int64           `bson:"updated_at"`
}

type apiKeyDoc struct {
	ID        string `bson:"_id"`
	APIKeyID  string `bson:"api_key_id"`
	TenantID  string `bson:"tenant_id"`
	KeyHash   string `bson:"key_hash"`
	KeyPrefix string `bson:"key_prefix"`
	Scope     string `bson:"scope"`
	Name      string `bson:"name"`
	CreatedBy string `bson:"created_by,omitempty"`
	CreatedAt int64  `bson:"created_at"`
	RevokedAt int64  `bson:"revoked_at,omitempty"`
}

// --- Conversion helpers ---

func flagToDoc(tid string, f *pb.FlagDefinition) *flagDoc {
	d := &flagDoc{
		ID:          docID(tid, f.GetKey()),
		Key:         f.GetKey(),
		Type:        f.GetType(),
		Semantics:   f.GetSemantics(),
		Enabled:     f.GetEnabled(),
		Description: f.GetDescription(),
		CreatedBy:   f.GetCreatedBy(),
		UpdatedBy:   f.GetUpdatedBy(),
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	for _, r := range f.GetRules() {
		rd := ruleDoc{
			RolloutPercentage: r.GetRolloutPercentage(),
			Variant:           r.GetVariant(),
		}
		for _, c := range r.GetConditions() {
			rd.Conditions = append(rd.Conditions, conditionDoc{
				Attribute: c.GetAttribute(),
				Operator:  c.GetOperator(),
				ValueJSON: c.GetValueJson(),
			})
		}
		d.Rules = append(d.Rules, rd)
	}
	if dr := f.GetDefaultResult(); dr != nil {
		d.DefaultResult = &evalResultDoc{
			Enabled: dr.GetEnabled(),
			Variant: dr.GetVariant(),
		}
	}
	return d
}

func flagFromDoc(d *flagDoc) *pb.FlagDefinition {
	f := &pb.FlagDefinition{
		Key:         d.Key,
		Type:        d.Type,
		Semantics:   d.Semantics,
		Enabled:     d.Enabled,
		Description: d.Description,
		CreatedBy:   d.CreatedBy,
		UpdatedBy:   d.UpdatedBy,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	for _, rd := range d.Rules {
		r := &pb.Rule{
			RolloutPercentage: rd.RolloutPercentage,
			Variant:           rd.Variant,
		}
		for _, cd := range rd.Conditions {
			r.Conditions = append(r.Conditions, &pb.Condition{
				Attribute: cd.Attribute,
				Operator:  cd.Operator,
				ValueJson: cd.ValueJSON,
			})
		}
		f.Rules = append(f.Rules, r)
	}
	if d.DefaultResult != nil {
		f.DefaultResult = &pb.EvalResult{
			Enabled: d.DefaultResult.Enabled,
			Variant: d.DefaultResult.Variant,
		}
	}
	return f
}

func experimentToDoc(tid string, e *pb.Experiment) *experimentDoc {
	d := &experimentDoc{
		ID:               docID(tid, e.GetKey()),
		Key:              e.GetKey(),
		Name:             e.GetName(),
		Status:           e.GetStatus(),
		StickyAssignment: e.GetStickyAssignment(),
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
	for _, v := range e.GetVariants() {
		d.Variants = append(d.Variants, variantDoc{
			Key:         v.GetKey(),
			Description: v.GetDescription(),
		})
	}
	for _, a := range e.GetAllocation() {
		d.Allocation = append(d.Allocation, allocationDoc{
			VariantKey: a.GetVariantKey(),
			Percentage: a.GetPercentage(),
		})
	}
	return d
}

func experimentFromDoc(d *experimentDoc) *pb.Experiment {
	e := &pb.Experiment{
		Key:              d.Key,
		Name:             d.Name,
		Status:           d.Status,
		StickyAssignment: d.StickyAssignment,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
	for _, vd := range d.Variants {
		e.Variants = append(e.Variants, &pb.Variant{
			Key:         vd.Key,
			Description: vd.Description,
		})
	}
	for _, ad := range d.Allocation {
		e.Allocation = append(e.Allocation, &pb.Allocation{
			VariantKey: ad.VariantKey,
			Percentage: ad.Percentage,
		})
	}
	return e
}

func apiKeyFromDoc(d *apiKeyDoc) *pb.APIKey {
	return &pb.APIKey{
		Id:        d.APIKeyID,
		KeyHash:   d.KeyHash,
		KeyPrefix: d.KeyPrefix,
		Scope:     d.Scope,
		Name:      d.Name,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
		RevokedAt: d.RevokedAt,
	}
}

// --- Flags ---

func (s *Store) CreateFlag(ctx context.Context, req *pb.CreateFlagRequest) (*pb.CreateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()
	now := time.Now().Unix()
	f.CreatedAt = now
	f.UpdatedAt = now

	doc := flagToDoc(tid, f)
	_, err := s.db.Collection(colFlags).ReplaceOne(ctx, idFilter(doc.ID), doc, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}
	return &pb.CreateFlagResponse{Flag: f}, nil
}

func (s *Store) GetFlag(ctx context.Context, req *pb.GetFlagRequest) (*pb.GetFlagResponse, error) {
	id := docID(req.GetTenant().GetTenantId(), req.GetFlagKey())
	var doc flagDoc
	err := s.db.Collection(colFlags).FindOne(ctx, idFilter(id)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &pb.GetFlagResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	return &pb.GetFlagResponse{Flag: flagFromDoc(&doc)}, nil
}

func (s *Store) ListFlags(ctx context.Context, req *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	flags, pi, err := listDocuments(ctx, s.db.Collection(colFlags),
		tenantFilter(req.GetTenant().GetTenantId()), req.GetPagination(),
		func(d *flagDoc) *pb.FlagDefinition { return flagFromDoc(d) })
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	return &pb.ListFlagsResponse{Flags: flags, Pagination: pi}, nil
}

func (s *Store) UpdateFlag(ctx context.Context, req *pb.UpdateFlagRequest) (*pb.UpdateFlagResponse, error) {
	tid := req.GetTenant().GetTenantId()
	f := req.GetFlag()
	id := docID(tid, f.GetKey())

	var old flagDoc
	err := s.db.Collection(colFlags).FindOne(ctx, idFilter(id)).Decode(&old)
	if err != nil {
		return nil, fmt.Errorf("update flag get: %w", err)
	}

	f.CreatedAt = old.CreatedAt
	f.UpdatedAt = time.Now().Unix()

	doc := flagToDoc(tid, f)
	_, err = s.db.Collection(colFlags).ReplaceOne(ctx, idFilter(id), doc)
	if err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}
	return &pb.UpdateFlagResponse{Flag: f}, nil
}

func (s *Store) DeleteFlag(ctx context.Context, req *pb.DeleteFlagRequest) (*pb.DeleteFlagResponse, error) {
	id := docID(req.GetTenant().GetTenantId(), req.GetFlagKey())
	_, err := s.db.Collection(colFlags).DeleteOne(ctx, idFilter(id))
	if err != nil {
		return nil, fmt.Errorf("delete flag: %w", err)
	}
	return &pb.DeleteFlagResponse{}, nil
}

// --- Assignments ---

func (s *Store) SaveAssignment(ctx context.Context, req *pb.SaveAssignmentRequest) (*pb.SaveAssignmentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	a := req.GetAssignment()

	assignedAt := time.Now().Unix()
	if a.GetAssignedAt() > 0 {
		assignedAt = a.GetAssignedAt()
	}

	doc := assignmentDoc{
		ID:         docID(tid, a.GetSubjectId(), a.GetFlagKey()),
		SubjectID:  a.GetSubjectId(),
		FlagKey:    a.GetFlagKey(),
		Enabled:    a.GetEnabled(),
		Variant:    a.GetVariant(),
		AssignedAt: assignedAt,
	}
	if a.GetExpiresAt() > 0 {
		t := time.Unix(a.GetExpiresAt(), 0)
		doc.ExpiresAt = &t
	}

	_, err := s.db.Collection(colAssignments).ReplaceOne(ctx, idFilter(doc.ID), doc, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("save assignment: %w", err)
	}
	return &pb.SaveAssignmentResponse{}, nil
}

func (s *Store) GetAssignment(ctx context.Context, req *pb.GetAssignmentRequest) (*pb.GetAssignmentResponse, error) {
	id := docID(req.GetTenant().GetTenantId(), req.GetSubjectId(), req.GetFlagKey())
	var doc assignmentDoc
	err := s.db.Collection(colAssignments).FindOne(ctx, idFilter(id)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}

	if doc.ExpiresAt != nil && doc.ExpiresAt.Before(time.Now()) {
		return &pb.GetAssignmentResponse{Found: false}, nil
	}

	a := &pb.Assignment{
		SubjectId:  doc.SubjectID,
		FlagKey:    doc.FlagKey,
		Enabled:    doc.Enabled,
		Variant:    doc.Variant,
		AssignedAt: doc.AssignedAt,
	}
	if doc.ExpiresAt != nil {
		a.ExpiresAt = doc.ExpiresAt.Unix()
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

	doc := experimentToDoc(tid, e)
	_, err := s.db.Collection(colExperiments).ReplaceOne(ctx, idFilter(doc.ID), doc, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	return &pb.CreateExperimentResponse{Experiment: e}, nil
}

func (s *Store) GetExperiment(ctx context.Context, req *pb.GetExperimentRequest) (*pb.GetExperimentResponse, error) {
	id := docID(req.GetTenant().GetTenantId(), req.GetExperimentKey())
	var doc experimentDoc
	err := s.db.Collection(colExperiments).FindOne(ctx, idFilter(id)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &pb.GetExperimentResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	return &pb.GetExperimentResponse{Experiment: experimentFromDoc(&doc)}, nil
}

func (s *Store) ListExperiments(ctx context.Context, req *pb.ListExperimentsRequest) (*pb.ListExperimentsResponse, error) {
	experiments, pi, err := listDocuments(ctx, s.db.Collection(colExperiments),
		tenantFilter(req.GetTenant().GetTenantId()), req.GetPagination(),
		func(d *experimentDoc) *pb.Experiment { return experimentFromDoc(d) })
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	return &pb.ListExperimentsResponse{Experiments: experiments, Pagination: pi}, nil
}

func (s *Store) UpdateExperiment(ctx context.Context, req *pb.UpdateExperimentRequest) (*pb.UpdateExperimentResponse, error) {
	tid := req.GetTenant().GetTenantId()
	e := req.GetExperiment()
	id := docID(tid, e.GetKey())

	var old experimentDoc
	err := s.db.Collection(colExperiments).FindOne(ctx, idFilter(id)).Decode(&old)
	if err != nil {
		return nil, fmt.Errorf("update experiment get: %w", err)
	}

	e.CreatedAt = old.CreatedAt
	e.UpdatedAt = time.Now().Unix()

	doc := experimentToDoc(tid, e)
	_, err = s.db.Collection(colExperiments).ReplaceOne(ctx, idFilter(id), doc)
	if err != nil {
		return nil, fmt.Errorf("update experiment: %w", err)
	}
	return &pb.UpdateExperimentResponse{Experiment: e}, nil
}

// --- API Keys ---

func (s *Store) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	tid := req.GetTenant().GetTenantId()
	k := req.GetApiKey()
	now := time.Now().Unix()

	doc := apiKeyDoc{
		ID:        docID(tid, k.GetId()),
		APIKeyID:  k.GetId(),
		TenantID:  tid,
		KeyHash:   k.GetKeyHash(),
		KeyPrefix: k.GetKeyPrefix(),
		Scope:     k.GetScope(),
		Name:      k.GetName(),
		CreatedBy: k.GetCreatedBy(),
		CreatedAt: now,
	}

	_, err := s.db.Collection(colAPIKeys).ReplaceOne(ctx, idFilter(doc.ID), doc, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &pb.CreateAPIKeyResponse{ApiKey: apiKeyFromDoc(&doc)}, nil
}

func (s *Store) GetAPIKeyByHash(ctx context.Context, req *pb.GetAPIKeyByHashRequest) (*pb.GetAPIKeyByHashResponse, error) {
	var doc apiKeyDoc
	err := s.db.Collection(colAPIKeys).FindOne(ctx, bson.M{"key_hash": req.GetKeyHash()}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &pb.GetAPIKeyByHashResponse{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &pb.GetAPIKeyByHashResponse{
		Found:    true,
		ApiKey:   apiKeyFromDoc(&doc),
		TenantId: doc.TenantID,
	}, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	keys, pi, err := listDocuments(ctx, s.db.Collection(colAPIKeys),
		tenantFilter(req.GetTenant().GetTenantId()), req.GetPagination(),
		func(d *apiKeyDoc) *pb.APIKey { return apiKeyFromDoc(d) })
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return &pb.ListAPIKeysResponse{ApiKeys: keys, Pagination: pi}, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	tid := req.GetTenant().GetTenantId()
	id := docID(tid, req.GetKeyId())

	var doc apiKeyDoc
	err := s.db.Collection(colAPIKeys).FindOne(ctx, idFilter(id)).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("revoke api key get: %w", err)
	}

	doc.RevokedAt = time.Now().Unix()
	_, err = s.db.Collection(colAPIKeys).ReplaceOne(ctx, idFilter(id), doc)
	if err != nil {
		return nil, fmt.Errorf("revoke api key: %w", err)
	}
	return &pb.RevokeAPIKeyResponse{}, nil
}
