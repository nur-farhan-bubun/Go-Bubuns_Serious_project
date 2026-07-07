package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ride-sharing/user-service/internal/domain"
)

// ─── Mock Repositories ──────────────────────────────────────────────────────

type mockDatingRepo struct {
	profile *domain.DatingProfile
	err     error
}

func (m *mockDatingRepo) GetByUserID(_ context.Context, userID string) (*domain.DatingProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.profile == nil || m.profile.UserID != userID {
		return nil, errors.New("dating profile not found for user: " + userID)
	}
	return m.profile, nil
}

func (m *mockDatingRepo) Upsert(_ context.Context, profile *domain.DatingProfile) error {
	if m.err != nil {
		return m.err
	}
	m.profile = profile
	return nil
}

func (m *mockDatingRepo) Delete(_ context.Context, userID string) error {
	if m.err != nil {
		return m.err
	}
	if m.profile == nil || m.profile.UserID != userID {
		return errors.New("dating profile not found for user: " + userID)
	}
	m.profile = nil
	return nil
}

type mockWorkerRepo struct {
	profile *domain.WorkerProfile
	err     error
}

func (m *mockWorkerRepo) GetByUserID(_ context.Context, userID string) (*domain.WorkerProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.profile == nil || m.profile.UserID != userID {
		return nil, errors.New("worker profile not found for user: " + userID)
	}
	return m.profile, nil
}

func (m *mockWorkerRepo) Upsert(_ context.Context, profile *domain.WorkerProfile) error {
	if m.err != nil {
		return m.err
	}
	m.profile = profile
	return nil
}

func (m *mockWorkerRepo) Delete(_ context.Context, userID string) error {
	if m.err != nil {
		return m.err
	}
	if m.profile == nil || m.profile.UserID != userID {
		return errors.New("worker profile not found for user: " + userID)
	}
	m.profile = nil
	return nil
}

type mockPhotoRepo struct {
	photos    []*domain.ProfilePhoto
	err       error
	idCounter int
}

func (m *mockPhotoRepo) ListByUserID(_ context.Context, userID string) ([]*domain.ProfilePhoto, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.ProfilePhoto
	for _, p := range m.photos {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPhotoRepo) GetByID(_ context.Context, photoID string) (*domain.ProfilePhoto, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, p := range m.photos {
		if p.ID == photoID {
			return p, nil
		}
	}
	return nil, errors.New("photo not found: " + photoID)
}

func (m *mockPhotoRepo) Create(_ context.Context, photo *domain.ProfilePhoto) error {
	if m.err != nil {
		return m.err
	}
	if photo.ID == "" {
		m.idCounter++
		photo.ID = fmt.Sprintf("mock-photo-%d", m.idCounter)
	}
	photo.CreatedAt = time.Now().UTC()
	m.photos = append(m.photos, photo)
	return nil
}

func (m *mockPhotoRepo) Delete(_ context.Context, photoID string) error {
	if m.err != nil {
		return m.err
	}
	for i, p := range m.photos {
		if p.ID == photoID {
			m.photos = append(m.photos[:i], m.photos[i+1:]...)
			return nil
		}
	}
	return errors.New("photo not found: " + photoID)
}

func (m *mockPhotoRepo) SetPrimary(_ context.Context, photoID, userID string) error {
	if m.err != nil {
		return m.err
	}
	found := false
	for _, p := range m.photos {
		if p.UserID == userID {
			p.IsPrimary = p.ID == photoID
			if p.ID == photoID {
				found = true
			}
		}
	}
	if !found {
		return errors.New("photo not found: " + photoID)
	}
	return nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func newTestService(dating DatingProfileRepository, worker WorkerProfileRepository, photo PhotoRepository) *Service {
	return &Service{
		datingRepo: dating,
		workerRepo: worker,
		photoRepo:  photo,
	}
}

func fixedDate() time.Time {
	return time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
}

// ─── Dating Profile Tests ───────────────────────────────────────────────────

func TestGetDatingProfile_Success(t *testing.T) {
	mock := &mockDatingRepo{
		profile: &domain.DatingProfile{
			UserID:       "user-1",
			Gender:       "MALE",
			InterestedIn: "FEMALE",
			BirthDate:    fixedDate(),
		},
	}
	svc := newTestService(mock, nil, nil)

	profile, err := svc.GetDatingProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Gender != "MALE" {
		t.Errorf("expected MALE, got %s", profile.Gender)
	}
	if profile.InterestedIn != "FEMALE" {
		t.Errorf("expected FEMALE, got %s", profile.InterestedIn)
	}
	if !profile.BirthDate.Equal(fixedDate()) {
		t.Errorf("expected birth date %v, got %v", fixedDate(), profile.BirthDate)
	}
}

func TestGetDatingProfile_NotFound(t *testing.T) {
	svc := newTestService(&mockDatingRepo{}, nil, nil)

	_, err := svc.GetDatingProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateDatingProfile_Create(t *testing.T) {
	mock := &mockDatingRepo{}
	svc := newTestService(mock, nil, nil)

	profile := &domain.DatingProfile{
		UserID:       "user-1",
		Gender:       "FEMALE",
		InterestedIn: "MALE",
		BirthDate:    fixedDate(),
		HeightCm:     intPtr(165),
	}

	err := svc.UpdateDatingProfile(context.Background(), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was stored
	retrieved, err := svc.GetDatingProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.Gender != "FEMALE" {
		t.Errorf("expected FEMALE, got %s", retrieved.Gender)
	}
	if retrieved.HeightCm == nil || *retrieved.HeightCm != 165 {
		t.Errorf("expected height 165, got %v", retrieved.HeightCm)
	}
}

func TestUpdateDatingProfile_UpdateExisting(t *testing.T) {
	mock := &mockDatingRepo{
		profile: &domain.DatingProfile{
			UserID:       "user-1",
			Gender:       "MALE",
			InterestedIn: "FEMALE",
			BirthDate:    fixedDate(),
		},
	}
	svc := newTestService(mock, nil, nil)

	updated := &domain.DatingProfile{
		UserID:       "user-1",
		Gender:       "MALE",
		InterestedIn: "BOTH",
		BirthDate:    fixedDate(),
		RelationshipGoal: "SERIOUS",
	}

	err := svc.UpdateDatingProfile(context.Background(), updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := svc.GetDatingProfile(context.Background(), "user-1")
	if retrieved.InterestedIn != "BOTH" {
		t.Errorf("expected BOTH, got %s", retrieved.InterestedIn)
	}
	if retrieved.RelationshipGoal != "SERIOUS" {
		t.Errorf("expected SERIOUS, got %s", retrieved.RelationshipGoal)
	}
}

func TestDeleteDatingProfile_Success(t *testing.T) {
	mock := &mockDatingRepo{
		profile: &domain.DatingProfile{
			UserID:       "user-1",
			Gender:       "MALE",
			InterestedIn: "FEMALE",
			BirthDate:    fixedDate(),
		},
	}
	svc := newTestService(mock, nil, nil)

	err := svc.DeleteDatingProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetDatingProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestDeleteDatingProfile_NotFound(t *testing.T) {
	svc := newTestService(&mockDatingRepo{}, nil, nil)

	err := svc.DeleteDatingProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Worker Profile Tests ───────────────────────────────────────────────────

func TestGetWorkerProfile_Success(t *testing.T) {
	mock := &mockWorkerRepo{
		profile: &domain.WorkerProfile{
			UserID:             "user-1",
			Skills:             []string{"Delivery", "Plumbing"},
			IsAvailable:        true,
			CompletedJobsCount: 10,
			RatingAvg:          4.5,
		},
	}
	svc := newTestService(nil, mock, nil)

	profile, err := svc.GetWorkerProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profile.Skills) != 2 || profile.Skills[0] != "Delivery" {
		t.Errorf("expected skills [Delivery Plumbing], got %v", profile.Skills)
	}
	if !profile.IsAvailable {
		t.Error("expected IsAvailable to be true")
	}
	if profile.CompletedJobsCount != 10 {
		t.Errorf("expected 10 jobs, got %d", profile.CompletedJobsCount)
	}
	if profile.RatingAvg != 4.5 {
		t.Errorf("expected rating 4.5, got %f", profile.RatingAvg)
	}
}

func TestGetWorkerProfile_NotFound(t *testing.T) {
	svc := newTestService(nil, &mockWorkerRepo{}, nil)

	_, err := svc.GetWorkerProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateWorkerProfile_Create(t *testing.T) {
	mock := &mockWorkerRepo{}
	svc := newTestService(nil, mock, nil)

	rate := 25.50
	profile := &domain.WorkerProfile{
		UserID:      "user-1",
		Skills:      []string{"Cooking", "Cleaning"},
		HourlyRate:  &rate,
		IsAvailable: true,
	}

	err := svc.UpdateWorkerProfile(context.Background(), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := svc.GetWorkerProfile(context.Background(), "user-1")
	if len(retrieved.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(retrieved.Skills))
	}
	if retrieved.HourlyRate == nil || *retrieved.HourlyRate != 25.50 {
		t.Errorf("expected hourly rate 25.50, got %v", retrieved.HourlyRate)
	}
}

func TestUpdateWorkerProfile_UpdateExisting(t *testing.T) {
	mock := &mockWorkerRepo{
		profile: &domain.WorkerProfile{
			UserID:      "user-1",
			Skills:      []string{"Delivery"},
			IsAvailable: false,
		},
	}
	svc := newTestService(nil, mock, nil)

	updated := &domain.WorkerProfile{
		UserID:      "user-1",
		Skills:      []string{"Delivery", "Plumbing"},
		IsAvailable: true,
	}

	err := svc.UpdateWorkerProfile(context.Background(), updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := svc.GetWorkerProfile(context.Background(), "user-1")
	if len(retrieved.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(retrieved.Skills))
	}
	if !retrieved.IsAvailable {
		t.Error("expected IsAvailable to be true")
	}
}

func TestDeleteWorkerProfile_Success(t *testing.T) {
	mock := &mockWorkerRepo{
		profile: &domain.WorkerProfile{
			UserID:      "user-1",
			Skills:      []string{"Delivery"},
			IsAvailable: true,
		},
	}
	svc := newTestService(nil, mock, nil)

	err := svc.DeleteWorkerProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetWorkerProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestDeleteWorkerProfile_NotFound(t *testing.T) {
	svc := newTestService(nil, &mockWorkerRepo{}, nil)

	err := svc.DeleteWorkerProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Profile Photo Tests ────────────────────────────────────────────────────

func TestListPhotos_Empty(t *testing.T) {
	svc := newTestService(nil, nil, &mockPhotoRepo{})

	photos, err := svc.ListPhotos(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(photos) != 0 {
		t.Errorf("expected empty list, got %d photos", len(photos))
	}
}

func TestAddPhoto_Success(t *testing.T) {
	mock := &mockPhotoRepo{}
	svc := newTestService(nil, nil, mock)

	photo := &domain.ProfilePhoto{
		UserID: "user-1",
		S3URL:  "https://s3.example.com/photo1.jpg",
	}

	err := svc.AddPhoto(context.Background(), photo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	photos, _ := svc.ListPhotos(context.Background(), "user-1")
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo, got %d", len(photos))
	}
	if photos[0].S3URL != "https://s3.example.com/photo1.jpg" {
		t.Errorf("unexpected URL: %s", photos[0].S3URL)
	}
}

func TestAddPhoto_Multiple(t *testing.T) {
	mock := &mockPhotoRepo{}
	svc := newTestService(nil, nil, mock)

	svc.AddPhoto(context.Background(), &domain.ProfilePhoto{
		UserID: "user-1",
		S3URL:  "https://s3.example.com/photo1.jpg",
	})
	svc.AddPhoto(context.Background(), &domain.ProfilePhoto{
		UserID: "user-1",
		S3URL:  "https://s3.example.com/photo2.jpg",
	})
	svc.AddPhoto(context.Background(), &domain.ProfilePhoto{
		UserID: "user-2",
		S3URL:  "https://s3.example.com/other.jpg",
	})

	photos, _ := svc.ListPhotos(context.Background(), "user-1")
	if len(photos) != 2 {
		t.Errorf("expected 2 photos for user-1, got %d", len(photos))
	}

	otherPhotos, _ := svc.ListPhotos(context.Background(), "user-2")
	if len(otherPhotos) != 1 {
		t.Errorf("expected 1 photo for user-2, got %d", len(otherPhotos))
	}
}

func TestDeletePhoto_Success(t *testing.T) {
	mock := &mockPhotoRepo{}
	svc := newTestService(nil, nil, mock)

	photo := &domain.ProfilePhoto{UserID: "user-1", S3URL: "https://s3.example.com/photo.jpg"}
	svc.AddPhoto(context.Background(), photo)

	err := svc.DeletePhoto(context.Background(), photo.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	photos, _ := svc.ListPhotos(context.Background(), "user-1")
	if len(photos) != 0 {
		t.Errorf("expected 0 photos after delete, got %d", len(photos))
	}
}

func TestDeletePhoto_NotFound(t *testing.T) {
	svc := newTestService(nil, nil, &mockPhotoRepo{})

	err := svc.DeletePhoto(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSetPrimaryPhoto_Success(t *testing.T) {
	mock := &mockPhotoRepo{}
	svc := newTestService(nil, nil, mock)

	svc.AddPhoto(context.Background(), &domain.ProfilePhoto{UserID: "user-1", S3URL: "https://s3.example.com/p1.jpg"})
	svc.AddPhoto(context.Background(), &domain.ProfilePhoto{UserID: "user-1", S3URL: "https://s3.example.com/p2.jpg"})

	photos, _ := svc.ListPhotos(context.Background(), "user-1")
	firstID := photos[0].ID
	secondID := photos[1].ID

	// Set second photo as primary
	result, err := svc.SetPrimaryPhoto(context.Background(), secondID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != secondID {
		t.Errorf("expected returned photo ID %s, got %s", secondID, result.ID)
	}
	if !result.IsPrimary {
		t.Error("expected returned photo to be primary")
	}

	// Verify primary state
	updated, _ := svc.ListPhotos(context.Background(), "user-1")
	for _, p := range updated {
		if p.ID == secondID && !p.IsPrimary {
			t.Errorf("photo %s should be primary", p.ID)
		}
		if p.ID == firstID && p.IsPrimary {
			t.Errorf("photo %s should not be primary", p.ID)
		}
	}
}

func TestSetPrimaryPhoto_NotFound(t *testing.T) {
	svc := newTestService(nil, nil, &mockPhotoRepo{})

	_, err := svc.SetPrimaryPhoto(context.Background(), "nonexistent", "user-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func intPtr(i int) *int {
	return &i
}
