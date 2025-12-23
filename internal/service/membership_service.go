package service

import (
	"context"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/jackc/pgx/v5"
)

const freeQuotaMonthly = 1000

type MembershipService struct {
	Repo repository.MembershipRepository
}

// GetState returns membership state and refreshes the monthly free quota window if needed.
func (s MembershipService) GetState(ctx context.Context) (*domain.MembershipState, error) {
	tx, err := s.Repo.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	state, err := s.ensureState(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return state, nil
}

// SetUsedQuota sets total used quota (free+topup) and recalculates balances.
func (s MembershipService) SetUsedQuota(ctx context.Context, used int) (*domain.MembershipState, error) {
	tx, err := s.Repo.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	state, err := s.normalizeUsed(ctx, tx, used)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return state, nil
}

// ConsumeWithTx decrements membership quota inside an existing transaction.
func (s MembershipService) ConsumeWithTx(ctx context.Context, tx pgx.Tx, units int) (*domain.MembershipState, error) {
	if units <= 0 {
		return s.ensureState(ctx, tx)
	}
	state, err := s.ensureState(ctx, tx)
	if err != nil {
		return nil, err
	}
	totalTopups, err := s.Repo.SumTopupsWithTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	freeAvail := maxInt(0, freeQuotaMonthly-state.FreeUsed)
	consumeFree := minInt(units, freeAvail)
	remaining := units - consumeFree
	consumeTopup := minInt(remaining, state.TopupBal)

	state.FreeUsed += consumeFree
	state.TopupBal -= consumeTopup
	state.UsedQuota = state.FreeUsed + int(totalTopups-int64(state.TopupBal))

	return s.Repo.SaveStateWithTx(ctx, tx, repository.SaveMembershipStateParams{
		UsedQuota:       state.UsedQuota,
		FreeUsed:        state.FreeUsed,
		FreePeriodStart: state.FreeStart,
		TopupBalance:    state.TopupBal,
	})
}

// normalizeUsed spreads a total used value across free quota then topup balance.
func (s MembershipService) normalizeUsed(ctx context.Context, tx pgx.Tx, used int) (*domain.MembershipState, error) {
	state, err := s.ensureState(ctx, tx)
	if err != nil {
		return nil, err
	}
	totalTopups, err := s.Repo.SumTopupsWithTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	freeUsed := minInt(used, freeQuotaMonthly)
	topupUsed := maxInt(used-freeUsed, 0)
	topupBalance := maxInt64(totalTopups-int64(topupUsed), 0)

	state.FreeUsed = freeUsed
	state.TopupBal = int(topupBalance)
	state.UsedQuota = freeUsed + topupUsed

	return s.Repo.SaveStateWithTx(ctx, tx, repository.SaveMembershipStateParams{
		UsedQuota:       state.UsedQuota,
		FreeUsed:        state.FreeUsed,
		FreePeriodStart: state.FreeStart,
		TopupBalance:    state.TopupBal,
	})
}

// ensureState loads state, resets monthly free quota window if needed, and persists changes.
func (s MembershipService) ensureState(ctx context.Context, tx pgx.Tx) (*domain.MembershipState, error) {
	state, err := s.Repo.GetStateWithTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if state.FreeStart.IsZero() || !sameMonth(state.FreeStart, periodStart) {
		state.FreeStart = periodStart
		state.FreeUsed = 0
	}
	// Recalculate usedQuota to stay consistent.
	totalTopups, err := s.Repo.SumTopupsWithTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	state.UsedQuota = state.FreeUsed + int(totalTopups-int64(state.TopupBal))
	return s.Repo.SaveStateWithTx(ctx, tx, repository.SaveMembershipStateParams{
		UsedQuota:       state.UsedQuota,
		FreeUsed:        state.FreeUsed,
		FreePeriodStart: state.FreeStart,
		TopupBalance:    state.TopupBal,
	})
}

func sameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
