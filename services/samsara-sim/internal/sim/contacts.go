package sim

import (
	"errors"
	"net/http"
	"slices"
	"time"
)

const (
	fieldFirstName       = "firstName"
	fieldLastName        = "lastName"
	fieldContactPhone    = "phone"
	maxContactFieldChars = 255
	labelContact         = "contact"
)

var contactFields = []string{keyEmail, fieldFirstName, fieldLastName, fieldContactPhone}

func (s *Server) registerContactRoutes() {
	s.mux.HandleFunc("GET /contacts", s.handleContactList)
	s.mux.HandleFunc("POST /contacts", s.handleContactCreate)
	s.mux.HandleFunc("GET /contacts/{id}", s.handleContactGet)
	s.mux.HandleFunc("PATCH /contacts/{id}", s.handleContactPatch)
	s.mux.HandleFunc("DELETE /contacts/{id}", s.handleContactDelete)
}

func contactBodyRules() []fieldRule {
	return []fieldRule{
		{
			Name:   keyEmail,
			Kind:   kindString,
			MaxLen: maxContactFieldChars,
			Check:  optionalEmail,
		},
		{Name: fieldFirstName, Kind: kindString, MaxLen: maxContactFieldChars},
		{Name: fieldLastName, Kind: kindString, MaxLen: maxContactFieldChars},
		{Name: fieldContactPhone, Kind: kindString, MaxLen: maxContactFieldChars},
	}
}

func optionalEmail(value any) error {
	if stringOf(value) == "" {
		return nil
	}
	return validateEmail(value)
}

func contactView(contact Record) Record {
	out := Record{keyID: recordID(contact)}
	for _, field := range contactFields {
		out[field] = stringValue(contact, field)
	}
	return out
}

func (s *Server) handleContactList(writer http.ResponseWriter, request *http.Request) {
	contacts, err := s.store.List(ResourceContacts)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	records := make([]Record, 0, len(contacts))
	for _, contact := range contacts {
		records = append(records, contactView(contact))
	}
	s.respondPage(writer, request, records, "|contact-list")
}

func (s *Server) handleContactGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	contact, err := s.store.Get(ResourceContacts, id)
	if err != nil {
		s.writeError(writer, contactLookupError(err, id))
		return
	}
	s.respondContact(writer, request, contact, "|contact-get")
}

func (s *Server) handleContactCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, contactBodyRules(), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	var created Record
	err = s.store.Transact(func(tx *storeTx) error {
		var insertErr error
		created, insertErr = tx.insert(ResourceContacts, sanitized, now)
		return insertErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondContact(writer, request, created, "|contact-create")
}

func (s *Server) handleContactPatch(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, contactBodyRules(), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	var updated Record
	err = s.store.Transact(func(tx *storeTx) error {
		var updateErr error
		updated, updateErr = tx.update(ResourceContacts, id, func(record Record) error {
			for field, value := range sanitized {
				record[field] = value
			}
			return nil
		}, now)
		return updateErr
	})
	if err != nil {
		s.writeError(writer, contactLookupError(err, id))
		return
	}
	s.respondContact(writer, request, updated, "|contact-patch")
}

func (s *Server) handleContactDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	err = s.store.Transact(func(tx *storeTx) error {
		if removeErr := tx.remove(ResourceContacts, id); removeErr != nil {
			return removeErr
		}
		return detachContactTx(tx, id, now)
	})
	if err != nil {
		s.writeError(writer, contactLookupError(err, id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func detachContactTx(tx *storeTx, contactID string, now time.Time) error {
	for _, address := range tx.records(ResourceAddresses) {
		contactIDs := stringListValues(address[fieldContactIDs])
		if !slices.Contains(contactIDs, contactID) {
			continue
		}
		_, err := tx.update(ResourceAddresses, recordID(address), func(record Record) error {
			remaining := slices.DeleteFunc(
				contactIDs,
				func(id string) bool { return id == contactID },
			)
			if len(remaining) == 0 {
				delete(record, fieldContactIDs)
				return nil
			}
			record[fieldContactIDs] = stringsAsAny(remaining)
			return nil
		}, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func contactLookupError(err error, id string) error {
	if errors.Is(err, ErrRecordNotFound) {
		return notFound(labelContact, id)
	}
	return err
}

func (s *Server) respondContact(
	writer http.ResponseWriter,
	request *http.Request,
	contact Record,
	signature string,
) {
	payload := map[string]any{keyData: contactView(contact)}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}
