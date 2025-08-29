// Package grams is tiny packaging telegram-bot-api
package grams

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type (
	Handler     func(ctx *tgbotapi.BotAPI, ut tgbotapi.Update)
	TaskHandler func(ctx *tgbotapi.BotAPI)
)

func TODO(ctx *tgbotapi.BotAPI, ut tgbotapi.Update) error {
	return nil
}

type Bot struct {
	*tgbotapi.BotAPI
	Schedule *cron.Cron

	AllowedUpdates []string
	Limit          int
	Offset         int
	Timeout        int

	updateHandler Handler

	commands              []tgbotapi.BotCommand
	commandHanlders       map[string]Handler
	commandDefaultHandler Handler

	msgHandler Handler

	chatHandlers       map[int64]Handler
	chatDefaultHandler Handler

	successfulPayment Handler
	preCheckoutQuery  Handler
	callbackQuery     Handler
}

func New(token string) *Bot {
	instance, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	return &Bot{
		BotAPI:          instance,
		Schedule:        cron.New(cron.WithSeconds()),
		commands:        []tgbotapi.BotCommand{},
		commandHanlders: make(map[string]Handler),
		chatHandlers:    make(map[int64]Handler),
	}
}

func (s *Bot) NewTask(spec string, fun TaskHandler) (cron.EntryID, error) {
	return s.Schedule.AddFunc(spec, func() { fun(s.BotAPI) })
}

func (s *Bot) RemoveTask(id cron.EntryID) {
	s.Schedule.Remove(id)
}

func (s *Bot) NewCommand(command tgbotapi.BotCommand, handler Handler) {
	s.commands = append(s.commands, command)
	s.commandHanlders[command.Command] = handler
}

func (s *Bot) NewDefaultCommand(fun Handler) {
	s.commandDefaultHandler = fun
}

func (s *Bot) NewChatMember(chatID int64, fun Handler) {
	s.chatHandlers[chatID] = fun
}

func (s *Bot) NewDefaultChatMember(fun Handler) {
	s.chatDefaultHandler = fun
}

func (s *Bot) NewUpdateEvent(fun Handler) {
	s.updateHandler = fun
}

func (s *Bot) NewMessage(fun Handler) {
	s.msgHandler = fun
}

func (s *Bot) OnSuccessfulPayment(fun Handler) {
	s.successfulPayment = fun
}

func (s *Bot) OnPrecheckoutQuery(fun Handler) {
	s.preCheckoutQuery = fun
}

func (s *Bot) OnCallbackQuery(fun Handler) {
	s.callbackQuery = fun
}

func (s *Bot) Listen() {
	s.Schedule.Start()
	s.Request(tgbotapi.NewSetMyCommands(s.commands...))

	update := tgbotapi.NewUpdate(0)
	update.AllowedUpdates = s.AllowedUpdates
	update.Limit = s.Limit
	update.Offset = s.Offset
	update.Timeout = s.Timeout

	for ut := range s.GetUpdatesChan(update) {
		// global update event
		go s.exec(s.updateHandler, ut)

		// message
		if ut.Message != nil {
			if ut.Message.IsCommand() {
				// handle command
				if h, ok := s.commandHanlders[ut.Message.Command()]; ok {
					go s.exec(h, ut)
				} else {
					go s.exec(s.commandDefaultHandler, ut)
				}
			} else {
				// handle msg
				go s.exec(s.msgHandler, ut)
			}

			// handle successful payment
			if ut.Message.SuccessfulPayment != nil {
				go s.exec(s.successfulPayment, ut)
			}
		}

		// handle chat member event
		if ut.ChatMember != nil {
			if h, ok := s.chatHandlers[ut.ChatMember.Chat.ID]; ok {
				go s.exec(h, ut)
			} else {
				go s.exec(s.chatDefaultHandler, ut)
			}
		}

		// handle precheckout
		if ut.PreCheckoutQuery != nil {
			go s.exec(s.preCheckoutQuery, ut)
		}

		// handle callback
		if ut.CallbackQuery != nil {
			go s.exec(s.callbackQuery, ut)
		}

	}
}

func (s *Bot) exec(fun Handler, ut tgbotapi.Update) {
	if fun == nil {
		return
	}
	fun(s.BotAPI, ut)
}
