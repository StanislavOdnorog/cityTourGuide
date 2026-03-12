package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

type mockCleanupRepo struct {
	completed []int
	failed    []struct {
		id  int
		err string
	}
}

func (m *mockCleanupRepo) ClaimBatch(context.Context, int) ([]domain.OrphanCleanupJob, error) {
	return nil, nil
}

func (m *mockCleanupRepo) MarkCompleted(_ context.Context, id int) error {
	m.completed = append(m.completed, id)
	return nil
}

func (m *mockCleanupRepo) MarkFailed(_ context.Context, id int, errMsg string) error {
	m.failed = append(m.failed, struct {
		id  int
		err string
	}{id: id, err: errMsg})
	return nil
}

func (m *mockCleanupRepo) ReclaimFailed(context.Context, int) (int, error) {
	return 0, nil
}

type mockObjectDeleter struct {
	deleteErr error
	exists    bool
	existsErr error
}

func (m *mockObjectDeleter) Delete(context.Context, string) error {
	return m.deleteErr
}

func (m *mockObjectDeleter) Exists(context.Context, string) (bool, error) {
	return m.exists, m.existsErr
}

func TestCleanupWorkerProcessJobTreatsMissingObjectAsSuccess(t *testing.T) {
	repo := &mockCleanupRepo{}
	storage := &mockObjectDeleter{
		deleteErr: errors.New("not found"),
		exists:    false,
	}
	w := NewCleanupWorker(repo, storage, nil)

	w.processJob(context.Background(), domain.OrphanCleanupJob{ID: 7, ObjectKey: "audio/1/2/3.mp3", Attempts: 1})

	if len(repo.completed) != 1 || repo.completed[0] != 7 {
		t.Fatalf("completed jobs = %v, want [7]", repo.completed)
	}
	if len(repo.failed) != 0 {
		t.Fatalf("failed jobs = %v, want none", repo.failed)
	}
}

func TestCleanupWorkerProcessJobMarksFailureForRetryableDeleteError(t *testing.T) {
	repo := &mockCleanupRepo{}
	storage := &mockObjectDeleter{
		deleteErr: errors.New("storage unavailable"),
		exists:    true,
	}
	w := NewCleanupWorker(repo, storage, nil)

	w.processJob(context.Background(), domain.OrphanCleanupJob{ID: 9, ObjectKey: "audio/1/2/4.mp3", Attempts: 2})

	if len(repo.completed) != 0 {
		t.Fatalf("completed jobs = %v, want none", repo.completed)
	}
	if len(repo.failed) != 1 || repo.failed[0].id != 9 {
		t.Fatalf("failed jobs = %v, want one failure for job 9", repo.failed)
	}
}
