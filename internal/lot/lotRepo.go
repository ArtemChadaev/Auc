package lot

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

func (r *Repo) WithTx(tx storage.DBTX) *Repo {
	return &Repo{db: tx}
}

func (r *Repo) create(
	ctx context.Context,
	sellerID uuid.UUID,
	itemID int64,
	status Status,
	startAmount int64,
	minStep int64,
	title string,
	description Description,
	startedAt *time.Time,
	endedAt *time.Time,
) (Lot, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description, started_at, ends_at) 
		SELECT $1, id, $3, $4, $5, $6, $7, $8, $9
		FROM items
		WHERE id = $2 AND owner_id = $1
		RETURNING *`,
		sellerID, itemID, status, startAmount, minStep, title, description, startedAt, endedAt,
	)
	if err != nil {
		return Lot{}, fmt.Errorf("lotRepo.create: %w", storage.ErrRepo(err))
	}
	lot, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Lot])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if res, _ := r.db.Exec(ctx, "SELECT 1 FROM items WHERE id = $1 AND owner_id = $2", itemID, sellerID); res.RowsAffected() == 0 {
				return Lot{}, fmt.Errorf("lotRepo.create: %w", storage.ErrForbidden)
			}
		}
		return Lot{}, fmt.Errorf("lotRepo.create: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) updateFromSeller(
	ctx context.Context,
	id int64,
	sellerID uuid.UUID,
	itemID int64,
	status Status,
	startAmount int64,
	minStep int64,
	title string,
	description Description,
	startedAt *time.Time,
	endedAt *time.Time,
) (Lot, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE lots 
		SET item_id = i.id, status = $4, start_amount = $5, min_step = $6, title = $7, description = $8, started_at = $9, ends_at = $10
		FROM items i
		WHERE lots.id = $1 AND lots.seller_id = $2 AND i.id = $3 AND i.owner_id = $2
		RETURNING lots.*`,
		id, sellerID, itemID, status, startAmount, minStep, title, description, startedAt, endedAt,
	)
	if err != nil {
		return Lot{}, fmt.Errorf("lotRepo.updateFromSeller: %w", storage.ErrRepo(err))
	}
	lot, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Lot])
	if err != nil {
		return Lot{}, fmt.Errorf("lotRepo.updateFromSeller: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) activate(ctx context.Context, id int64, sellerID uuid.UUID, startedAt, endedAt *time.Time) (Lot, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE lots 
		SET status = $1,
		    started_at = COALESCE($4, started_at, now()),
		    ends_at = COALESCE($5, ends_at)
		WHERE id = $2 AND seller_id = $3 AND status = $6
		RETURNING *`,
		Active, id, sellerID, startedAt, endedAt, Approved,
	)
	if err != nil {
		return Lot{}, fmt.Errorf("lotRepo.activate: %w", storage.ErrRepo(err))
	}
	lot, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Lot])
	if err != nil {
		var currentStatus Status
		var currentSeller uuid.UUID
		checkErr := r.db.QueryRow(ctx, "SELECT status, seller_id FROM lots WHERE id = $1", id).Scan(&currentStatus, &currentSeller)
		if checkErr == nil {
			if currentSeller != sellerID {
				return Lot{}, fmt.Errorf("lotRepo.activate: %w", ErrForbidden)
			}
			if currentStatus != Approved {
				return Lot{}, fmt.Errorf("lotRepo.activate: %w", ErrLotNotApproved)
			}
		}
		return Lot{}, fmt.Errorf("lotRepo.activate: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) updateAfterFinish(ctx context.Context, id int64, winnerID uuid.UUID, finishAmount int64) error {
	res, err := r.db.Exec(ctx, `
		UPDATE lots 
		SET winner_id = $1, finish_amount = $2
		WHERE id = $3`,
		winnerID, finishAmount, id,
	)
	if err != nil {
		return fmt.Errorf("lotRepo.updateAfterFinish: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.updateAfterFinish: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) getById(ctx context.Context, id int64) (Lot, error) {
	var lot Lot
	row, err := r.db.Query(ctx, "SELECT * FROM lots WHERE id = $1 AND status IN ($2, $3, $4, $5)", id, Approved, Active, Finished, Cancelled)
	if err != nil {
		return lot, fmt.Errorf("lotRepo.getById: %w", storage.ErrRepo(err))
	}
	lot, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Lot])
	if err != nil {
		return lot, fmt.Errorf("lotRepo.getById: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) getMy(ctx context.Context, sellerID uuid.UUID) ([]Lot, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM lots WHERE seller_id = $1 ORDER BY id DESC", sellerID)
	if err != nil {
		return nil, fmt.Errorf("lotRepo.getMy: %w", storage.ErrRepo(err))
	}
	lots, err := pgx.CollectRows(rows, pgx.RowToStructByName[Lot])
	if err != nil {
		return nil, fmt.Errorf("lotRepo.getMy: %w", storage.ErrRepo(err))
	}
	if lots == nil {
		lots = []Lot{}
	}
	return lots, nil
}

func (r *Repo) list(ctx context.Context) ([]Lot, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM lots WHERE status IN ($1, $2) ORDER BY id DESC", Active, Finished)
	if err != nil {
		return nil, fmt.Errorf("lotRepo.list: %w", storage.ErrRepo(err))
	}
	lots, err := pgx.CollectRows(rows, pgx.RowToStructByName[Lot])
	if err != nil {
		return nil, fmt.Errorf("lotRepo.list: %w", storage.ErrRepo(err))
	}
	if lots == nil {
		lots = []Lot{}
	}
	return lots, nil
}

func (r *Repo) cancel(ctx context.Context, id int64, sellerId uuid.UUID) error {
	res, err := r.db.Exec(ctx, "UPDATE lots SET status = $3 WHERE id = $1 AND seller_id = $2 AND (status NOT IN ($4, $5) OR status = $5 AND started_at > now() + INTERVAL '1 hour')", id, sellerId, Cancelled, Finished, Active)
	if err != nil {
		return fmt.Errorf("lotRepo.cancel: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.cancel: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) getByIdForUpdate(ctx context.Context, id int64) (Lot, error) {
	var lot Lot
	row, err := r.db.Query(ctx, `
		SELECT id, seller_id, item_id, status, winner_id, start_amount, min_step, finish_amount, title, description, created_at, started_at, ends_at, finished_at 
		FROM lots 
		WHERE id = $1 AND status = 'active'
		FOR UPDATE`, id)
	if err != nil {
		return lot, fmt.Errorf("lotRepo.getByIdForUpdate: %w", storage.ErrRepo(err))
	}
	lot, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Lot])
	if err != nil {
		return lot, fmt.Errorf("lotRepo.getByIdForUpdate: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) createBid(ctx context.Context, idLot int64, uid uuid.UUID, amount int) error {
	_, err := r.db.Exec(ctx, "INSERT INTO bids (lot_id, user_id, amount) VALUES ($1, $2, $3)", idLot, uid, amount)
	if err != nil {
		return fmt.Errorf("lotRepo.createBid: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) getLastBid(ctx context.Context, idLot int64) (Bid, error) {
	row, err := r.db.Query(ctx, `
		SELECT id, lot_id, user_id, amount, created_at 
		FROM bids 
		WHERE lot_id = $1 
		ORDER BY id DESC 
		LIMIT 1`, idLot)
	if err != nil {
		return Bid{}, fmt.Errorf("lotRepo.getLastBid: %w", storage.ErrRepo(err))
	}
	bid, err := pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Bid])
	if err != nil {
		return Bid{}, fmt.Errorf("lotRepo.getLastBid: %w", storage.ErrRepo(err))
	}
	return bid, nil
}

func (r *Repo) getLotBids(ctx context.Context, idLot int64) ([]Bid, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, lot_id, user_id, amount, created_at
		FROM bids
		WHERE lot_id = $1
		ORDER BY id DESC`, idLot)
	if err != nil {
		return nil, fmt.Errorf("lotRepo.getLotBids: %w", storage.ErrRepo(err))
	}
	bids, err := pgx.CollectRows(rows, pgx.RowToStructByName[Bid])
	if err != nil {
		return nil, fmt.Errorf("lotRepo.getLotBids: %w", storage.ErrRepo(err))
	}
	if bids == nil {
		bids = []Bid{}
	}
	return bids, nil
}

func (r *Repo) getUserForUpdate(ctx context.Context, uid uuid.UUID) (user.User, error) {
	var u user.User
	row := r.db.QueryRow(ctx, "SELECT id, name, avatar_id, email, deleted_at, balance, hold, created_at FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE", uid)
	err := row.Scan(&u.ID, &u.Name, &u.AvatarID, &u.Email, &u.DeletedAt, &u.Balance, &u.Hold, &u.CreatedAt)
	if err != nil {
		return u, fmt.Errorf("userRepo.getUser: %w", storage.ErrRepo(err))
	}
	return u, nil
}

// changeHold передается контекст, uid и изменение changeHold тоесть увеличить на 10 (changeHold=10) или уменьшить на 20 (changeHold=-20)
func (r *Repo) changeHold(ctx context.Context, uid uuid.UUID, changeHold int) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET hold = hold + $1 WHERE id = $2 AND deleted_at IS NULL", changeHold, uid)
	if err != nil {
		return fmt.Errorf("lotRepo.changeHold: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.changeHold: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) approve(ctx context.Context, id int64) error {
	res, err := r.db.Exec(ctx, `
		UPDATE lots
		SET status = CASE
			WHEN ((description->>'auto_activate')::boolean = true)
			     AND started_at IS NOT NULL AND ends_at IS NOT NULL AND started_at > now() + INTERVAL '1 hour'
			THEN 'active'::text
			ELSE 'approved'::text
		END
		WHERE id = $1 AND status = $2`,
		id, PendingReview,
	)
	if err != nil {
		return fmt.Errorf("lotRepo.approve: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.approve: %w", storage.ErrNotFound)
	}
	return nil
}
