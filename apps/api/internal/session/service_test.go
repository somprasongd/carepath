package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

type fakeVerifier struct {
	identity identity.Identity
	err      error
}

func (f *fakeVerifier) Verify(_ context.Context, _ string) (identity.Identity, error) {
	return f.identity, f.err
}

type fakeRepo struct {
	created *Session
	get     Session
	getErr  error
}

func (f *fakeRepo) Create(_ context.Context, s Session) error {
	f.created = &s
	return nil
}

func (f *fakeRepo) Get(_ context.Context, _ string) (Session, error) {
	return f.get, f.getErr
}

func TestCreateLineSuccess(t *testing.T) {
	verifier := &fakeVerifier{identity: identity.Identity{Source: "line", ExternalID: "U123", DisplayName: "Somchai"}}
	repo := &fakeRepo{}
	s := NewService(repo, verifier, false, time.Hour)

	got, err := s.Create(context.Background(), "line", "some-id-token")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if got.Identity != verifier.identity {
		t.Errorf("Identity = %+v, want %+v", got.Identity, verifier.identity)
	}
	if got.Token == "" {
		t.Error("Token is empty")
	}
	if repo.created == nil {
		t.Fatal("repo.Create was not called")
	}
}

func TestCreateLineVerifyFailure(t *testing.T) {
	wantErr := errors.New("bad token")
	verifier := &fakeVerifier{err: wantErr}
	s := NewService(&fakeRepo{}, verifier, false, time.Hour)

	if _, err := s.Create(context.Background(), "line", "bad-token"); !errors.Is(err, wantErr) {
		t.Errorf("Create() error = %v, want %v", err, wantErr)
	}
}

func TestCreateLineNotConfigured(t *testing.T) {
	s := NewService(&fakeRepo{}, nil, false, time.Hour)

	_, err := s.Create(context.Background(), "line", "token")
	if !errors.Is(err, ErrLineAuthNotConfigured) {
		t.Errorf("Create() error = %v, want ErrLineAuthNotConfigured", err)
	}
}

func TestCreateDemoAllowed(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil, true, time.Hour)

	got, err := s.Create(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if got.Identity.Source != "demo" {
		t.Errorf("Identity.Source = %q, want demo", got.Identity.Source)
	}
}

func TestCreateDemoDisabled(t *testing.T) {
	s := NewService(&fakeRepo{}, nil, false, time.Hour)

	_, err := s.Create(context.Background(), "demo", "")
	if !errors.Is(err, ErrDemoAuthDisabled) {
		t.Errorf("Create() error = %v, want ErrDemoAuthDisabled", err)
	}
}

func TestCreateUnknownSource(t *testing.T) {
	s := NewService(&fakeRepo{}, nil, true, time.Hour)

	_, err := s.Create(context.Background(), "google", "")
	if !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("Create() error = %v, want identity.ErrInvalidToken", err)
	}
}

func TestGetDelegatesToRepo(t *testing.T) {
	want := identity.Identity{Source: "line", ExternalID: "U123"}
	repo := &fakeRepo{get: Session{Identity: want}}
	s := NewService(repo, nil, false, time.Hour)

	got, err := s.Get(context.Background(), "some-token")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Get() = %+v, want %+v", got, want)
	}
}

func TestGetExpiredOrMissing(t *testing.T) {
	repo := &fakeRepo{getErr: ErrNotFound}
	s := NewService(repo, nil, false, time.Hour)

	_, err := s.Get(context.Background(), "stale-token")
	if apperr.KindOf(err) != apperr.KindUnauthorized {
		t.Errorf("Get() error kind = %v, want KindUnauthorized", apperr.KindOf(err))
	}
}
