package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"hackatonApp/internal/service"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

type Bot struct {
	api        *maxbot.Api
	events     *service.EventService
	selfID     int64
	miniAppURL string
}

func New(token string, events *service.EventService, miniAppURL string) (*Bot, error) {
	api, err := maxbot.NewApi(token)
	if err != nil {
		return nil, fmt.Errorf("max bot: init api: %w", err)
	}

	return &Bot{
		api:        api,
		events:     events,
		miniAppURL: miniAppURL,
	}, nil
}

func (b *Bot) Run(ctx context.Context) {
	info, err := b.api.Bots.GetMyInfo(ctx)
	if err != nil {
		slog.Error("max bot: get bot info failed", "error", err)
		return
	}
	b.selfID = info.UserID
	slog.Info("max bot: started", "name", info.Username, "id", info.UserID)

	var marker int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, nextMarker, err := b.api.Subscriptions.GetUpdates(ctx, marker)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Error("max bot: get updates failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		marker = nextMarker

		for _, upd := range updates {
			b.handleUpdate(ctx, upd)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, upd model.Update) {
	if upd.Message == nil {
		return
	}

	chatID := upd.Message.Recipient.ChatID
	text := upd.Message.Body.Text

	reply, err := b.buildReply(ctx, text)
	if err != nil {
		slog.Error("max bot: build reply failed", "error", err)
		reply = "Что то пошло не так, попробуй еще раз чуть позже"
	}

	msg := maxbot.NewMessage().SetChat(chatID).SetText(reply)
	if _, err := b.api.Messages.Send(ctx, msg); err != nil {
		slog.Error("max bot: send reply failed", "error", err)
	}
}

func (b *Bot) buildReply(ctx context.Context, text string) (string, error) {
	args := strings.Fields(strings.TrimSpace(text))
	if len(args) == 0 {
		return helpText(), nil
	}

	switch strings.ToLower(args[0]) {
	case "/start", "/help":
		return helpText(), nil
	case "/cities":
		return b.listCities(ctx)
	case "/categories":
		return b.listCategories(ctx)
	case "/events":
		return b.searchReply(ctx, args[1:])
	default:
		return "Не понял команду \n\n" + helpText(), nil
	}
}

func helpText() string {
	return "Я помогу найти события\n\n" +
		"Команды:\n" +
		"/events <город> [бесплатно] - поиск событий\n" +
		"/cities - какие города сейчас есть в базе\n" +
		"/categories - какие категории есть в базе\n" +
		"Пример: /events Москва бесплатно"
}

func (b *Bot) listCities(ctx context.Context) (string, error) {
	cities, err := b.events.Cities(ctx)
	if err != nil {
		return "", fmt.Errorf("bot: list cities: %w", err)
	}
	if len(cities) == 0 {
		return "Пока в базе нет ни одного города с событиями", nil
	}
	return "Доступные города:\n" + strings.Join(cities, ", "), nil
}

func (b *Bot) listCategories(ctx context.Context) (string, error) {
	categories, err := b.events.Categories(ctx)
	if err != nil {
		return "", fmt.Errorf("bot: list categories: %w", err)
	}
	if len(categories) == 0 {
		return "Категории пока не определены", nil
	}
	return "Доступные категории:\n" + strings.Join(categories, ", "), nil
}

func (b *Bot) searchReply(ctx context.Context, args []string) (string, error) {
	params := service.SearchParams{}
	for _, a := range args {
		switch strings.ToLower(a) {
		case "бесплатно", "free":
			params.OnlyFree = true
		case "пушкинская", "пушкин":
			params.OnlyPushkin = true
		default:
			params.City = a
		}
	}

	events, err := b.events.Search(ctx, params)
	if err != nil {
		return "", fmt.Errorf("bot: search: %w", err)
	}
	if len(events) == 0 {
		return "По твоему запросу ничего не нашлось \n\nПопробуй /cities, чтобы увидеть доступные города.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Нашел %d событий:\n\n", len(events)))
	for i, ev := range events {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("...и еще %d, уточни запрос для более узкого списка", len(events)-10))
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n%s, %s\n%s\n\n", i+1, ev.Title, ev.City, ev.StartAt.Format("02.01 15:04"), ev.URL))
	}
	return sb.String(), nil
}

// Если будем подключать Mini app
// func (b *Bot) sendOpenAppButton(ctx context.Context, chatID int64) {
// 	keyboard := model.NewKeyboard()
// 	keyboard.AddRow().AddOpenApp("Открыть афишу", b.selfID)

// 	msg := maxbot.NewMessage().SetChat(chatID).SetText("Вот афиша:").AddKeyboard(keyboard)
// 	if _, err := b.api.Messages.Send(ctx, msg); err != nil {
// 		slog.Error("max bot: send open_app button failed", "error", err)
// 	}
// }

func (b *Bot) RegisterWebhook(ctx context.Context, mux *http.ServeMux, publicURL, secret string) error {
	mux.Handle("/max/webhook", b.api.GetHandler(b.handleUpdate, secret))

	updateType := []string{"message_created", "bot_started", "message_callback"}
	if _, err := b.api.Subscriptions.Subscribe(ctx, publicURL, secret, updateType, ""); err != nil {
		return fmt.Errorf("max bot: subscribe webhook: %w", err)
	}
	return nil
}
