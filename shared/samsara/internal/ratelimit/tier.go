package ratelimit

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Tier uint8

const (
	TierUnlisted Tier = iota
	TierLevelOne
	TierLevelTwo
	TierLevelThree
	TierLegacyOne
	TierLegacyTwo
)

const (
	TokenRequestsPerSecond = 150
	wildcardSegment        = "{id}"
)

func (t Tier) Limit() rate.Limit {
	switch t {
	case TierLevelOne:
		return rate.Every(time.Minute / 100)
	case TierLevelTwo:
		return rate.Limit(5)
	case TierLevelThree:
		return rate.Limit(10)
	case TierLegacyOne:
		return rate.Limit(25)
	case TierLegacyTwo:
		return rate.Limit(50)
	case TierUnlisted:
		return rate.Inf
	default:
		return rate.Inf
	}
}

func (t Tier) String() string {
	switch t {
	case TierLevelOne:
		return "level-one"
	case TierLevelTwo:
		return "level-two"
	case TierLevelThree:
		return "level-three"
	case TierLegacyOne:
		return "legacy-tier-1"
	case TierLegacyTwo:
		return "legacy-tier-2"
	case TierUnlisted:
		return "unlisted"
	default:
		return "unlisted"
	}
}

type Endpoint struct {
	Key  string
	Tier Tier
}

type routeSpec struct {
	spec string
	tier Tier
}

type route struct {
	method   string
	template string
	segments []string
	tier     Tier
}

var routes = buildRoutes([]routeSpec{
	{"GET /addresses", TierLevelTwo},
	{"POST /addresses", TierLevelOne},
	{"GET /addresses/{id}", TierLegacyOne},
	{"PATCH /addresses/{id}", TierLevelOne},
	{"DELETE /addresses/{id}", TierLevelOne},
	{"GET /assets", TierUnlisted},
	{"POST /assets", TierLevelOne},
	{"PATCH /assets", TierLevelOne},
	{"DELETE /assets", TierLevelOne},
	{"GET /assets/location-and-speed/stream", TierLevelTwo},
	{"GET /dvirs/stream", TierLevelTwo},
	{"GET /dvirs/{id}", TierUnlisted},
	{"GET /fleet/drivers", TierLevelTwo},
	{"POST /fleet/drivers", TierLevelOne},
	{"GET /fleet/drivers/{id}", TierLegacyOne},
	{"PATCH /fleet/drivers/{id}", TierLevelOne},
	{"GET /fleet/drivers/tachograph-files/history", TierLevelTwo},
	{"GET /fleet/dvirs/history", TierLegacyTwo},
	{"GET /fleet/hos/clocks", TierLegacyOne},
	{"GET /fleet/hos/daily-logs", TierLevelTwo},
	{"GET /fleet/hos/logs", TierLevelTwo},
	{"GET /fleet/hos/violations", TierUnlisted},
	{"GET /fleet/routes", TierLevelTwo},
	{"POST /fleet/routes", TierLevelOne},
	{"GET /fleet/routes/{id}", TierLegacyOne},
	{"PATCH /fleet/routes/{id}", TierLevelOne},
	{"DELETE /fleet/routes/{id}", TierLevelOne},
	{"GET /fleet/vehicles", TierLegacyOne},
	{"GET /fleet/vehicles/{id}", TierLegacyOne},
	{"PATCH /fleet/vehicles/{id}", TierLevelOne},
	{"GET /fleet/vehicles/locations", TierLegacyOne},
	{"GET /fleet/vehicles/locations/feed", TierLegacyTwo},
	{"GET /fleet/vehicles/locations/history", TierLegacyTwo},
	{"GET /fleet/vehicles/stats", TierLegacyTwo},
	{"GET /fleet/vehicles/stats/feed", TierLegacyTwo},
	{"GET /fleet/vehicles/stats/history", TierLegacyTwo},
	{"GET /fleet/vehicles/tachograph-files/history", TierLevelTwo},
	{"GET /form-submissions", TierUnlisted},
	{"POST /form-submissions", TierLevelOne},
	{"PATCH /form-submissions", TierLevelOne},
	{"GET /form-submissions/stream", TierUnlisted},
	{"GET /form-templates", TierUnlisted},
	{"GET /live-shares", TierUnlisted},
	{"POST /live-shares", TierLevelOne},
	{"PATCH /live-shares", TierLevelOne},
	{"DELETE /live-shares", TierLevelOne},
	{"GET /v1/fleet/messages", TierLevelTwo},
	{"POST /v1/fleet/messages", TierLevelOne},
	{"GET /webhooks", TierUnlisted},
	{"POST /webhooks", TierLevelOne},
	{"GET /webhooks/{id}", TierUnlisted},
	{"PATCH /webhooks/{id}", TierLevelOne},
	{"DELETE /webhooks/{id}", TierLevelOne},
})

func buildRoutes(specs []routeSpec) []route {
	out := make([]route, 0, len(specs))
	for _, s := range specs {
		method, template, _ := strings.Cut(s.spec, " ")
		out = append(out, route{
			method:   method,
			template: template,
			segments: splitPath(template),
			tier:     s.tier,
		})
	}
	return out
}

func Classify(method, path string) (Endpoint, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	segments := splitPath(path)

	best := -1
	bestScore := -1
	for i := range routes {
		r := &routes[i]
		if r.method != method || len(r.segments) != len(segments) {
			continue
		}
		score, ok := matchSegments(r.segments, segments)
		if ok && score > bestScore {
			best = i
			bestScore = score
		}
	}
	if best < 0 {
		return Endpoint{}, false
	}
	r := &routes[best]
	return Endpoint{Key: r.method + " " + r.template, Tier: r.tier}, true
}

func matchSegments(template, actual []string) (int, bool) {
	literal := 0
	for i := range template {
		if template[i] == wildcardSegment {
			if actual[i] == "" {
				return 0, false
			}
			continue
		}
		if template[i] != actual[i] {
			return 0, false
		}
		literal++
	}
	return literal, true
}

func splitPath(path string) []string {
	if before, _, ok := strings.Cut(path, "?"); ok {
		path = before
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
