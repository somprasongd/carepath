package share

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"time"

	"carepath/apps/api/internal/journey"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
)

// Service is the only entry point other modules may call.
type Service interface {
	// CreateLink mints a new share link for visitID. The raw token is
	// returned exactly once; only its sha256 is stored.
	CreateLink(ctx context.Context, visitID string) (LinkSecret, error)
	// Resolve exchanges a share token for the redacted SharedJourney. Every
	// failure — unknown, expired, revoked token — is the same ErrInvalidLink.
	Resolve(ctx context.Context, token string) (SharedJourney, error)
	// RevokeAll revokes every still-active link of the visit ("หยุดแชร์").
	// Idempotent: revoking when none are active is a no-op.
	RevokeAll(ctx context.Context, visitID string) error
}

type service struct {
	repo     Repo
	journeys journey.Service
	tx       db.Transactor
	ttl      time.Duration
}

func NewService(repo Repo, journeys journey.Service, tx db.Transactor, ttl time.Duration) Service {
	return &service{repo: repo, journeys: journeys, tx: tx, ttl: ttl}
}

func (s *service) CreateLink(ctx context.Context, visitID string) (LinkSecret, error) {
	// The visit must exist in the projection — a shareable journey is the
	// whole point, and the FK would reject the insert anyway.
	if _, err := s.journeys.GetVisit(ctx, visitID); err != nil {
		return LinkSecret{}, err
	}

	count, err := s.repo.CountActive(ctx, visitID)
	if err != nil {
		return LinkSecret{}, err
	}
	if count >= MaxActiveLinks {
		return LinkSecret{}, ErrTooManyLinks
	}

	token, err := newToken()
	if err != nil {
		return LinkSecret{}, apperr.Wrapf(apperr.KindInternal, err, "share: generate token")
	}
	now := time.Now()
	link := ShareLink{
		TokenHash: hashToken(token), VisitID: visitID,
		CreatedAt: now, ExpiresAt: now.Add(s.ttl),
	}
	if err := s.repo.Create(ctx, link); err != nil {
		return LinkSecret{}, err
	}
	log := logger.FromContext(ctx)
	log.Info("share link created", "visit_id", visitID, "expires_at", link.ExpiresAt.UTC().String())
	return LinkSecret{Token: token, ExpiresAt: link.ExpiresAt}, nil
}

func (s *service) Resolve(ctx context.Context, token string) (SharedJourney, error) {
	link, err := s.repo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return SharedJourney{}, ErrInvalidLink
	}
	if !link.Active(time.Now()) {
		return SharedJourney{}, ErrInvalidLink
	}

	// The FK cascades on visit deletion, so a resolved link always has a
	// visit; a projection hiccup still folds into the same single answer.
	view, err := s.journeys.GetJourney(ctx, link.VisitID)
	if err != nil {
		return SharedJourney{}, ErrInvalidLink
	}
	return BuildSharedJourney(view, link.ExpiresAt), nil
}

func (s *service) RevokeAll(ctx context.Context, visitID string) error {
	revoked, err := s.repo.RevokeActive(ctx, visitID)
	if err != nil {
		return err
	}
	log := logger.FromContext(ctx)
	log.Info("share links revoked", "visit_id", visitID, "count", revoked)
	return nil
}

// BuildSharedJourney redacts a journey view into what a relative may see
// (ADR-0011 §3). Redaction is by construction: nothing sensitive is passed
// in, so nothing sensitive can come out. The unit test pins this by asserting
// the marshalled JSON contains none of the sensitive substrings.
func BuildSharedJourney(view journey.View, expiresAt time.Time) SharedJourney {
	shared := SharedJourney{Status: StatusWaiting, UpdatedAt: view.SyncedAt, ExpiresAt: expiresAt}
	if view.Completed {
		shared.Status = StatusDone
		return shared
	}

	// The relative's question is "ตอนนี้อยู่ไหน": a step being served wins over
	// one merely waiting; the recommended step is the waiting fallback.
	var current *journey.StepView
	for i := range view.Steps {
		if view.Steps[i].Status == journey.StepStarted {
			current = &view.Steps[i]
			break
		}
	}
	if current == nil && view.Recommended != nil {
		current = view.Recommended
	}
	if current == nil {
		return shared // nothing actionable yet — still just WAITING
	}

	step := &SharedStep{Title: stepTitle(*current), Status: StatusWaiting}
	if current.Status == journey.StepStarted {
		shared.Status = StatusInService
		step.Status = StatusInService
	}
	if sp := current.ServicePoint; sp != nil {
		step.ServicePointName = displayServicePointName(sp.Name)
		if sp.Place != nil && sp.Place.Floor != nil {
			step.FloorName = sp.Place.Floor.Name
		}
	}
	shared.CurrentStep = step
	return shared
}

// kindTitles maps a step kind to the Thai text a relative sees. Clinic codes,
// step keys, and kinds themselves never appear — the label is the whole story.
var kindTitles = map[string]string{
	journey.KindRegistration: "ลงทะเบียน",
	journey.KindLab:          "เจาะเลือด / ส่งตรวจแล็บ",
	journey.KindXray:         "ถ่ายภาพเอกซเรย์",
	journey.KindEKG:          "ตรวจคลื่นไฟฟ้าหัวใจ",
	journey.KindUltrasound:   "ตรวจอัลตราซาวด์",
	journey.KindClinic:       "พบแพทย์",
	journey.KindCashier:      "ชำระค่าบริการ",
	journey.KindPharmacy:     "รับยา",
}

func stepTitle(step journey.StepView) string {
	if step.Kind == journey.KindClinic && step.Round != nil && *step.Round > 1 {
		return "กลับมาพบแพทย์"
	}
	if title, ok := kindTitles[step.Kind]; ok {
		return title
	}
	return "ขั้นตอนการรักษา"
}

// codeSuffix strips the parenthesised internal code service points carry in
// their display names ("อายุรกรรม (MED)" → "อายุรกรรม"): the clinic code is
// exactly what SharedJourney must not reveal.
var codeSuffix = regexp.MustCompile(`\s+\([A-Za-z0-9:_-]+\)$`)

func displayServicePointName(name string) string {
	return codeSuffix.ReplaceAllString(name, "")
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken is sha256 hex — a fast hash is correct for a 256-bit random
// token (contrast argon2id for passwords; same reasoning as refresh_token).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
