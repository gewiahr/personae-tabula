package service

import (
	"context"
	"encoding/json"
	"math/rand/v2"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/postgres"
	"personae-tabula/internal/repository/redis"
)

type EventService struct {
	eventRepo *postgres.EventRepository
	roomCache *redis.RoomCache
}

func NewEventService(
	eventRepo *postgres.EventRepository,
	roomCache *redis.RoomCache,
) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		roomCache: roomCache,
	}
}

type RollFormulaResult struct {
	Rolls []DiceRollResult `json:"rolls"`
	Mod   int              `json:"mod"`
}

type DiceRollResult struct {
	Dice   int   `json:"dice"`
	Amount int   `json:"amount"`
	Result []int `json:"result"`
}

type RollResult struct {
	Id         int64             `json:"id"`
	Type       string            `json:"type"`
	Formula    RollFormulaResult `json:"formula"`
	Title      string            `json:"title"`
	Roller     string            `json:"roller"`
	Message    string            `json:"message"`
	RollResult int               `json:"rollResult"`
}

func (s *EventService) ProcessSystemEvent(ctx context.Context, event *domain.SystemWSEvent[any]) error {
	//var err error
	// eventContent := &domain.RollRequestEvent{}
	// var rollResult *RollResult

	// switch event.Type {
	// case domain.SystemWSEventTypeJoin:
	// 	contentJSON, _ := json.Marshal(event.Content)
	// 	err := json.Unmarshal(contentJSON, eventContent)
	// 	// rollRequest, ok := event.Content.(RollRequest)
	// 	// if !ok {
	// 	// 	return nil, errors.New("failed to parse roll request")
	// 	// }
	// 	rollResult, err = s.ProcessRoll(ctx, eventContent)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	respEvent = &domain.WSEvent{
	// 		UserID:    event.UserID,
	// 		TableID:   event.TableID,
	// 		Timestamp: event.Timestamp,
	// 		Type:      event.Type,
	// 		Content:   rollResult,
	// 	}
	// }

	// var dbEvent *domain.TableEvent
	// if respEvent != nil {
	// 	dbEvent, err = respEvent.ToDBEvent()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// } else {
	// 	dbEvent, err = event.ToDBEvent()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// // В PostgreSQL
	// if err := s.eventRepo.Save(ctx, dbEvent); err != nil {
	// 	return nil, err
	// }

	// // В Redis кэш
	// if err := s.roomCache.PushEvent(ctx, event.TableID, event); err != nil {
	// 	log.Printf("error push to redis: %d", err)
	// 	// Логируем ошибку, но не прерываем выполнение
	// 	// return err
	// }

	// if respEvent != nil {
	// 	if content, ok := respEvent.Content.(*RollResult); ok && content != nil {
	// 		content.Id = dbEvent.ID
	// 	}
	// 	return respEvent, nil
	// }

	return nil
}

func (s *EventService) ProcessTableEvent(ctx context.Context, userID, tableID int64, event *domain.TableWSEvent[any], timestamp int64) (*domain.TableWSEvent[any], error) {
	//var err error
	//var respEvent *domain.TableWSEvent[any]
	//var content any
	//var rollResult *RollResult

	switch event.Type {
	case domain.TableWSEventTypeRoll:
		eventContent := &domain.RollRequestEvent{}
		contentJSON, _ := json.Marshal(event.Content)
		err := json.Unmarshal(contentJSON, eventContent)
		rollResult, err := s.ProcessRoll(ctx, eventContent)
		//content, err = s.ProcessRoll(ctx, eventContent)
		if err != nil {
			return nil, err
		}

		rollResultBytes, err := json.Marshal(rollResult)
		if err != nil {
			return nil, err
		}
		tableEvent, err := s.eventRepo.SaveEvent(ctx, userID, tableID, timestamp, string(event.Type), string(rollResultBytes))
		if err != nil {
			return nil, err
		}

		responseEvent := &RollResult{
			Id:         tableEvent.ID,
			Type:       tableEvent.EventType,
			Formula:    rollResult.Formula,
			Title:      rollResult.Title,
			Roller:     rollResult.Roller,
			Message:    rollResult.Message,
			RollResult: rollResult.RollResult,
		}

		return &domain.TableWSEvent[any]{
			Id:      tableEvent.ID,
			Type:    domain.TableWSEventTypeRoll,
			Content: responseEvent,
		}, nil

		// respEvent := &domain.WSEvent[*domain.TableWSEvent[*RollResult]]{
		// 	Type: domain.WSEventTypeTable,
		// 	Event: &domain.TableWSEvent[*RollResult]{
		// 		Id:      0,
		// 		Type:    domain.TableWSEventTypeRoll,
		// 		Content: rollResult,
		// 	},
		// 	UserID:    event.UserID,
		// 	TableID:   event.TableID,
		// 	Timestamp: event.Timestamp,
		// }

		// respEvent := &domain.TableWSEvent[*RollResult]{
		// 	Id:      0,
		// 	Type:    domain.TableWSEventTypeRoll,
		// 	Content: rollResult,
		// }

		// return &domain.TableWSEvent[any]{
		// 	Id:      0,
		// 	Type:    domain.TableWSEventTypeRoll,
		// 	Content: rollResult,
		// }, nil
	}

	// var dbEvent *domain.TableEvent
	// if respEvent != nil {
	// 	dbEvent, err = respEvent.ToDBEvent()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// } else {
	// 	dbEvent, err = event.ToDBEvent()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if respEvent != nil {
	// 	if content, ok := respEvent.Content.(*RollResult); ok && content != nil {
	// 		content.Id = dbEvent.ID
	// 	}
	// 	return respEvent, nil
	// }

	return &domain.TableWSEvent[any]{
		Id:      0,
		Type:    domain.TableWSEventTypeRoll,
		Content: event,
	}, nil
}

func (s *EventService) ProcessRoll(ctx context.Context, request *domain.RollRequestEvent) (*RollResult, error) {
	var result *RollResult

	switch request.Type {
	case "skill_roll":
		result = s.ProcessSkillRoll(request)
	case "attr_roll":
		result = s.ProcessSkillRoll(request) //s.ProcessAttributeRoll(request)
	case "save_roll":
		result = s.ProcessSkillRoll(request) //s.ProcessSaveRoll(request)
	case "damage_roll":
		result = s.ProcessSkillRoll(request) //s.ProcessDamageRoll(request)
	default:
		result = s.ProcessSkillRoll(request)
	}

	return result, nil
}

func (s *EventService) ProcessSkillRoll(request *domain.RollRequestEvent) *RollResult {
	var result = &RollResult{
		Type:    request.Type,
		Title:   request.Title,
		Roller:  request.Roller,
		Formula: RollFormulaResult{Mod: request.Formula.Mod, Rolls: []DiceRollResult{}},
	}
	rollResult := 0
	for _, roll := range request.Formula.Rolls {
		var rollResults []int
		for i := 0; i < roll.Amount; i++ {
			rolled := rand.IntN(roll.Dice) + 1
			rollResults = append(rollResults, rolled)
			rollResult = rollResult + rolled
		}
		result.Formula.Rolls = append(result.Formula.Rolls, DiceRollResult{Dice: roll.Dice, Amount: roll.Amount, Result: rollResults})
	}
	result.RollResult = rollResult + request.Formula.Mod
	return result
}

func (s *EventService) ProcessAttributeRoll(request *domain.RollRequestEvent) *RollResult {
	return nil
}

func (s *EventService) ProcessSaveRoll(request *domain.RollRequestEvent) *RollResult {
	return nil
}

func (s *EventService) ProcessDamageRoll(request *domain.RollRequestEvent) *RollResult {
	return nil
}

// Получить историю событий
// func (s *EventService) GetHistory(ctx context.Context, tableID int64, limit, offset int) ([]domain.WSEvent, error) {
// 	dbEvents, err := s.eventRepo.GetTableFeed(ctx, tableID, limit, offset)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var events []domain.WSEvent
// 	for _, dbEvent := range dbEvents {
// 		event, err := domain.EventFromDBEvent(&dbEvent)
// 		if err != nil {
// 			continue
// 		}
// 		events = append(events, *event)
// 	}

// 	return events, nil
// }

// Обработка броска кубов
// func (s *EventService) ProcessDiceRoll(ctx context.Context, tableID, userID int64, username, diceString string) (*domain.WSEvent, error) {
// 	// Здесь логика броска кубов
// 	result := calculateDiceRoll(diceString)

// 	event := &domain.WSEvent{
// 		Type:      domain.WSEventTypeRoll,
// 		TableID:   tableID,
// 		UserID:    userID,
// 		Content:   result,
// 		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
// 	}

// 	if err := s.ProcessEvent(ctx, event); err != nil {
// 		return nil, err
// 	}

// 	return event, nil
// }

// func calculateDiceRoll(diceString string) map[string]any {
// 	// Простая реализация для примера
// 	// В реальности нужно парсить "2d6+3" и т.д.
// 	return map[string]any{
// 		"dice":     diceString,
// 		"result":   10,
// 		"details":  []int{4, 3},
// 		"modifier": 3,
// 	}
// }
