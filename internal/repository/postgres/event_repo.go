package postgres

import (
	"context"

	"personae-tabula/internal/domain"

	"github.com/uptrace/bun"
)

type EventRepository struct {
	db *bun.DB
}

func NewEventRepository(db *bun.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Сохранить событие в БД
func (r *EventRepository) Save(ctx context.Context, event *domain.TableEvent) error {
	_, err := r.db.NewInsert().Model(event).Exec(ctx)
	return err
}

// Получить фид событий комнаты (текстовый фид)
func (r *EventRepository) GetTableFeed(ctx context.Context, tableID int64, limit, offset int) ([]domain.TableEvent, error) {
	var events []domain.TableEvent
	err := r.db.NewSelect().
		Model(&events).
		Where("table_id = ?", tableID).
		Relation("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	return events, err
}

// Получить фид в текстовом формате
func (r *EventRepository) GetTableFeedText(ctx context.Context, tableID int64, limit int) ([]string, error) {
	events, err := r.GetTableFeed(ctx, tableID, limit, 0)
	if err != nil {
		return nil, err
	}

	var texts []string
	for _, event := range events {
		wsEvent, err := domain.EventFromDBEvent(&event)
		if err != nil {
			continue
		}
		texts = append(texts, wsEvent.FormatText())
	}
	return texts, nil
}
