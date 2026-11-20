package sim

import (
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

func (s *Server) Delivered(user, name string, contents []byte) {
	p, ok := s.peers[user]
	if !ok {
		return
	}
	s.receive(p, &inbound{raw: contents, transport: transportSFTP, fileName: name})
}

func (s *Server) PickedUp(user, name string) {
	p, ok := s.peers[user]
	if !ok {
		return
	}
	sent, found := s.journal.Find(p.name(), func(c *journal.Message) bool {
		return c.Direction == journal.ToTrenova && c.Transport == transportSFTP &&
			c.FileName == name && c.PickedUpAt == nil
	})
	if !found {
		return
	}
	at := s.clock.Now().UTC()
	s.journal.Update("file.picked_up", sent.ID, func(stored *journal.Message) {
		stored.PickedUpAt = &at
	})
	s.changed()
}
