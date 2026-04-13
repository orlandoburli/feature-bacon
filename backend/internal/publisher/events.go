package publisher

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	EventFlagCreated         = "flag.created"
	EventFlagUpdated         = "flag.updated"
	EventFlagDeleted         = "flag.deleted"
	EventExperimentCreated   = "experiment.created"
	EventExperimentUpdated   = "experiment.updated"
	EventExperimentStarted   = "experiment.started"
	EventExperimentPaused    = "experiment.paused"
	EventExperimentCompleted = "experiment.completed"
	EventExposure            = "experiment.exposure"
)

func NewEvent(eventType, tenantID string, payload any) *pb.Event {
	payloadJSON, _ := json.Marshal(payload)
	return &pb.Event{
		EventId:     uuid.New().String(),
		EventType:   eventType,
		TenantId:    tenantID,
		Timestamp:   time.Now().Unix(),
		PayloadJson: string(payloadJSON),
	}
}
