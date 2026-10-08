package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/replies"
)

// ReplyService reads reviews with their replies and writes replies.
// *replies.Service satisfies it.
type ReplyService interface {
	Get(ctx context.Context, user auth.User, reviewID int64) (replies.Detail, error)
	Draft(ctx context.Context, user auth.User, reviewID int64) (*replies.Reply, bool, error)
	Save(ctx context.Context, user auth.User, reviewID int64, text string, seen *time.Time) (*replies.Reply, error)
	MarkReplied(ctx context.Context, user auth.User, reviewID int64, text string, seen *time.Time) (*replies.Reply, error)
}

// replyJSON, replySave and reviewDetail are Reply, ReplySave and
// ReviewDetail in api/openapi.yaml.
type replyJSON struct {
	Status        string     `json:"status"`
	DraftText     *string    `json:"draft_text"`
	PromptVersion *int       `json:"prompt_version"`
	ReplyText     *string    `json:"reply_text"`
	Edited        bool       `json:"edited"`
	RepliedBy     *userRef   `json:"replied_by"`
	RepliedAt     *time.Time `json:"replied_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type replySave struct {
	ReplyText        string     `json:"reply_text"`
	BasedOnUpdatedAt *time.Time `json:"based_on_updated_at"`
}

type reviewDetail struct {
	reviewSummary
	Reply          *replyJSON `json:"reply"`
	OutletManagers []userRef  `json:"outlet_managers"`
	CanReply       bool       `json:"can_reply"`
}

func newReplyJSON(r *replies.Reply) *replyJSON {
	if r == nil {
		return nil
	}
	out := &replyJSON{Status: r.Status, DraftText: r.DraftText, PromptVersion: r.PromptVersion, ReplyText: r.ReplyText,
		Edited: r.Edited(), RepliedAt: r.RepliedAt, UpdatedAt: r.UpdatedAt.UTC()}
	if r.RepliedBy != nil {
		out.RepliedBy = &userRef{ID: r.RepliedBy.ID, Name: r.RepliedBy.Name}
	}
	return out
}

// reviewID reads the path's review_id; anything but a positive integer is 400.
func reviewID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("review_id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &malformedError{reason: "review_id must be a positive integer"}
	}
	return id, nil
}

// getReview answers GET /api/v1/reviews/{review_id} (operationId getReview).
func getReview(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		id, err := reviewID(r)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		det, err := d.Replies.Get(r.Context(), user, id)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		managers := make([]userRef, len(det.Managers))
		for i, m := range det.Managers {
			managers[i] = userRef{ID: m.ID, Name: m.Name}
		}
		WriteJSON(w, http.StatusOK, reviewDetail{reviewSummary: newReviewSummary(det.Review),
			Reply: newReplyJSON(det.Reply), OutletManagers: managers, CanReply: det.CanReply})
	})
}

// createReplyDraft answers POST /api/v1/reviews/{review_id}/draft.
func createReplyDraft(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		id, err := reviewID(r)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		reply, drafting, err := d.Replies.Draft(r.Context(), user, id)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		if drafting {
			w.Header().Set("Retry-After", "2")
			WriteJSON(w, http.StatusAccepted, newReplyJSON(reply))
			return
		}
		WriteJSON(w, http.StatusOK, newReplyJSON(reply))
	})
}

// replyWrite answers PUT .../reply (saveReply) and POST .../replied
// (markReplied); the role is checked before the body is read.
func replyWrite(d Deps, mark bool) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		id, err := reviewID(r)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		if user.Role != auth.RoleOutletManager {
			writeDomainError(w, r, d.Logger, replies.ErrRoleNotAllowed)
			return
		}
		var body replySave
		if err := decodeJSON(w, r, &body); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		var reply *replies.Reply
		if mark {
			reply, err = d.Replies.MarkReplied(r.Context(), user, id, body.ReplyText, body.BasedOnUpdatedAt)
		} else {
			reply, err = d.Replies.Save(r.Context(), user, id, body.ReplyText, body.BasedOnUpdatedAt)
		}
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		WriteJSON(w, http.StatusOK, newReplyJSON(reply))
	})
}
