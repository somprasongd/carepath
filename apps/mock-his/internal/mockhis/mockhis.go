// Package mockhis is the deterministic fake HIS: it owns visit and step
// status (the system of record per ADR-0008), exposes the canonical
// contract in packages/contracts/openapi/mock-his.yaml, and keeps an
// append-only event log. It is not a second implementation of CarePath
// business logic (see docs/integration/mock-his.md).
package mockhis

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Canonical visit statuses (contract schema VisitStatus).
const (
	VisitActive    = "ACTIVE"
	VisitCompleted = "COMPLETED"
	VisitCancelled = "CANCELLED"
)

// Canonical step statuses (contract schema StepStatus).
const (
	StepPending   = "PENDING"
	StepReady     = "READY"
	StepStarted   = "STARTED"
	StepCompleted = "COMPLETED"
	StepCancelled = "CANCELLED"
)

// Canonical event types (contract schema EventType).
const (
	EventVisitOpened      = "visit.opened"
	EventVisitUpdated     = "visit.updated"
	EventServiceRequested = "service.requested"
	EventServiceStarted   = "service.started"
	EventServiceCompleted = "service.completed"
	EventServiceCancelled = "service.cancelled"
)

// VisitStep mirrors the contract VisitStep schema.
type VisitStep struct {
	Sequence    int    `json:"sequence"`
	ServiceCode string `json:"serviceCode"`
	Status      string `json:"status"`
}

// Visit mirrors the contract Visit schema.
type Visit struct {
	VisitID    string      `json:"visitId"`
	PatientRef string      `json:"patientRef"`
	Status     string      `json:"status"`
	Steps      []VisitStep `json:"steps"`
}

// TransitionCommand mirrors the contract TransitionCommand schema. CommandID
// is the caller-assigned idempotency key.
type TransitionCommand struct {
	CommandID string `json:"commandId"`
	To        string `json:"to"`
}

// HISEvent mirrors the contract HISEvent envelope.
type HISEvent struct {
	EventID    string         `json:"eventId"`
	OccurredAt time.Time      `json:"occurredAt"`
	VisitID    string         `json:"visitId"`
	PatientRef string         `json:"patientRef"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
}

// ErrorKind maps to the contract's error responses.
type ErrorKind int

const (
	ErrBadInput ErrorKind = iota
	ErrNotFound
	ErrConflict
)

// Error is a Store-level failure the HTTP layer maps to a status code.
type Error struct {
	Kind ErrorKind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// Store is the in-memory HIS state. Seed data and seed events are
// deterministic so demo scenarios replay identically on every start.
type Store struct {
	mu       sync.Mutex
	visits   map[string]*Visit
	events   []HISEvent
	applied  map[string]bool
	eventSeq int
	visitSeq int
	seedBase time.Time
	now      func() time.Time
}

// defaultServiceCodes is the standard demo flow used when a created visit
// does not specify its own steps.
var defaultServiceCodes = []string{"REGISTRATION", "SCREENING", "DOCTOR", "LAB", "PHARMACY"}

// NewStore returns a seeded store. Seed events replay the visit history that
// the seed snapshot already reflects (opened, requested per step, completed
// steps), with timestamps derived from a fixed base for determinism.
func NewStore() *Store {
	s := &Store{
		visits:   map[string]*Visit{},
		applied:  map[string]bool{},
		visitSeq: 1, // VISIT-001 is the seed
		seedBase: time.Date(2026, 9, 19, 9, 0, 0, 0, time.FixedZone("ICT", 7*60*60)),
		now:      time.Now,
	}
	v := &Visit{
		VisitID:    "VISIT-001",
		PatientRef: "PATIENT-DEMO-001",
		Status:     VisitActive,
		Steps: []VisitStep{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: StepCompleted},
			{Sequence: 2, ServiceCode: "SCREENING", Status: StepCompleted},
			{Sequence: 3, ServiceCode: "DOCTOR", Status: StepCompleted},
			{Sequence: 4, ServiceCode: "LAB", Status: StepReady},
			{Sequence: 5, ServiceCode: "PHARMACY", Status: StepPending},
		},
	}
	s.visits[v.VisitID] = v

	s.appendSeed(EventVisitOpened, v, map[string]any{"status": v.Status})
	for _, step := range v.Steps {
		s.appendSeed(EventServiceRequested, v, map[string]any{
			"sequence": step.Sequence, "serviceCode": step.ServiceCode,
		})
	}
	for _, step := range v.Steps {
		if step.Status == StepCompleted {
			s.appendSeed(EventServiceCompleted, v, map[string]any{
				"sequence": step.Sequence, "serviceCode": step.ServiceCode,
			})
		}
	}
	return s
}

func (s *Store) appendSeed(eventType string, v *Visit, payload map[string]any) {
	s.eventSeq++
	s.events = append(s.events, HISEvent{
		EventID:    EventID(s.eventSeq),
		OccurredAt: s.seedBase.Add(time.Duration(s.eventSeq) * time.Minute),
		VisitID:    v.VisitID,
		PatientRef: v.PatientRef,
		Type:       eventType,
		Payload:    payload,
	})
}

// EventID formats the zero-padded, lexicographically sortable event id.
func EventID(seq int) string { return fmt.Sprintf("EVT-%06d", seq) }

// GetVisit returns the current snapshot.
func (s *Store) GetVisit(visitID string) (Visit, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.visits[visitID]
	if !ok {
		return Visit{}, false
	}
	return *copyVisit(v), true
}

// ListVisits returns every visit snapshot, ordered by visit id — the
// demo-driver surface the console selects from.
func (s *Store) ListVisits() []Visit {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.visits))
	for id := range s.visits {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Visit, 0, len(ids))
	for _, id := range ids {
		out = append(out, *copyVisit(s.visits[id]))
	}
	return out
}

// CreateVisit opens a new ACTIVE visit: the first step is READY, the rest
// PENDING. Codes fall back to the standard demo template when empty. Every
// fact is announced as canonical events (visit.opened, service.requested per
// step) so downstream consumers see the visit through the contract only.
func (s *Store) CreateVisit(patientRef string, serviceCodes []string) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	codes := make([]string, 0, len(serviceCodes))
	for _, code := range serviceCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			return Visit{}, &Error{Kind: ErrBadInput, Msg: "serviceCodes must be non-empty strings"}
		}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		codes = defaultServiceCodes
	}

	s.visitSeq++
	v := &Visit{
		VisitID:    fmt.Sprintf("VISIT-%03d", s.visitSeq),
		PatientRef: patientRef,
		Status:     VisitActive,
	}
	if v.PatientRef == "" {
		v.PatientRef = fmt.Sprintf("PATIENT-DEMO-%03d", s.visitSeq)
	}
	for i, code := range codes {
		status := StepPending
		if i == 0 {
			status = StepReady
		}
		v.Steps = append(v.Steps, VisitStep{Sequence: i + 1, ServiceCode: code, Status: status})
	}

	s.visits[v.VisitID] = v
	s.append(EventVisitOpened, v, map[string]any{"status": v.Status})
	for _, step := range v.Steps {
		s.append(EventServiceRequested, v, map[string]any{
			"sequence": step.Sequence, "serviceCode": step.ServiceCode,
		})
	}
	return *copyVisit(v), nil
}

// AddOrder appends one ordered service as a PENDING step after the current
// last sequence; it becomes READY when the preceding open step completes,
// like any other step. Announced as a canonical service.requested event.
func (s *Store) AddOrder(visitID, serviceCode string) (VisitStep, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	serviceCode = strings.TrimSpace(serviceCode)
	if serviceCode == "" {
		return VisitStep{}, &Error{Kind: ErrBadInput, Msg: "serviceCode is required"}
	}
	v, ok := s.visits[visitID]
	if !ok {
		return VisitStep{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	if v.Status != VisitActive {
		return VisitStep{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("visit is %s and cannot take new orders", v.Status)}
	}

	next := 1
	for _, step := range v.Steps {
		if step.Sequence >= next {
			next = step.Sequence + 1
		}
	}
	step := VisitStep{Sequence: next, ServiceCode: serviceCode, Status: StepPending}
	v.Steps = append(v.Steps, step)
	s.append(EventServiceRequested, v, map[string]any{
		"sequence": step.Sequence, "serviceCode": step.ServiceCode,
	})
	return step, nil
}

func copyVisit(v *Visit) *Visit {
	out := *v
	out.Steps = append([]VisitStep(nil), v.Steps...)
	return &out
}

// Transition applies a command to one step. Idempotent by contract: a replayed
// commandId, or a transition to the step's current status, is a no-op success
// that returns the current step. Completing a step also readies the next
// PENDING step, and completing the last open step completes the visit — both
// upstream HIS facts, reported as visit.updated events.
func (s *Store) Transition(visitID string, sequence int, cmd TransitionCommand) (VisitStep, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cmd.To != StepStarted && cmd.To != StepCompleted && cmd.To != StepCancelled {
		return VisitStep{}, &Error{Kind: ErrBadInput, Msg: fmt.Sprintf("unknown target status %q", cmd.To)}
	}
	v, ok := s.visits[visitID]
	if !ok {
		return VisitStep{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	idx := -1
	for i, st := range v.Steps {
		if st.Sequence == sequence {
			idx = i
			break
		}
	}
	if idx < 0 {
		return VisitStep{}, &Error{Kind: ErrNotFound, Msg: "step not found"}
	}

	step := v.Steps[idx]
	if s.applied[cmd.CommandID] || step.Status == cmd.To {
		s.applied[cmd.CommandID] = true
		return step, nil
	}

	from := step.Status
	switch {
	case from == StepCompleted || from == StepCancelled:
		return VisitStep{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("step %d is %s and cannot transition to %s", sequence, from, cmd.To)}
	case cmd.To == StepStarted && from != StepPending && from != StepReady:
		return VisitStep{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("step %d is %s and cannot transition to STARTED", sequence, from)}
	}

	step.Status = cmd.To
	v.Steps[idx] = step
	s.applied[cmd.CommandID] = true
	s.append(EventServiceType(cmd.To), v, map[string]any{
		"sequence": step.Sequence, "serviceCode": step.ServiceCode,
	})

	if cmd.To == StepCompleted {
		for i := range v.Steps {
			if v.Steps[i].Status == StepPending {
				v.Steps[i].Status = StepReady
				s.append(EventVisitUpdated, v, map[string]any{
					"sequence": v.Steps[i].Sequence, "serviceCode": v.Steps[i].ServiceCode,
				})
				break
			}
		}
	}
	// Closing the last open step ends the visit: completing it completes the
	// visit, cancelling it cancels the visit. Both are upstream HIS facts,
	// reported as visit.updated events.
	if (cmd.To == StepCompleted || cmd.To == StepCancelled) &&
		s.openSteps(v) == 0 && v.Status == VisitActive {
		if cmd.To == StepCompleted {
			v.Status = VisitCompleted
		} else {
			v.Status = VisitCancelled
		}
		s.append(EventVisitUpdated, v, map[string]any{"status": v.Status})
	}
	return step, nil
}

func (s *Store) openSteps(v *Visit) int {
	n := 0
	for _, st := range v.Steps {
		if st.Status != StepCompleted && st.Status != StepCancelled {
			n++
		}
	}
	return n
}

// CancelVisit cancels an ACTIVE visit outright: every open step is cancelled
// (canonical service.cancelled each) and the visit itself becomes CANCELLED
// (canonical visit.updated). Cancelling an already-CANCELLED visit is a no-op;
// a COMPLETED visit cannot be cancelled.
func (s *Store) CancelVisit(visitID string) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visits[visitID]
	if !ok {
		return Visit{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	switch v.Status {
	case VisitCancelled:
		return *copyVisit(v), nil
	case VisitCompleted:
		return Visit{}, &Error{Kind: ErrConflict, Msg: "visit is COMPLETED and cannot be cancelled"}
	}

	for i := range v.Steps {
		if v.Steps[i].Status != StepCompleted && v.Steps[i].Status != StepCancelled {
			v.Steps[i].Status = StepCancelled
			s.append(EventServiceCancelled, v, map[string]any{
				"sequence": v.Steps[i].Sequence, "serviceCode": v.Steps[i].ServiceCode,
			})
		}
	}
	v.Status = VisitCancelled
	s.append(EventVisitUpdated, v, map[string]any{"status": v.Status})
	return *copyVisit(v), nil
}

// EventServiceType maps a target step status to its service.* event type.
func EventServiceType(to string) string {
	switch to {
	case StepStarted:
		return EventServiceStarted
	case StepCompleted:
		return EventServiceCompleted
	case StepCancelled:
		return EventServiceCancelled
	}
	return EventVisitUpdated
}

func (s *Store) append(eventType string, v *Visit, payload map[string]any) {
	s.eventSeq++
	s.events = append(s.events, HISEvent{
		EventID:    EventID(s.eventSeq),
		OccurredAt: s.now(),
		VisitID:    v.VisitID,
		PatientRef: v.PatientRef,
		Type:       eventType,
		Payload:    payload,
	})
}

// Events returns up to limit events strictly after the cursor, oldest first,
// plus the next cursor (empty when the page is empty).
func (s *Store) Events(after string, limit int) ([]HISEvent, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	page := make([]HISEvent, 0, limit)
	for _, e := range s.events {
		if len(page) == limit {
			break
		}
		if after == "" || e.EventID > after {
			page = append(page, e)
		}
	}
	next := ""
	if n := len(page); n > 0 {
		next = page[n-1].EventID
	}
	return page, next
}
