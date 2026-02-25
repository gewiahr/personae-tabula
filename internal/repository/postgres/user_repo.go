package postgres

import (
	"context"
	"database/sql"

	"personae-tabula/internal/domain"

	"github.com/uptrace/bun"
)

type UserRepository struct {
	db *bun.DB
}

func NewUserRepository(db *bun.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.NewInsert().Model(user).Returning("*").Exec(ctx)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	user := new(domain.User)
	err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := new(domain.User)
	err := r.db.NewSelect().Model(user).Where("username = ?", username).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}
