package nbapi

import (
	"context"
	"net/http"
	"sort"
	"time"
)

// Event is an audit log entry — mirrors the spec's Event schema
// (shared/management/http/api/openapi.yml). ActivityCode is left as a plain
// string rather than a Go enum: the spec's enum has 80+ values and grows
// with every new feature, so validating against a hardcoded list here would
// only go stale.
type Event struct {
	ID             string            `json:"id"`
	Timestamp      time.Time         `json:"timestamp"`
	Activity       string            `json:"activity"`
	ActivityCode   string            `json:"activity_code"`
	InitiatorID    string            `json:"initiator_id"`
	InitiatorName  string            `json:"initiator_name"`
	InitiatorEmail string            `json:"initiator_email"`
	TargetID       string            `json:"target_id"`
	Meta           map[string]string `json:"meta"`
}

// ListAuditEvents returns every audit event on the account, most recent
// first — the API itself doesn't guarantee an order.
func (c *Client) ListAuditEvents(ctx context.Context, token string) ([]Event, error) {
	var events []Event
	if err := c.do(ctx, token, http.MethodGet, "/events/audit", nil, nil, &events); err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return events, nil
}
