package session

import (
	"context"
	"encoding/base32"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/session"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

const (
	sessionIDLen  = 32
	defaultMaxAge = 60 * 60 * 24 * 30 // 30 days
	defaultPath   = "/"
)

var store *Store

func GetStore() (*Store, error) {
	if store == nil {
		return nil, errors.New("session store is not initialized")
	}
	return store, nil
}

func SetStore(newStore *Store) {
	store = newStore
}

// Options for entstore.
type Options struct {
	SkipCreateTable bool
}

// Store represents an entstore.
type Store struct {
	client      *ent.Client
	opts        Options
	Codecs      []securecookie.Codec
	SessionOpts *sessions.Options
}

// New creates a new entstore session.
func New(client *ent.Client, keyPairs ...[]byte) *Store {
	return NewOptions(client, Options{
		SkipCreateTable: true,
	}, keyPairs...)
}

// NewOptions creates a new entstore session with options.
func NewOptions(client *ent.Client, opts Options, keyPairs ...[]byte) *Store {
	st := &Store{
		client: client,
		opts:   opts,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		SessionOpts: &sessions.Options{
			Path:   defaultPath,
			MaxAge: defaultMaxAge,
		},
	}

	if !st.opts.SkipCreateTable {
		log.Print(`Not Supported: entstore does not support creating tables automatically. Please use ent CLI to create the table.`)
	}

	return st
}

// Get returns a session for the given name after adding it to the registry.
func (st *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(st, name)
}

// New creates a session with the name without adding it to the registry.
func (st *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(st, name)
	opts := *st.SessionOpts
	session.Options = &opts
	session.IsNew = true

	st.MaxAge(st.SessionOpts.MaxAge)

	s, sessionErr := st.getSessionFromCookie(r.Context(), r, session.Name())
	if sessionErr != nil {
		return session, sessionErr // Continue with a new session if error
	}

	if s != nil {
		if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values, st.Codecs...); err != nil {
			return session, err
		}
		session.ID = s.ID
		session.IsNew = false
	}

	return session, nil
}

// Save session and set cookie header.
func (st *Store) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	s, err := st.getSessionFromCookie(r.Context(), r, session.Name())
	if err != nil {
		return err
	}

	if session.Options.MaxAge < 0 {
		if s != nil {
			if sessionErr := st.client.Session.DeleteOneID(s.ID).Exec(r.Context()); sessionErr != nil {
				return sessionErr
			}
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	data, err := securecookie.EncodeMulti(session.Name(), session.Values, st.Codecs...)
	if err != nil {
		return err
	}

	now := time.Now()
	expire := now.Add(time.Second * time.Duration(session.Options.MaxAge))
	if s == nil {
		if st.client == nil {
			return errors.New("ent client is nil")
		}

		session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(sessionIDLen)), "=")

		_, saveErr := st.client.Session.
			Create().
			SetID(session.ID).
			SetData(data).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetExpiresAt(expire).
			Save(r.Context())
		if saveErr != nil {
			return saveErr
		}
	} else {
		if _, updateErr := st.client.Session.
			UpdateOneID(s.ID).
			SetData(data).
			SetUpdatedAt(now).
			SetExpiresAt(expire).
			Save(r.Context()); updateErr != nil {
			return updateErr
		}
	}

	// Cookie encoding
	id, err := securecookie.EncodeMulti(session.Name(), session.ID, st.Codecs...)
	if err != nil {
		return err
	}

	// Setting the cookie
	http.SetCookie(w, sessions.NewCookie(session.Name(), id, session.Options))

	return nil
}

// getSessionFromCookie looks for an existing EntSession from a session ID stored inside a cookie.
func (st *Store) getSessionFromCookie(ctx context.Context, r *http.Request, name string) (*ent.Session, error) {
	if cookie, err := r.Cookie(name); err == nil {
		sessionID := ""
		if decodeErr := securecookie.DecodeMulti(name, cookie.Value, &sessionID, st.Codecs...); decodeErr != nil {
			return nil, decodeErr
		}
		session, queryErr := st.client.Session.
			Query().
			Where(session.IDEQ(sessionID), session.ExpiresAtGT(time.Now())).
			Only(ctx)
		if queryErr != nil {
			return nil, queryErr
		}
		return session, nil
	}
	return nil, errors.New("session not found")
}

// MaxAge sets the maximum age for the store and the underlying cookie implementation.
func (st *Store) MaxAge(age int) {
	st.SessionOpts.MaxAge = age
	for _, codec := range st.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// MaxLength restricts the maximum length of new sessions to l.
func (st *Store) MaxLength(l int) {
	for _, c := range st.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// Cleanup deletes expired sessions.
func (st *Store) Cleanup() {
	affected, err := st.client.Session.Delete().Where(session.ExpiresAtLTE(time.Now())).Exec(context.Background())
	if err != nil {
		log.Printf("failed to cleanup expired sessions: %v", err)
	}

	log.Printf("cleanup: %d sessions removed", affected)
}
