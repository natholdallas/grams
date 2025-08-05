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

	updateHandlers []Handler

	registerCommands      []tgbotapi.BotCommand
	commandHanlders       map[string]Handler
	defaultCommandHandler *Handler
	msgHandlers           []Handler

	chatHandlers       map[int64]Handler
	defaultChatHandler *Handler

	successfulPaymentHandler *Handler
	preCheckoutQueryHandler  *Handler
	callbackQueryHandler     *Handler
}

func New(token string) TelegramBot {
	instance, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	return TelegramBot{
		Instance:         instance,
		Schedule:         cron.New(cron.WithSeconds()),
		updateHandlers:   []Handler{},
		msgHandlers:      []Handler{},
		registerCommands: []tgbotapi.BotCommand{},
		commandHanlders:  make(map[string]Handler),
		chatHandlers:     make(map[int64]Handler),
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
	s.registerCommands = append(s.registerCommands, command)
	s.commandHanlders[command.Command] = handler
}

func (s *TelegramBot) NewDefaultCommand(handler Handler) {
	s.defaultCommandHandler = &handler
}

func (s *TelegramBot) NewChatMember(chatID int64, handler Handler) {
	s.chatHandlers[chatID] = handler
}

func (s *TelegramBot) NewDefaultChatMember(handler Handler) {
	s.defaultChatHandler = &handler
}

func (s *TelegramBot) NewUpdateEvent(handler Handler) {
	s.updateHandlers = append(s.updateHandlers, handler)
}

func (s *TelegramBot) NewMessage(handler Handler) {
	s.msgHandlers = append(s.msgHandlers, handler)
}

func (s *TelegramBot) OnSuccessfulPayment(handler Handler) {
	s.successfulPaymentHandler = &handler
}

func (s *TelegramBot) OnPrecheckoutQuery(handler Handler) {
	s.preCheckoutQueryHandler = &handler
}

func (s *TelegramBot) OnCallbackQuery(handler Handler) {
	s.callbackQueryHandler = &handler
}

func (s *TelegramBot) Listen() {
	s.Schedule.Start()
	s.Instance.Request(tgbotapi.NewSetMyCommands(s.registerCommands...))
	for ut := range s.Instance.GetUpdatesChan(tgbotapi.NewUpdate(0)) {
		// global update event
		for _, fun := range s.updateHandlers {
			s.execHandler(&fun, ut)
		}

		// message
		if ut.Message != nil {
			if ut.Message.IsCommand() {
				// handle command
				if h, ok := s.commandHanlders[ut.Message.Command()]; ok {
					s.execHandler(&h, ut)
				} else {
					s.execHandler(s.defaultCommandHandler, ut)
				}
			} else {
				// handle msg
				for _, fun := range s.msgHandlers {
					s.execHandler(&fun, ut)
				}
			}

			// handle successful payment
			if ut.Message.SuccessfulPayment != nil {
				s.execHandler(s.successfulPaymentHandler, ut)
			}
		}

		// handle chat member event
		if ut.ChatMember != nil {
			if h, ok := s.chatHandlers[ut.ChatMember.Chat.ID]; ok {
				s.execHandler(&h, ut)
			} else {
				s.execHandler(s.defaultChatHandler, ut)
			}
		}

		// handle precheckout
		if ut.PreCheckoutQuery != nil {
			s.execHandler(s.preCheckoutQueryHandler, ut)
		}

		// handle callback
		if ut.CallbackQuery != nil {
			s.execHandler(s.callbackQueryHandler, ut)
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
