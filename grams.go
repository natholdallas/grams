// Package grams is tiny packaging telegram-bot-api
package grams

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type (
	Handler     func(ctx *tgbotapi.BotAPI, ut tgbotapi.Update) error
	TaskHandler func(ctx *tgbotapi.BotAPI) error
)

type TelegramBot struct {
	Instance *tgbotapi.BotAPI
	Schedule *cron.Cron

	AllowedUpdates []string
	Limit          int
	Offset         int
	Timeout        int

	UpdateHandlers []Handler

	RegisterCommands      []tgbotapi.BotCommand
	CommandHanlders       map[string]Handler
	DefaultCommandHandler *Handler
	MsgHandlers           []Handler

	ChatHandlers       map[int64]Handler
	DefaultChatHandler *Handler

	SuccessfulPaymentHandler *Handler
	PreCheckoutQueryHandler  *Handler
	CallbackQueryHandler     *Handler
}

func New(token string) TelegramBot {
	instance, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	return TelegramBot{
		Instance:         instance,
		Schedule:         cron.New(cron.WithSeconds()),
		UpdateHandlers:   []Handler{},
		MsgHandlers:      []Handler{},
		RegisterCommands: []tgbotapi.BotCommand{},
		CommandHanlders:  make(map[string]Handler),
		ChatHandlers:     make(map[int64]Handler),
	}
}

func (s *TelegramBot) Task(spec string, handler TaskHandler) (cron.EntryID, error) {
	return s.Schedule.AddFunc(spec, taskWrap(s.Instance, handler))
}

func (s *TelegramBot) RemoveTask(id cron.EntryID) {
	s.Schedule.Remove(id)
}

func taskWrap(instance *tgbotapi.BotAPI, handler TaskHandler) func() {
	return func() {
		err := handler(instance)
		if err != nil {
			log.Println(err)
		}
	}
}

func (s *TelegramBot) NewCommand(command tgbotapi.BotCommand, handler Handler) {
	s.RegisterCommands = append(s.RegisterCommands, command)
	s.CommandHanlders[command.Command] = handler
}

func (s *TelegramBot) NewDefaultCommand(handler Handler) {
	s.DefaultCommandHandler = &handler
}

func (s *TelegramBot) NewChatMember(chatID int64, handler Handler) {
	s.ChatHandlers[chatID] = handler
}

func (s *TelegramBot) NewDefaultChatMember(handler Handler) {
	s.DefaultChatHandler = &handler
}

func (s *TelegramBot) NewUpdateEvent(handler Handler) {
	s.UpdateHandlers = append(s.UpdateHandlers, handler)
}

func (s *TelegramBot) NewMessage(handler Handler) {
	s.MsgHandlers = append(s.MsgHandlers, handler)
}

func (s *TelegramBot) OnSuccessfulPayment(handler Handler) {
	s.SuccessfulPaymentHandler = &handler
}

func (s *TelegramBot) OnPrecheckoutQuery(handler Handler) {
	s.PreCheckoutQueryHandler = &handler
}

func (s *TelegramBot) OnCallbackQuery(handler Handler) {
	s.CallbackQueryHandler = &handler
}

func (s *TelegramBot) Listen() {
	s.Schedule.Start()
	s.Instance.Request(tgbotapi.NewSetMyCommands(s.RegisterCommands...))
	update := tgbotapi.NewUpdate(0)
	update.AllowedUpdates = s.AllowedUpdates
	update.Limit = s.Limit
	update.Offset = s.Offset
	update.Timeout = s.Timeout
	for ut := range s.Instance.GetUpdatesChan(update) {
		// global update event
		for _, fun := range s.UpdateHandlers {
			s.execHandler(&fun, ut)
		}

		// message
		if ut.Message != nil {
			if ut.Message.IsCommand() {
				// handle command
				if h, ok := s.CommandHanlders[ut.Message.Command()]; ok {
					s.execHandler(&h, ut)
				} else {
					s.execHandler(s.DefaultCommandHandler, ut)
				}
			} else {
				// handle msg
				for _, fun := range s.MsgHandlers {
					s.execHandler(&fun, ut)
				}
			}

			// handle successful payment
			if ut.Message.SuccessfulPayment != nil {
				s.execHandler(s.SuccessfulPaymentHandler, ut)
			}
		}

		// handle chat member event
		if ut.ChatMember != nil {
			if h, ok := s.ChatHandlers[ut.ChatMember.Chat.ID]; ok {
				s.execHandler(&h, ut)
			} else {
				s.execHandler(s.DefaultChatHandler, ut)
			}
		}

		// handle precheckout
		if ut.PreCheckoutQuery != nil {
			s.execHandler(s.PreCheckoutQueryHandler, ut)
		}

		// handle callback
		if ut.CallbackQuery != nil {
			s.execHandler(s.CallbackQueryHandler, ut)
		}

	}
}

func (s *TelegramBot) execHandler(fun *Handler, ut tgbotapi.Update) {
	if fun != nil {
		if err := (*fun)(s.Instance, ut); err != nil {
			log.Println(err)
		}
	}
}
