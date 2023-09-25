package stats

import "time"

// MonthYearLayout represents the layout time.Parse requires.
const MonthYearLayout string = "2006-01"

// Member represents reactions pertaining to a particular member of the slack organization within a given month and year.
type Member struct {
	ID               int64     `json:"id"`
	Date             time.Time `json:"date"`
	SlackUID         string    `json:"slackUID"`
	ReceivedLikes    int64     `json:"receivedLikes"`
	ReceivedDislikes int64     `json:"receivedDislikes"`
}

func NewMember(id int64, date time.Time, slackUID string, receivedLikes int64, receivedDislikes int64) *Member {
	return &Member{
		ID:               id,
		Date:             date,
		SlackUID:         slackUID,
		ReceivedLikes:    receivedLikes,
		ReceivedDislikes: receivedDislikes,
	}
}

// MemberService represents a service for managing a Member.
type MemberService interface {
	// FindMemberByID retrieves a Member by ID.
	// Returns ErrNotFound if the ID does not exist.
	FindMemberByID(id int64) (*Member, error)

	// FindMember retrives a Member by his Slack User ID, and date (month and year).
	// Returns ErrNotFound if no matches found.
	FindMember(SlackUID string, date time.Time) (*Member, error)

	// CreateMember creates a new Member.
	CreateMember(m *Member) error

	// UpdateMember updates a Member.
	UpdateMember(id int64, upd MemberUpdate) error

	// DeleteMember permanently deletes a Member
	DeleteMember(id int64) error
}

// MemberUpdate represents a set of fields to be updated via UpdateMember().
type MemberUpdate struct {
	ReceivedLikes    *int64
	ReceivedDislikes *int64
}
