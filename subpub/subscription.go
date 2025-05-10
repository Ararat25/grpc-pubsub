package subpub

import "sync"

type subscription struct {
	handler MessageHandler
	msgCh   chan any
	once    sync.Once
	sb      *subPub
	subject string
}

func (s *subscription) Unsubscribe() {
	s.once.Do(func() {
		s.sb.unsubscribe(s.subject, s)
		close(s.msgCh)
	})
}
