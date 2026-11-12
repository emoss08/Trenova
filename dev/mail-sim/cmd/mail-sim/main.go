// Command mail-sim stands in for an email provider in development.
//
// Trenova sends mail through Resend or Postmark. Neither can be reached with
// no key, so on a development machine every write that ends in an email stops
// there: a plan's "email the customer" step fails, and nothing after it can be
// tried. This answers Resend's send call, keeps what it was sent, and shows it
// at /, so a developer can run a plan to its end and read the email it would
// have delivered. It delivers nothing anywhere.
//
// Point the Resend integration's base URL at it (the desk fixtures do) and set
// integrations.allowPrivateNetworks in the service's config, since it listens
// on localhost.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const maxKept = 200

type message struct {
	ID       string    `json:"id"`
	From     string    `json:"from"`
	To       []string  `json:"to"`
	CC       []string  `json:"cc,omitempty"`
	BCC      []string  `json:"bcc,omitempty"`
	ReplyTo  any       `json:"reply_to,omitempty"`
	Subject  string    `json:"subject"`
	HTML     string    `json:"html,omitempty"`
	Text     string    `json:"text,omitempty"`
	Received time.Time `json:"received"`
}

type outbox struct {
	mu   sync.Mutex
	kept []message
}

func (o *outbox) add(m message) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.kept = append([]message{m}, o.kept...)
	if len(o.kept) > maxKept {
		o.kept = o.kept[:maxKept]
	}
}

func (o *outbox) all() []message {
	o.mu.Lock()
	defer o.mu.Unlock()

	return append([]message(nil), o.kept...)
}

func newID() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(raw)
}

var page = template.Must(template.New("outbox").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Mail simulator</title>
<style>
body{font:14px/1.5 system-ui,sans-serif;margin:24px;max-width:860px}
article{border:1px solid #8884;border-radius:8px;padding:12px 16px;margin:0 0 12px}
h1{font-size:18px} h2{font-size:15px;margin:0 0 4px} small{opacity:.7}
pre{white-space:pre-wrap;margin:8px 0 0}
</style></head><body>
<h1>Mail simulator</h1>
<p><small>What Trenova sent, newest first. Nothing here was delivered to anyone.</small></p>
{{range .}}<article>
<h2>{{.Subject}}</h2>
<small>{{.Received.Format "Jan 2 15:04:05"}} · from {{.From}} · to {{range $i, $to := .To}}{{if $i}}, {{end}}{{$to}}{{end}}</small>
<pre>{{if .Text}}{{.Text}}{{else}}{{.HTML}}{{end}}</pre>
</article>{{else}}<p>Nothing has been sent yet.</p>{{end}}
</body></html>`))

func main() {
	kept := &outbox{}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /emails", func(w http.ResponseWriter, r *http.Request) {
		var m message
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 30<<20)).Decode(&m); err != nil {
			http.Error(w, `{"message":"the body is not a Resend email"}`, http.StatusUnprocessableEntity)
			return
		}
		m.ID, m.Received = newID(), time.Now()
		kept.add(m)
		log.Printf("kept %q to %v", m.Subject, m.To)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": m.ID})
	})
	mux.HandleFunc("GET /domains", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	})
	mux.HandleFunc("GET /_sim/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /_sim/messages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(kept.all())
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, kept.all()); err != nil {
			log.Printf("render the outbox: %v", err)
		}
	})

	addr := os.Getenv("MAIL_SIM_ADDR")
	if addr == "" {
		addr = ":8092"
	}
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("mail simulator listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
