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
	query *gen.Queries
	db    *sql.DB
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(query *gen.Queries, db *sql.DB) *MemberService {
	return &MemberService{
		query: query,
		db:    db,
	}
}

// FindMemberByID retrieves a Member by ID.
// Returns ErrNotFound if the ID does not exist.
func (ms *MemberService) FindMemberByID(id int64) (*stats.Member, error) {
	genMember, err := ms.query.FindMemberByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, stats.ErrNotFound
		}
		return nil, err
	}

	d, err := time.Parse(stats.MonthYearLayout, genMember.MonthYear)
	if err != nil {
		return nil, err
	}

	return &stats.Member{
		ID:               genMember.ID,
		SlackUID:         genMember.SlackUid,
		ReceivedLikes:    genMember.ReceivedLikes,
		ReceivedDislikes: genMember.ReceivedDislikes,
		Date:             d,
	}, nil
}

// FindMember retrives a Member by his Slack User ID, the Month, and the Year.
// Returns ErrNotFound if not matches found.
func (ms *MemberService) FindMember(SlackUID string, date time.Time) (*stats.Member, error) {

	genMember, err := ms.query.FindMember(context.TODO(), gen.FindMemberParams{
		SlackUid:  SlackUID,
		MonthYear: date.Format(stats.MonthYearLayout),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, stats.ErrNotFound
		}
		return nil, err
	}

	return &stats.Member{
		ID:               genMember.ID,
		SlackUID:         genMember.SlackUid,
		ReceivedLikes:    genMember.ReceivedLikes,
		ReceivedDislikes: genMember.ReceivedDislikes,
		Date:             date,
	}, nil
}

// CreateMember creates a new Member.
func (ms *MemberService) CreateMember(m *stats.Member) error {
	if m == nil {
		return fmt.Errorf("CreateMember: m reference is nil")
	}

	_, err := ms.query.CreateMember(context.TODO(), gen.CreateMemberParams{
		SlackUid:  m.SlackUID,
		MonthYear: m.Date.Format(stats.MonthYearLayout),
	})
	if err != nil {
		return fmt.Errorf("CreateMember: %w", err)
	}
	return nil
}

// UpdateMember updates a Member.
func (ms *MemberService) UpdateMember(id int64, upd stats.MemberUpdate) error {
	tx, err := ms.db.Begin()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember db.Begin: %w", err)
	}

	genMember, err := ms.query.FindMemberByID(context.TODO(), id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember FindMemberByID: %w", err)
	}

	if upd.ReceivedLikes != nil {
		genMember.ReceivedLikes = *upd.ReceivedLikes
	}
	if upd.ReceivedDislikes != nil {
		genMember.ReceivedDislikes = *upd.ReceivedDislikes
	}
	err = ms.query.UpdateMember(context.TODO(), gen.UpdateMemberParams{
		ReceivedLikes:    genMember.ReceivedLikes,
		ReceivedDislikes: genMember.ReceivedDislikes,
		ID:               id,
	})
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember: %w", err)
	}

	tx.Commit()
	return nil
}

// DeleteMember permanently deletes a Member
func (ms *MemberService) DeleteMember(id int64) error {
	err := ms.query.DeleteMember(context.TODO(), id)
	if err != nil {
		return fmt.Errorf("DeleteMember: %w", err)
	}
	return nil
}

func genMemberToMember(mem *gen.Member) (*stats.Member, error) {
	date, err := time.Parse(stats.MonthYearLayout, mem.MonthYear)
	if err != nil {
		return nil, err
	}
	return stats.NewMember(mem.ID, date, mem.SlackUid, mem.ReceivedLikes, mem.ReceivedDislikes), nil
}
