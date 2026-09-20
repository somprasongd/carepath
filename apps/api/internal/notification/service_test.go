package notification

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// fakes for every port; each records what the engine did so tests assert
// behavior, not logs.

type fakeQueueSource struct {
	visits []ActiveVisit
	steps  map[string][]QueueStep // visitID → steps
	errAt  string                 // visitID whose Queue call fails
}

func (f *fakeQueueSource) ActiveVisits(_ context.Context) ([]ActiveVisit, error) {
	return f.visits, nil
}

func (f *fakeQueueSource) Queue(_ context.Context, visitID string) ([]QueueStep, error) {
	if visitID == f.errAt {
		return nil, errors.New("queue read blew up")
	}
	return f.steps[visitID], nil
}

type fakeStore struct {
	claimed map[string]bool // "visit/step" → already sent
	enabled map[string]bool // visitID → pref (absent = enabled)
	calls   int             // SendOnce invocations
}

func (f *fakeStore) SendOnce(_ context.Context, visitID, stepKey, _ string) (bool, error) {
	f.calls++
	key := visitID + "/" + stepKey
	if f.claimed[key] {
		return false, nil
	}
	if f.claimed == nil {
		f.claimed = map[string]bool{}
	}
	f.claimed[key] = true
	return true, nil
}

func (f *fakeStore) Enabled(_ context.Context, visitID string) (bool, error) {
	if f.enabled == nil {
		return true, nil
	}
	enabled, ok := f.enabled[visitID]
	if !ok {
		return true, nil
	}
	return enabled, nil
}

func (f *fakeStore) SetEnabled(_ context.Context, visitID string, enabled bool) error {
	if f.enabled == nil {
		f.enabled = map[string]bool{}
	}
	f.enabled[visitID] = enabled
	return nil
}

type fakeRecipients struct {
	user map[string]string // visitID → LINE user id (absent = no recipient)
}

func (f *fakeRecipients) LineUser(_ context.Context, visitID string) (string, bool, error) {
	id, ok := f.user[visitID]
	return id, ok, nil
}

type fakeNotifier struct {
	channel   string
	sent      []string // one entry per delivered message "recipient|text"
	failEvery bool
}

func (f *fakeNotifier) Channel() string { return f.channel }

func (f *fakeNotifier) Notify(_ context.Context, recipient, text string) error {
	if f.failEvery {
		return errors.New("channel down")
	}
	f.sent = append(f.sent, recipient+"|"+text)
	return nil
}

func newEngine(queues QueueSource, store *fakeStore, recipients *fakeRecipients, notifier *fakeNotifier) *Service {
	return NewService(queues, store, recipients, notifier, slog.New(slog.DiscardHandler))
}

func TestSweepSendsWhenRecommendedStepIsNear(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1", ServicePoint: "ORDERTYPE:LAB"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 3}}},
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{user: map[string]string{"V1": "U123"}}, &fakeNotifier{channel: "line"}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent = %d messages, want 1", len(notifier.sent))
	}
	if got := notifier.sent[0]; got != "U123|"+nearQueueText {
		t.Fatalf("message = %q, want recipient plus the fixed no-PHI text", got)
	}
}

func TestSweepHoldsWhenQueueIsFar(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 4}}},
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{user: map[string]string{"V1": "U123"}}, &fakeNotifier{}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 0 || store.calls != 0 {
		t.Fatalf("far queue must not claim or send; sent=%d claims=%d", len(notifier.sent), store.calls)
	}
}

func TestSweepJudgesOnlyTheRecommendedStep(t *testing.T) {
	// The other actionable step is near, but the recommendation is the far
	// one — a message must not contradict the app's single primary action.
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "XRAY:1"}},
		steps: map[string][]QueueStep{"V1": {
			{StepKey: "XRAY:1", WaitingAhead: 7},
			{StepKey: "LAB:1", WaitingAhead: 0},
		}},
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{user: map[string]string{"V1": "U123"}}, &fakeNotifier{}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 0 || store.calls != 0 {
		t.Fatalf("near-but-not-recommended step must not trigger; sent=%d claims=%d", len(notifier.sent), store.calls)
	}
}

func TestSweepSendsExactlyOncePerStep(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 2}}},
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{user: map[string]string{"V1": "U123"}}, &fakeNotifier{}
	engine := newEngine(queues, store, recipients, notifier)
	// Recompute-happy sweep: run many times, as queue wobble and plan
	// recomputation (FR-28) would cause in production.
	for i := 0; i < 5; i++ {
		if err := engine.Sweep(context.Background()); err != nil {
			t.Fatalf("sweep %d: %v", i, err)
		}
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent = %d messages over 5 sweeps, want exactly 1", len(notifier.sent))
	}
}

func TestSweepOptOutSkipsWithoutClaiming(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 1}}},
	}
	store := &fakeStore{enabled: map[string]bool{"V1": false}}
	recipients, notifier := &fakeRecipients{user: map[string]string{"V1": "U123"}}, &fakeNotifier{}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 0 || store.calls != 0 {
		t.Fatalf("opted-out visit must not claim or send; sent=%d claims=%d", len(notifier.sent), store.calls)
	}
	// Re-enabling later can still deliver the step's message.
	store.enabled["V1"] = true
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep after re-enable: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent after re-enable = %d, want 1", len(notifier.sent))
	}
}

func TestSweepNoRecipientSkipsWithoutClaiming(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 0}}},
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{}, &fakeNotifier{}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 0 || store.calls != 0 {
		t.Fatalf("recipient-less visit must not claim (ADR-0013 §3); sent=%d claims=%d", len(notifier.sent), store.calls)
	}
	// A later real LINE login can still be notified.
	recipients.user = map[string]string{"V1": "U999"}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep after login: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent after login = %d, want 1", len(notifier.sent))
	}
}

func TestSweepFailedSendDoesNotResendNextSweep(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "V1", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"V1": {{StepKey: "LAB:1", WaitingAhead: 1}}},
	}
	store, recipients := &fakeStore{}, &fakeRecipients{user: map[string]string{"V1": "U123"}}
	failing := &fakeNotifier{failEvery: true}
	if err := newEngine(queues, store, recipients, failing).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	healthy := &fakeNotifier{}
	if err := newEngine(queues, store, recipients, healthy).Sweep(context.Background()); err != nil {
		t.Fatalf("recovery sweep: %v", err)
	}
	if len(healthy.sent) != 0 {
		t.Fatalf("a claimed-but-failed send must not retry (spam guard); sent=%d", len(healthy.sent))
	}
}

func TestSweepContinuesPastOneFailingVisit(t *testing.T) {
	queues := &fakeQueueSource{
		visits: []ActiveVisit{{VisitID: "BAD"}, {VisitID: "GOOD", StepKey: "LAB:1"}},
		steps:  map[string][]QueueStep{"GOOD": {{StepKey: "LAB:1", WaitingAhead: 2}}},
		errAt:  "BAD",
	}
	store, recipients, notifier := &fakeStore{}, &fakeRecipients{user: map[string]string{"GOOD": "U1"}}, &fakeNotifier{}
	if err := newEngine(queues, store, recipients, notifier).Sweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("one visit's queue failure must not stop the sweep; sent=%d", len(notifier.sent))
	}
}

func TestPrefRoundTrip(t *testing.T) {
	store := &fakeStore{}
	engine := newEngine(&fakeQueueSource{}, store, &fakeRecipients{}, &fakeNotifier{})
	pref, err := engine.GetPref(context.Background(), "V1")
	if err != nil {
		t.Fatalf("get pref: %v", err)
	}
	if !pref.Enabled {
		t.Fatalf("absent pref must read as enabled; got %+v", pref)
	}
	if _, err := engine.SetPref(context.Background(), "V1", false); err != nil {
		t.Fatalf("set pref: %v", err)
	}
	pref, err = engine.GetPref(context.Background(), "V1")
	if err != nil {
		t.Fatalf("re-get pref: %v", err)
	}
	if pref.Enabled {
		t.Fatalf("pref after opt-out = %+v, want disabled", pref)
	}
}
