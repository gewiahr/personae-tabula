package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/postgres"
	"personae-tabula/internal/repository/redis"
)

type TableService struct {
	tableRepo *postgres.TableRepository
	eventRepo *postgres.EventRepository
	roomCache *redis.RoomCache
}

func NewTableService(
	tableRepo *postgres.TableRepository,
	eventRepo *postgres.EventRepository,
	roomCache *redis.RoomCache,
) *TableService {
	return &TableService{
		tableRepo: tableRepo,
		eventRepo: eventRepo,
		roomCache: roomCache,
	}
}

func (s *TableService) CreateTable(ctx context.Context, createdBy int64, name, description string) (*domain.Table, error) {
	joinCode := hex.EncodeToString(func() []byte {
		b := make([]byte, 4)
		rand.Read(b)
		return b
	}())

	table := &domain.Table{
		Name:      name,
		CreatedBy: createdBy,
		IsActive:  true,
		JoinCode:  joinCode,
	}

	if err := s.tableRepo.Create(ctx, table); err != nil {
		return nil, err
	}

	// Создаем системное событие о создании стола
	// event := &domain.WSEvent{
	// 	Type:      domain.WSEventTypeSystem,
	// 	TableID:   table.ID,
	// 	Content:   "Стол создан",
	// 	Timestamp: table.CreatedAt.UnixNano() / int64(time.Millisecond),
	// }

	// s.saveEvent(ctx, event)

	return table, nil
}

func (s *TableService) GetTable(ctx context.Context, id int64) (*domain.Table, error) {
	return s.tableRepo.GetByID(ctx, id)
}

func (s *TableService) GetTableFeed(ctx context.Context, tableID int64, limit int) ([]domain.TableEvent, error) { //([]string, error) {
	// Сначала пробуем получить из кэша
	// events, err := s.roomCache.GetRecentEvents(ctx, tableID, limit)
	// if err == nil && len(events) > 0 {
	// 	return events, nil
	// }

	// Если в кэше нет, грузим из БД
	//return s.eventRepo.GetTableFeedText(ctx, tableID, limit)
	return s.eventRepo.GetTableFeed(ctx, tableID, 50, 0)
}

// Внутренний метод для сохранения события
// func (s *TableService) saveEvent(ctx context.Context, event *domain.WSEvent) error {
// 	dbEvent, err := event.ToDBEvent()
// 	if err != nil {
// 		return err
// 	}

// 	// Сохраняем в PostgreSQL
// 	if err := s.eventRepo.Save(ctx, dbEvent); err != nil {
// 		return err
// 	}

// 	// Сохраняем в Redis кэш
// 	return s.roomCache.PushEvent(ctx, event.TableID, event)
// }

func (s *TableService) ListTables(ctx context.Context, limit, offset int) ([]domain.Table, error) {
	tables, err := s.tableRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
