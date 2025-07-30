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
	Instance                 *tgbotapi.BotAPI
	Schedule                 *cron.Cron
	registerCommands         []tgbotapi.BotCommand
	msgHandler               *Handler
	commandHanlders          map[string]Handler
	defaultCommandHandler    *Handler
	chatHandlers             map[int64]Handler
	defaultChatHandler       *Handler
	successfulPaymentHandler *Handler
	preCheckoutQueryHandler  *Handler
}

func New(token string) TelegramBot {
	instance, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	return TelegramBot{
		Instance:         instance,
		Schedule:         cron.New(cron.WithSeconds()),
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

func (s *TelegramBot) OnSuccessfulPayment(handler Handler) {
	s.successfulPaymentHandler = &handler
}

func (s *TelegramBot) OnPrecheckoutQuery(handler Handler) {
	s.preCheckoutQueryHandler = &handler
}

func (s *TelegramBot) Listen() {
	s.Schedule.Start()
	s.Instance.Request(tgbotapi.NewSetMyCommands(s.registerCommands...))
	for ut := range s.Instance.GetUpdatesChan(tgbotapi.NewUpdate(0)) {
		if ut.Message != nil {

			if ut.Message.IsCommand() {
				// handle command
				s.handleCommand(ut)
			} else {
				// handle msg
				s.handleMessage(ut)
			}

			// handle successful payment
			if ut.Message.SuccessfulPayment != nil {
				if s.successfulPaymentHandler != nil {
					if err := (*s.successfulPaymentHandler)(s.Instance, ut); err != nil {
						log.Println(err)
					}
				}
			}

		}

		// handle chat member event
		if ut.ChatMember != nil {
			s.handleChatMember(ut)
		}

		// handle precheckout
		if ut.PreCheckoutQuery != nil {
			if s.preCheckoutQueryHandler != nil {
				if err := (*s.preCheckoutQueryHandler)(s.Instance, ut); err != nil {
					log.Println(err)
				}
			}
		}

	}
}

func (s *TelegramBot) handleMessage(ut tgbotapi.Update) {
	if s.msgHandler == nil {
		return
	}
	if err := (*s.msgHandler)(s.Instance, ut); err != nil {
		log.Println(err)
	}
}

func (s *TelegramBot) handleCommand(ut tgbotapi.Update) {
	if h, ok := s.commandHanlders[ut.Message.Command()]; ok {
		if err := h(s.Instance, ut); err != nil {
			log.Println(err)
		}
	} else if h := *s.defaultCommandHandler; h != nil {
		if err := h(s.Instance, ut); err != nil {
			log.Println(err)
		}
	}
}

func (s *TelegramBot) handleChatMember(ut tgbotapi.Update) {
	if h, ok := s.chatHandlers[ut.ChatMember.Chat.ID]; ok {
		if err := h(s.Instance, ut); err != nil {
			log.Println(err)
		}
	} else if h := *s.defaultChatHandler; h != nil {
		if err := h(s.Instance, ut); err != nil {
			log.Println(err)
		}
	}
}
