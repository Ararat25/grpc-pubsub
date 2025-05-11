package subpub

import "sync"

// subscription - структура реализующая интерфейс Subscription
type subscription struct {
	handler MessageHandler // это функция обратного вызова, которая обрабатывает сообщения, доставляемые подписчикам
	msgCh   chan any       // канал для передачи сообщения подписчику
	once    sync.Once      // примитив синхронзации для контроля состояния гонки
	sb      *subPub        // объект структуры Publisher-Subscriber сервиса
	subject string         // тема на которую совершена подписка
}

// Unsubscribe отписывает подписчика от темы
func (s *subscription) Unsubscribe() {
	s.once.Do(func() {
		s.sb.unsubscribe(s.subject, s)
		close(s.msgCh)
	})
}
