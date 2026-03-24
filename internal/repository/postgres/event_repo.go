package postgres

import (
	"context"
	"time"

	"personae-tabula/internal/domain"

	"github.com/uptrace/bun"
)

type EventRepository struct {
	db *bun.DB
}

func NewEventRepository(db *bun.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) SaveEvent(ctx context.Context, userID, tableID, timestamp int64, eventType, content string) (*domain.TableEvent, error) {
	event := &domain.TableEvent{
		TableID:   tableID,
		EventType: eventType,
		UserID:    userID,
		CreatedAt: time.Now(),
		Content:   content,
	}

	err := r.Save(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// Сохранить событие в БД
func (r *EventRepository) Save(ctx context.Context, event *domain.TableEvent) error {
	_, err := r.db.NewInsert().Model(event).Returning("*").Exec(ctx, event)
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
func (r *EventRepository) GetTableFeedText(ctx context.Context, tableID int64, limit int) ([]domain.TableEvent, error) {
	events, err := r.GetTableFeed(ctx, tableID, limit, 0)
	if err != nil {
		return nil, err
	}

	return events, nil

	// var texts []string
	// for _, event := range events {
	// 	wsEvent, err := domain.EventFromDBEvent(&event)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	texts = append(texts, wsEvent.FormatText())
	// }
	// return texts, nil
}
