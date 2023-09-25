package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ddritzenhoff/stats"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	ThumbsUp   = "thumbsup"
	ThumbsDown = "thumbsdown"
)

// Slacker represents a service for handling Slack push events.
type Slacker interface {
	HandleEvents(w http.ResponseWriter, r *http.Request) error
	HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error
}

// Slack represents a service for handling specific Slack events.
type Slack struct {
	// Services used by Slack
	LeaderboardService stats.LeaderboardService
	MemberService      stats.MemberService
	client             *slack.Client

	// Dependencies
	SigningSecret string
	ChannelID     string
}

// NewSlackService creates a new instance of slackService.
func NewSlackService(ms stats.MemberService, ls stats.LeaderboardService, signingSecret string, botSigningKey string, channelID string) (Slacker, error) {
	return &Slack{
		MemberService:      ms,
		LeaderboardService: ls,
		client:             slack.New(botSigningKey),
		SigningSecret:      signingSecret,
		ChannelID:          channelID,
	}, nil
}

// HandleMonthlyUpdate
func (s *Slack) HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	rawDate := r.FormValue("date")
	if rawDate == "" {
		return errors.New("no date value provided within the form")
	}
	date, err := time.Parse(stats.MonthYearLayout, rawDate)
	if err != nil {
		return err
	}

	leaderboard, err := s.LeaderboardService.FindLeaderboard(date)
	if err != nil {
		return err
	}

	var sectionBlocks []slack.Block
	headerText := slack.NewTextBlockObject("mrkdwn", "*Monthly Stats Update*", false, false)
	headerSection := slack.NewHeaderBlock(headerText)
	sectionBlocks = append(sectionBlocks, headerSection)

	mostLikesReceivedMembers := fmt.Sprintf("Most likes received this month (aka good boy of the month): <@%s> (%d)", leaderboard.MostReceivedLikesMember.SlackUID, leaderboard.MostReceivedLikesMember.ReceivedLikes)
	sectionText := slack.NewTextBlockObject("mrkdwn", mostLikesReceivedMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostDislikesReceivedMembers := fmt.Sprintf("Most dislikes received this month: <@%s> (%d)", leaderboard.MostReceivedDislikesMember.SlackUID, leaderboard.MostReceivedDislikesMember.ReceivedDislikes)
	sectionText = slack.NewTextBlockObject("mrkdwn", mostDislikesReceivedMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	msg := slack.MsgOptionBlocks(sectionBlocks...)
	_, _, err = s.client.PostMessage(s.ChannelID, msg)
	if err != nil {
		return fmt.Errorf("WeeklyUpdate PostMessage: %w", err)
	}
	return nil
}

// handleEvents handles Slack push events.
func (s *Slack) HandleEvents(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	sv, err := slack.NewSecretsVerifier(r.Header, s.SigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("HandleEvents: %w", err)
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("HandleEvents: %w", err)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.ReactionAddedEvent:
			err := s.HandleReactionAddedEvent(ev)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return fmt.Errorf("HandleEvents: %w", err)
			}
		case *slackevents.ReactionRemovedEvent:
			err := s.HandleReactionRemovedEvent(ev)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return fmt.Errorf("HandleEvents: %w", err)
			}
		}
	}
	return nil
}

// HandleReactionAddedEvent handles the event when a user reacts to the post of another user.
func (s *Slack) HandleReactionAddedEvent(e *slackevents.ReactionAddedEvent) error {

	// If the user is reacting to his own message, do nothing.
	if e.User == e.ItemUser {
		return nil
	}

	date := time.Now().UTC()

	// Create the member (user being reacted to) if he does not already exist within the database.
	itemMember, err := s.MemberService.FindMember(e.ItemUser, date)
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date:     date,
			})
			if err != nil {
				return fmt.Errorf("HandleReactionAddedEvent CreateMember itemMember: %w", err)
			}
			return s.HandleReactionAddedEvent(e)
		}
		return fmt.Errorf("HandleReactionAddedEvent FindMember ItemUser: %w", err)
	}

	// Update the reactions.
	if e.Reaction == ThumbsUp {
		itemMember.ReceivedLikes += 1
	} else if e.Reaction == ThumbsDown {
		itemMember.ReceivedDislikes += 1
	}

	// Update the stats of the User being reacted to.
	s.MemberService.UpdateMember(itemMember.ID, stats.MemberUpdate{
		ReceivedLikes:    &itemMember.ReceivedLikes,
		ReceivedDislikes: &itemMember.ReceivedDislikes,
	})
	return nil
}

// max finds the max between two int64s and returns it.
func max(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// HandleReactionRemovedEvent handles the event when a user removes a reaction from another user's post.
func (s *Slack) HandleReactionRemovedEvent(e *slackevents.ReactionRemovedEvent) error {
	date := time.Now().UTC()

	// If the user is reacting to his own message, do nothing.
	if e.User == e.ItemUser {
		return nil
	}

	// Create the member (user being reacted to) if he does not already exist within the database.
	itemMember, err := s.MemberService.FindMember(e.ItemUser, date)
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date:     date,
			})
			if err != nil {
				return fmt.Errorf("HandleReactionRemovedEvent CreateMember itemMember: %w", err)
			}
			return s.HandleReactionRemovedEvent(e)
		}
		return fmt.Errorf("HandleReactionRemovedEvent FindMember ItemUser: %w", err)
	}

	// Update the reactions.
	if e.Reaction == ThumbsUp {
		itemMember.ReceivedLikes = max(itemMember.ReceivedLikes-1, 0)
	} else if e.Reaction == ThumbsDown {
		itemMember.ReceivedDislikes = max(itemMember.ReceivedDislikes-1, 0)
	}

	// Update the stats of the User being reacted to.
	s.MemberService.UpdateMember(itemMember.ID, stats.MemberUpdate{
		ReceivedLikes:    &itemMember.ReceivedLikes,
		ReceivedDislikes: &itemMember.ReceivedDislikes,
	})
	return nil
}
