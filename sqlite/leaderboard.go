package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/ddritzenhoff/stats"
	"github.com/ddritzenhoff/stats/sqlite/gen"
)

// Ensure service implements interface.
var _ stats.LeaderboardService = (*LeaderboardService)(nil)

// LeaderboardService represents a service for managing Members.
type LeaderboardService struct {
	query *gen.Queries
	db    *sql.DB
}

// NewLeaderboardService returns a new instance of MemberService.
func NewLeaderboardService(query *gen.Queries, db *sql.DB) *LeaderboardService {
	return &LeaderboardService{
		query: query,
		db:    db,
	}
}

// FindLeaderboard retrives a Leadboard by its date (year and month).
// Returns ErrNotFound if no matches are found.
func (ls *LeaderboardService) FindLeaderboard(date time.Time) (*stats.Leaderboard, error) {
	genMostReceivedLikesMember, err := ls.query.MostLikesReceived(context.TODO(), date.Format(stats.MonthYearLayout))
	if err != nil {
		return nil, err
	}
	mostReceivedLikesMember, err := genMemberToMember(&genMostReceivedLikesMember)
	if err != nil {
		return nil, err
	}

	genMostReceivedDislikesMember, err := ls.query.MostDislikesReceived(context.TODO(), date.Format(stats.MonthYearLayout))
	if err != nil {
		return nil, err
	}
	mostReceivedDislikesMember, err := genMemberToMember(&genMostReceivedDislikesMember)
	if err != nil {
		return nil, err
	}

	return &stats.Leaderboard{
		Date:                       date,
		MostReceivedLikesMember:    *mostReceivedLikesMember,
		MostReceivedDislikesMember: *mostReceivedDislikesMember,
	}, nil
}
