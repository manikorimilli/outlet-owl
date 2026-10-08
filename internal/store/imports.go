package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/imports"
)

// The import service stores imports through the store.
var _ imports.Store = (*Store)(nil)

// ImportByRequestID returns the import stored under the request id, with its
// rejected rows; ok is false when there is none.
func (s *Store) ImportByRequestID(ctx context.Context, requestID string) (imports.Result, bool, error) {
	return importByRequestID(ctx, New(s.Pool), requestID)
}

func importByRequestID(ctx context.Context, q *Queries, requestID string) (imports.Result, bool, error) {
	r, err := q.GetImportByRequestID(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return imports.Result{}, false, nil
	}
	if err != nil {
		return imports.Result{}, false, fmt.Errorf("query import: %w", err)
	}
	rows, err := q.ListImportRejections(ctx, r.ID)
	if err != nil {
		return imports.Result{}, false, fmt.Errorf("query import rejections: %w", err)
	}
	rej := make([]connector.Rejection, len(rows))
	for i, row := range rows {
		rej[i] = connector.Rejection{Number: int(row.RowNumber), Reason: row.Reason}
	}
	return imports.Result{
		ID: r.ID, FileName: r.FileName, CreatedAt: r.CreatedAt,
		ImportedCount: int(r.ImportedCount), DuplicateCount: int(r.DuplicateCount), RejectedCount: int(r.RejectedCount),
		Rejections: rej,
	}, true, nil
}

// OutletIDsByName maps each outlet's name, as imports.OutletKey folds it, to
// its id.
func (s *Store) OutletIDsByName(ctx context.Context) (map[string]int64, error) {
	rows, err := New(s.Pool).ListOutletNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("query outlet names: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[imports.OutletKey(r.Name)] = r.ID
	}
	return out, nil
}

// SaveImport writes the import, its new reviews and its rejected rows in one
// transaction (data model section 5). When the request id is already stored,
// including by an import that committed while this one waited on it, it
// writes nothing and returns that import with created false.
func (s *Store) SaveImport(ctx context.Context, w imports.Write) (imports.Result, bool, error) {
	var res imports.Result
	created := false
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := New(s.Pool).WithTx(tx)
		row, err := q.CreateImport(ctx, CreateImportParams{RequestID: w.RequestID, FileName: w.FileName})
		if errors.Is(err, pgx.ErrNoRows) {
			return errAlreadyImported
		}
		if err != nil {
			return fmt.Errorf("insert import: %w", err)
		}

		p := InsertReviewsParams{ImportID: row.ID}
		for _, r := range w.Reviews {
			p.OutletIds = append(p.OutletIds, r.OutletID)
			p.Sources = append(p.Sources, r.Source)
			p.ReviewDates = append(p.ReviewDates, pgtype.Date{Time: r.Date, Valid: true})
			p.Ratings = append(p.Ratings, int16(r.Rating))
			p.ReviewTexts = append(p.ReviewTexts, r.Text)
			p.ReviewerNames = append(p.ReviewerNames, r.ReviewerName)
		}
		ids, err := q.InsertReviews(ctx, p)
		if err != nil {
			return fmt.Errorf("insert reviews: %w", err)
		}

		rp := InsertImportRejectionsParams{ImportID: row.ID}
		for _, r := range w.Rejections {
			rp.RowNumbers = append(rp.RowNumbers, int32(r.Number))
			rp.Reasons = append(rp.Reasons, r.Reason)
		}
		if len(w.Rejections) > 0 {
			if err := q.InsertImportRejections(ctx, rp); err != nil {
				return fmt.Errorf("insert import rejections: %w", err)
			}
		}

		res = imports.Result{
			ID: row.ID, FileName: w.FileName, CreatedAt: row.CreatedAt,
			ImportedCount:  len(ids),
			DuplicateCount: len(w.Reviews) - len(ids),
			RejectedCount:  len(w.Rejections),
			Rejections:     w.Rejections,
		}
		if err := q.SetImportCounts(ctx, SetImportCountsParams{
			ID: row.ID, ImportedCount: int32(res.ImportedCount),
			DuplicateCount: int32(res.DuplicateCount), RejectedCount: int32(res.RejectedCount),
		}); err != nil {
			return fmt.Errorf("set import counts: %w", err)
		}
		created = true
		return nil
	})
	if errors.Is(err, errAlreadyImported) {
		prev, ok, err := s.ImportByRequestID(ctx, w.RequestID)
		if err != nil {
			return imports.Result{}, false, err
		}
		if !ok {
			return imports.Result{}, false, fmt.Errorf("import %s: the conflicting import is gone", w.RequestID)
		}
		return prev, false, nil
	}
	if err != nil {
		return imports.Result{}, false, err
	}
	if res.Rejections == nil {
		res.Rejections = []connector.Rejection{}
	}
	return res, created, nil
}

// errAlreadyImported rolls back an import whose request id is taken.
var errAlreadyImported = errors.New("store: the request id is already imported")
