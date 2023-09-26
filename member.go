package stats

import "time"

// MonthYear represents a string with the following format: <year>-<month>.
// i.e. `2024-02` represents February 2024.
type MonthYear string

const monthYearLayout string = "2006-01"

// NewMonthYear returns a new instance of MonthYear.
func NewMonthYear(t time.Time) MonthYear {
	return MonthYear(t.UTC().Format(monthYearLayout))
}

// NewMonthYearString returns a new instance of MonthYear.
func NewMonthYearString(s string) (MonthYear, error) {
	t, err := time.Parse(monthYearLayout, s)
	if err != nil {
		return "", err
	}
	return NewMonthYear(t), nil
}

// String returns the string representation of MonthYear.
func (my *MonthYear) String() string {
	return string(*my)
}

// Member represents reactions pertaining to a particular member of the slack organization within a given month and year.
type Member struct {
	ID               int64     `json:"id"`
	Date             MonthYear `json:"date"`
	SlackUID         string    `json:"slackUID"`
	ReceivedLikes    int64     `json:"receivedLikes"`
	ReceivedDislikes int64     `json:"receivedDislikes"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `jons:"updatedAt"`
}

// NewMember returns a new instance of Member.
func NewMember(id int64, date MonthYear, slackUID string, receivedLikes int64, receivedDislikes int64, createdAt time.Time, updatedAt time.Time) *Member {
	return &Member{
		ID:               id,
		Date:             date,
		SlackUID:         slackUID,
		ReceivedLikes:    receivedLikes,
		ReceivedDislikes: receivedDislikes,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

// MemberService represents a service for managing a Member.
type MemberService interface {
	// FindMemberByID retrieves a Member by ID.
	// Returns ErrNotFound if the ID does not exist.
	FindMemberByID(id int64) (*Member, error)

	// FindMember retrives a Member by his Slack User ID, and date (month and year).
	// Returns ErrNotFound if no matches found.
	FindMember(SlackUID string, date MonthYear) (*Member, error)

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
