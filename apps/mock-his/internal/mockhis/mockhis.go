// Package mockhis is the deterministic fake HIS: it owns visit, clinic, and
// order facts (the system of record per ADR-0008, amended by ADR-0009),
// exposes the canonical contract in packages/contracts/openapi/mock-his.yaml,
// and keeps an append-only event log. Per ADR-0009 it has no concept of an
// ordered patient journey — that is CarePath's journey planner's job — so
// this store never models steps, only the facts a real HIS could report. It
// is not a second implementation of CarePath business logic (see
// docs/integration/mock-his.md).
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

// Canonical visit types (contract schema VisitType).
const (
	VisitWalkin      = "WALKIN"
	VisitAppointment = "APPOINTMENT"
)

// Canonical order types (contract schema OrderType).
const (
	OrderTypeLab  = "LAB"
	OrderTypeXray = "XRAY"
	OrderTypeEKG  = "EKG"
	OrderTypeUS   = "US"
	OrderTypeDrug = "DRUG"
)

// Canonical order statuses (contract schema OrderStatus).
const (
	OrderPlaced    = "PLACED"
	OrderPerformed = "PERFORMED"
	OrderResulted  = "RESULTED"
	OrderCancelled = "CANCELLED"
)

// Canonical event types (contract schema EventType).
const (
	EventVisitOpened        = "visit.opened"
	EventVisitUpdated       = "visit.updated"
	EventVisitClosed        = "visit.closed"
	EventOrderPlaced        = "order.placed"
	EventOrderPerformed     = "order.performed"
	EventOrderResulted      = "order.resulted"
	EventOrderCancelled     = "order.cancelled"
	EventEncounterCompleted = "encounter.completed"
)

// Clinic mirrors the contract Clinic schema.
type Clinic struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// Order mirrors the contract Order schema.
type Order struct {
	OrderRef        string     `json:"orderRef"`
	OrderType       string     `json:"orderType"`
	OrderName       string     `json:"orderName"`
	OrderedByClinic string     `json:"orderedByClinic"`
	OrderedAt       time.Time  `json:"orderedAt"`
	Status          string     `json:"status"`
	PerformedAt     *time.Time `json:"performedAt,omitempty"`
	ResultedAt      *time.Time `json:"resultedAt,omitempty"`
}

// Visit mirrors the contract Visit schema.
type Visit struct {
	VisitID     string    `json:"visitId"`
	PatientRef  string    `json:"patientRef"`
	PatientName string    `json:"patientName,omitempty"`
	VisitType   string    `json:"visitType"`
	Status      string    `json:"status"`
	Clinics     []Clinic  `json:"clinics"`
	Orders      []Order   `json:"orders"`
	OpenedAt    time.Time `json:"openedAt"`
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
	orderVis map[string]string // orderRef -> visitId, for the /demo/orders/{orderRef}/* endpoints
	events   []HISEvent
	eventSeq int
	visitSeq int
	orderSeq int
	seedBase time.Time
	now      func() time.Time
}

// NewStore returns a seeded store. The seed visit is an appointment patient
// with one pre-visit lab order — the ADR-0009 example flow — so the console
// and CarePath's journey planner both have something to show on first boot.
func NewStore() *Store {
	s := &Store{
		visits:   map[string]*Visit{},
		orderVis: map[string]string{},
		visitSeq: 1, // VISIT-001 is the seed
		orderSeq: 1, // ORD-001 is the seed
		seedBase: time.Date(2026, 9, 19, 9, 0, 0, 0, time.FixedZone("ICT", 7*60*60)),
		now:      time.Now,
	}
	v := &Visit{
		VisitID: "VISIT-001", PatientRef: "PATIENT-DEMO-001", PatientName: "สมชาย ใจดี",
		VisitType: VisitAppointment, Status: VisitActive,
		Clinics:  []Clinic{{Code: "MED", Name: "อายุรกรรม"}},
		OpenedAt: s.seedBase,
		Orders: []Order{
			{OrderRef: "ORD-001", OrderType: OrderTypeLab, OrderName: "CBC", OrderedByClinic: "MED",
				OrderedAt: s.seedBase.Add(-30 * time.Minute), Status: OrderPlaced},
		},
	}
	s.visits[v.VisitID] = v
	s.orderVis["ORD-001"] = v.VisitID

	s.appendSeed(EventVisitOpened, v, map[string]any{
		"patientName": v.PatientName, "visitType": v.VisitType, "clinics": v.Clinics,
	})
	s.appendSeed(EventOrderPlaced, v, orderPayload(v.Orders[0]))
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

func orderPayload(o Order) map[string]any {
	return map[string]any{
		"orderRef": o.OrderRef, "orderType": o.OrderType, "orderName": o.OrderName,
		"orderedByClinic": o.OrderedByClinic, "orderedAt": o.OrderedAt,
	}
}

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

// ListVisits returns every visit snapshot, ordered by visit id.
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

func copyVisit(v *Visit) *Visit {
	out := *v
	out.Clinics = append([]Clinic(nil), v.Clinics...)
	out.Orders = append([]Order(nil), v.Orders...)
	return &out
}

// OpenVisitClinic is one clinic assignment on an OpenVisit call.
type OpenVisitClinic struct{ Code, Name string }

// OpenVisitOrder is one pre-visit order on an OpenVisit call.
type OpenVisitOrder struct{ OrderType, OrderName, OrderedByClinic string }

// OpenVisit opens a new ACTIVE visit (mirrors registration in a real HIS).
// Every fact is announced as canonical events (visit.opened, order.placed
// per pre-visit order) so downstream consumers see the visit through the
// contract only.
func (s *Store) OpenVisit(patientRef, patientName, visitType string, clinics []OpenVisitClinic, orders []OpenVisitOrder) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if visitType != VisitWalkin && visitType != VisitAppointment {
		return Visit{}, &Error{Kind: ErrBadInput, Msg: fmt.Sprintf("visitType must be %s or %s", VisitWalkin, VisitAppointment)}
	}
	if len(clinics) == 0 {
		return Visit{}, &Error{Kind: ErrBadInput, Msg: "at least one clinic is required"}
	}
	for _, c := range clinics {
		if strings.TrimSpace(c.Code) == "" {
			return Visit{}, &Error{Kind: ErrBadInput, Msg: "clinicCode must be non-empty"}
		}
	}

	s.visitSeq++
	v := &Visit{
		VisitID: fmt.Sprintf("VISIT-%03d", s.visitSeq), PatientRef: patientRef, PatientName: patientName,
		VisitType: visitType, Status: VisitActive, OpenedAt: s.now(),
	}
	if v.PatientRef == "" {
		v.PatientRef = fmt.Sprintf("PATIENT-DEMO-%03d", s.visitSeq)
	}
	for _, c := range clinics {
		v.Clinics = append(v.Clinics, Clinic{Code: c.Code, Name: c.Name})
	}
	s.visits[v.VisitID] = v
	s.append(EventVisitOpened, v, map[string]any{
		"patientName": v.PatientName, "visitType": v.VisitType, "clinics": v.Clinics,
	})

	for _, o := range orders {
		if _, err := s.placeOrderLocked(v, o.OrderType, o.OrderName, o.OrderedByClinic, v.OpenedAt.Add(-time.Minute)); err != nil {
			return Visit{}, err
		}
	}
	return *copyVisit(v), nil
}

// AddClinic assigns an additional clinic to an ACTIVE visit.
func (s *Store) AddClinic(visitID, clinicCode, clinicName string) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(clinicCode) == "" {
		return Visit{}, &Error{Kind: ErrBadInput, Msg: "clinicCode is required"}
	}
	v, ok := s.visits[visitID]
	if !ok {
		return Visit{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	if v.Status != VisitActive {
		return Visit{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("visit is %s and cannot take new clinics", v.Status)}
	}
	v.Clinics = append(v.Clinics, Clinic{Code: clinicCode, Name: clinicName})
	s.append(EventVisitUpdated, v, map[string]any{"clinics": v.Clinics})
	return *copyVisit(v), nil
}

// PlaceOrder places an order against an ACTIVE visit.
func (s *Store) PlaceOrder(visitID, orderType, orderName, orderedByClinic string) (Order, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visits[visitID]
	if !ok {
		return Order{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	if v.Status != VisitActive {
		return Order{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("visit is %s and cannot take new orders", v.Status)}
	}
	order, err := s.placeOrderLocked(v, orderType, orderName, orderedByClinic, s.now())
	if err != nil {
		return Order{}, err
	}
	return *order, nil
}

func (s *Store) placeOrderLocked(v *Visit, orderType, orderName, orderedByClinic string, orderedAt time.Time) (*Order, *Error) {
	switch orderType {
	case OrderTypeLab, OrderTypeXray, OrderTypeEKG, OrderTypeUS, OrderTypeDrug:
	default:
		return nil, &Error{Kind: ErrBadInput, Msg: fmt.Sprintf("unknown orderType %q", orderType)}
	}
	orderName = strings.TrimSpace(orderName)
	orderedByClinic = strings.TrimSpace(orderedByClinic)
	if orderName == "" || orderedByClinic == "" {
		return nil, &Error{Kind: ErrBadInput, Msg: "orderName and orderedByClinic are required"}
	}

	s.orderSeq++
	order := Order{
		OrderRef: fmt.Sprintf("ORD-%03d", s.orderSeq), OrderType: orderType, OrderName: orderName,
		OrderedByClinic: orderedByClinic, OrderedAt: orderedAt, Status: OrderPlaced,
	}
	v.Orders = append(v.Orders, order)
	s.orderVis[order.OrderRef] = v.VisitID
	s.append(EventOrderPlaced, v, orderPayload(order))
	return &v.Orders[len(v.Orders)-1], nil
}

// findOrder must be called with s.mu held.
func (s *Store) findOrder(orderRef string) (*Visit, int, *Error) {
	visitID, ok := s.orderVis[orderRef]
	if !ok {
		return nil, -1, &Error{Kind: ErrNotFound, Msg: "order not found"}
	}
	v := s.visits[visitID]
	for i := range v.Orders {
		if v.Orders[i].OrderRef == orderRef {
			return v, i, nil
		}
	}
	return nil, -1, &Error{Kind: ErrNotFound, Msg: "order not found"}
}

// MarkPerformed marks an order as performed — the procedure happened.
func (s *Store) MarkPerformed(orderRef string) (Order, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, i, err := s.findOrder(orderRef)
	if err != nil {
		return Order{}, err
	}
	if v.Orders[i].Status != OrderPlaced {
		return Order{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("order %s is %s and cannot be marked performed", orderRef, v.Orders[i].Status)}
	}
	now := s.now()
	v.Orders[i].Status = OrderPerformed
	v.Orders[i].PerformedAt = &now
	s.append(EventOrderPerformed, v, map[string]any{"orderRef": orderRef, "performedAt": now})
	return v.Orders[i], nil
}

// MarkResulted marks an order's result as reported. Only valid once
// PERFORMED — a result cannot exist before the procedure did.
func (s *Store) MarkResulted(orderRef string) (Order, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, i, err := s.findOrder(orderRef)
	if err != nil {
		return Order{}, err
	}
	if v.Orders[i].Status != OrderPerformed {
		return Order{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("order %s is %s and cannot be marked resulted", orderRef, v.Orders[i].Status)}
	}
	now := s.now()
	v.Orders[i].Status = OrderResulted
	v.Orders[i].ResultedAt = &now
	s.append(EventOrderResulted, v, map[string]any{"orderRef": orderRef, "resultedAt": now})
	return v.Orders[i], nil
}

// CancelOrder cancels an order that has not yet resulted.
func (s *Store) CancelOrder(orderRef string) (Order, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, i, err := s.findOrder(orderRef)
	if err != nil {
		return Order{}, err
	}
	if v.Orders[i].Status == OrderResulted || v.Orders[i].Status == OrderCancelled {
		return Order{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("order %s is already %s", orderRef, v.Orders[i].Status)}
	}
	v.Orders[i].Status = OrderCancelled
	s.append(EventOrderCancelled, v, map[string]any{"orderRef": orderRef})
	return v.Orders[i], nil
}

// CompleteEncounter announces that the given clinic is finished examining
// the patient for this round (ADR-0009 §4).
func (s *Store) CompleteEncounter(visitID, clinicCode string) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visits[visitID]
	if !ok {
		return Visit{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	found := false
	for _, c := range v.Clinics {
		if c.Code == clinicCode {
			found = true
			break
		}
	}
	if !found {
		return Visit{}, &Error{Kind: ErrNotFound, Msg: fmt.Sprintf("clinic %s is not assigned to this visit", clinicCode)}
	}
	now := s.now()
	s.append(EventEncounterCompleted, v, map[string]any{"clinicCode": clinicCode, "completedAt": now})
	return *copyVisit(v), nil
}

// CompleteVisit completes an ACTIVE visit.
func (s *Store) CompleteVisit(visitID string) (Visit, *Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visits[visitID]
	if !ok {
		return Visit{}, &Error{Kind: ErrNotFound, Msg: "visit not found"}
	}
	if v.Status != VisitActive {
		return Visit{}, &Error{Kind: ErrConflict, Msg: fmt.Sprintf("visit is %s and cannot be completed", v.Status)}
	}
	v.Status = VisitCompleted
	s.append(EventVisitClosed, v, map[string]any{"status": v.Status})
	return *copyVisit(v), nil
}

// CancelVisit cancels an ACTIVE visit outright: every open order is
// cancelled (canonical order.cancelled each) and the visit itself becomes
// CANCELLED (canonical visit.closed). Cancelling an already-CANCELLED visit
// is a no-op; a COMPLETED visit cannot be cancelled.
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

	for i := range v.Orders {
		if v.Orders[i].Status == OrderPlaced || v.Orders[i].Status == OrderPerformed {
			v.Orders[i].Status = OrderCancelled
			s.append(EventOrderCancelled, v, map[string]any{"orderRef": v.Orders[i].OrderRef})
		}
	}
	v.Status = VisitCancelled
	s.append(EventVisitClosed, v, map[string]any{"status": v.Status})
	return *copyVisit(v), nil
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
