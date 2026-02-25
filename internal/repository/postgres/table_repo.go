package postgres

import (
	"context"
	"database/sql"

	"personae-tabula/internal/domain"

	"github.com/uptrace/bun"
)

type TableRepository struct {
	db *bun.DB
}

func NewTableRepository(db *bun.DB) *TableRepository {
	return &TableRepository{db: db}
}

func (r *TableRepository) Create(ctx context.Context, table *domain.Table) error {
	_, err := r.db.NewInsert().Model(table).Returning("*").Exec(ctx)
	return err
}

func (r *TableRepository) GetByID(ctx context.Context, id int64) (*domain.Table, error) {
	table := new(domain.Table)
	err := r.db.NewSelect().
		Model(table).
		Relation("Creator").
		Where("t.id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return table, err
}

func (r *TableRepository) List(ctx context.Context, limit, offset int) ([]domain.Table, error) {
	var tables []domain.Table
	err := r.db.NewSelect().
		Model(&tables).
		Relation("Creator").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	return tables, err
}

func (r *TableRepository) Update(ctx context.Context, table *domain.Table) error {
	_, err := r.db.NewUpdate().Model(table).WherePK().Exec(ctx)
	return err
}

func (r *TableRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&domain.Table{}).Where("id = ?", id).Exec(ctx)
	return err
}
