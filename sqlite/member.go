package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ddritzenhoff/stats"
	"github.com/ddritzenhoff/stats/sqlite/gen"
)

// Ensure service implements interface.
var _ stats.MemberService = (*MemberService)(nil)

// MemberService represents a service for managing Members.
type MemberService struct {
	db *DB
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(db *DB) *MemberService {
	return &MemberService{
		db: db,
	}
}

// FindMemberByID retrieves a Member by ID.
// Returns ErrNotFound if the ID does not exist.
func (ms *MemberService) FindMemberByID(id int64) (*stats.Member, error) {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// fetch member
	genMember, err := ms.db.query.FindMemberByID(context.TODO(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, stats.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return genMemberToMember(&genMember)
}

// FindMember retrives a Member by his Slack User ID, the Month, and the Year.
// Returns ErrNotFound if not matches found.
func (ms *MemberService) FindMember(SlackUID string, date stats.MonthYear) (*stats.Member, error) {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	genMember, err := ms.db.query.FindMember(context.TODO(), gen.FindMemberParams{
		SlackUid:  SlackUID,
		MonthYear: date.String(),
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, stats.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return genMemberToMember(&genMember)
}

// CreateMember creates a new Member.
func (ms *MemberService) CreateMember(m *stats.Member) error {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if m == nil {
		return fmt.Errorf("CreateMember: m reference is nil")
	}

	m.CreatedAt = tx.now
	m.UpdatedAt = m.CreatedAt

	genMem, err := ms.db.query.CreateMember(context.TODO(), gen.CreateMemberParams{
		SlackUid:  m.SlackUID,
		MonthYear: m.Date.String(),
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("CreateMember: %w", err)
	}

	m.ID = genMem.ID
	m.ReceivedDislikes = genMem.ReceivedDislikes
	m.ReceivedLikes = genMem.ReceivedLikes

	return tx.Commit()
}

// UpdateMember updates a Member.
func (ms *MemberService) UpdateMember(id int64, upd stats.MemberUpdate) error {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("UpdateMember db.Begin: %w", err)
	}
	defer tx.Rollback()

	genMember, err := ms.db.query.FindMemberByID(context.TODO(), id)
	if err != nil {
		return fmt.Errorf("UpdateMember FindMemberByID: %w", err)
	}

	if upd.ReceivedLikes != nil {
		genMember.ReceivedLikes = *upd.ReceivedLikes
	}
	if upd.ReceivedDislikes != nil {
		genMember.ReceivedDislikes = *upd.ReceivedDislikes
	}
	err = ms.db.query.UpdateMember(context.TODO(), gen.UpdateMemberParams{
		ReceivedLikes:    genMember.ReceivedLikes,
		ReceivedDislikes: genMember.ReceivedDislikes,
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		ID:               id,
	})
	if err != nil {
		return fmt.Errorf("UpdateMember: %w", err)
	}

	return tx.Commit()
}

// DeleteMember permanently deletes a Member.
func (ms *MemberService) DeleteMember(id int64) error {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = ms.db.query.DeleteMember(context.TODO(), id)
	if err != nil {
		return fmt.Errorf("DeleteMember: %w", err)
	}
	return nil
}

// genMemberToMember converts the sqlite member type to the stats member type.
func genMemberToMember(mem *gen.Member) (*stats.Member, error) {
	date, err := stats.NewMonthYearString(mem.MonthYear)
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339, mem.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := time.Parse(time.RFC3339, mem.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return stats.NewMember(mem.ID, date, mem.SlackUid, mem.ReceivedLikes, mem.ReceivedDislikes, createdAt, updatedAt), nil
}
