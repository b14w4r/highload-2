package services

import (
	"context"
	"time"

	"go-microservice/utils"
)

type Notification struct {
	Action string
	UserID int
	At     time.Time
}

type Notifier struct {
	log  *utils.Logger
	ch   chan Notification
	errs chan<- error
}

func NewNotifier(log *utils.Logger, buffer int) *Notifier {
	return &Notifier{
		log: log,
		ch:  make(chan Notification, buffer),
	}
}

func (n *Notifier) BindErrorSink(errs chan<- error) {
	n.errs = errs
}

func (n *Notifier) Notify(action string, userID int) {
	select {
	case n.ch <- Notification{Action: action, UserID: userID, At: time.Now()}:
	default:
		// Не блокируем запросы: при переполнении просто дропаем
		if n.errs != nil {
			select {
			case n.errs <- context.DeadlineExceeded:
			default:
			}
		}
	}
}

func (n *Notifier) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				n.log.Info("notifier stopped")
				return
			case msg := <-n.ch:
				// Тут могла бы быть интеграция (email, kafka, etc.)
				n.log.Info("notify", "action", msg.Action, "user_id", msg.UserID, "at", msg.At.UTC().Format(time.RFC3339Nano))
			}
		}
	}()
}

