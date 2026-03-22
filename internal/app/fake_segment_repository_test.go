package app_test

import (
	"context"
	"sync"

	"github.com/karo/cuttlegate/internal/domain"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// fakeSegmentRepository is an in-memory implementation of ports.SegmentRepository
// for use in app-layer unit tests. Safe for concurrent access.
type fakeSegmentRepository struct {
	mu      sync.RWMutex
	byID    map[string]*domain.Segment
	members map[string]map[string]struct{}
}

var _ ports.SegmentRepository = (*fakeSegmentRepository)(nil)

func newFakeSegmentRepository() *fakeSegmentRepository {
	return &fakeSegmentRepository{
		byID:    make(map[string]*domain.Segment),
		members: make(map[string]map[string]struct{}),
	}
}

func (r *fakeSegmentRepository) Create(_ context.Context, segment *domain.Segment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.byID {
		if s.ProjectID == segment.ProjectID && s.Slug == segment.Slug {
			return domain.ErrConflict
		}
	}
	cp := *segment
	r.byID[segment.ID] = &cp
	return nil
}

func (r *fakeSegmentRepository) GetBySlug(_ context.Context, projectID, slug string) (*domain.Segment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.byID {
		if s.ProjectID == projectID && s.Slug == slug {
			cp := *s
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeSegmentRepository) List(_ context.Context, projectID string) ([]*domain.Segment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domain.Segment
	for _, s := range r.byID {
		if s.ProjectID == projectID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeSegmentRepository) ListWithCount(_ context.Context, projectID string) ([]*ports.SegmentWithCount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ports.SegmentWithCount
	for _, s := range r.byID {
		if s.ProjectID == projectID {
			cp := *s
			count := len(r.members[s.ID])
			out = append(out, &ports.SegmentWithCount{Segment: &cp, MemberCount: count})
		}
	}
	return out, nil
}

func (r *fakeSegmentRepository) UpdateName(_ context.Context, id, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	s.Name = name
	return nil
}

func (r *fakeSegmentRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.byID, id)
	delete(r.members, id)
	return nil
}

func (r *fakeSegmentRepository) SetMembers(_ context.Context, segmentID string, userKeys []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := make(map[string]struct{}, len(userKeys))
	for _, k := range userKeys {
		set[k] = struct{}{}
	}
	r.members[segmentID] = set
	return nil
}

func (r *fakeSegmentRepository) ListMembers(_ context.Context, segmentID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := r.members[segmentID]
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out, nil
}

func (r *fakeSegmentRepository) IsMember(_ context.Context, segmentID string, userKey string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.members[segmentID][userKey]
	return ok, nil
}
