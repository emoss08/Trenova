package sim

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	messageMaxTextLength     = 2500
	messageDefaultDuration   = 24 * time.Hour
	messageSenderDispatch    = "dispatch"
	messageSenderDriver      = "driver"
	messageDispatchName      = "Dispatch"
	messageDelayRate         = 0.4
	messageReadDelayMin      = time.Minute
	messageReadDelaySpread   = 14 * time.Minute
	fieldMessageDriverID     = "driverId"
	fieldMessageText         = "text"
	fieldMessageIsRead       = "isRead"
	fieldMessageSender       = "sender"
	fieldMessageSentAtMs     = "sentAtMs"
	fieldMessageDriverIDList = "driverIds"
)

func (s *Server) registerMessageRoutes() {
	s.mux.HandleFunc("GET /v1/fleet/messages", s.handleMessageList)
	s.mux.HandleFunc("POST /v1/fleet/messages", s.handleMessageCreate)
}

func parseInt64Param(request *http.Request, name string) (int64, bool, error) {
	raw := queryValue(request, name)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false, invalidParameter(name, "must be an integer number of milliseconds")
	}
	return value, true, nil
}

func (s *Server) handleMessageList(writer http.ResponseWriter, request *http.Request) {
	now := s.simNow()
	endMs, hasEnd, err := parseInt64Param(request, "endMs")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if !hasEnd {
		endMs = now.UnixMilli()
	}
	durationMs, hasDuration, err := parseInt64Param(request, "durationMs")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if !hasDuration {
		durationMs = messageDefaultDuration.Milliseconds()
	}
	if durationMs < 0 {
		s.writeError(writer, invalidParameter("durationMs", "must be greater than or equal to 0"))
		return
	}
	if endMs < 0 {
		s.writeError(writer, invalidParameter("endMs", "must be greater than or equal to 0"))
		return
	}
	startMs := endMs - min(durationMs, endMs)

	stored, err := s.store.List(ResourceMessages)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	messages := make([]Record, 0, len(stored))
	for _, record := range stored {
		rendered, ok := renderStoredMessage(record, now)
		if ok {
			messages = append(messages, rendered)
		}
	}
	if s.live != nil {
		messages = append(messages, s.live.GeneratedDriverMessages(
			now,
			time.UnixMilli(startMs).UTC(),
			time.UnixMilli(endMs).UTC(),
		)...)
	}
	data := make([]any, 0, len(messages))
	selected := make([]Record, 0, len(messages))
	for _, message := range messages {
		sentAt, _ := int64Value(message[fieldMessageSentAtMs])
		if sentAt >= startMs && sentAt <= endMs {
			selected = append(selected, message)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left, _ := int64Value(selected[i][fieldMessageSentAtMs])
		right, _ := int64Value(selected[j][fieldMessageSentAtMs])
		if left != right {
			return left > right
		}
		leftDriver, _ := int64Value(selected[i][fieldMessageDriverID])
		rightDriver, _ := int64Value(selected[j][fieldMessageDriverID])
		return leftDriver < rightDriver
	})
	for _, message := range selected {
		data = append(data, map[string]any(message))
	}
	s.respondJSON(writer, request, requestSignature(request)+"|message-list", map[string]any{keyData: data})
}

func renderStoredMessage(record Record, now time.Time) (Record, bool) {
	driverID, okDriver := int64Value(record[fieldMessageDriverID])
	sentAt, okSent := int64Value(record[fieldMessageSentAtMs])
	if !okDriver || !okSent {
		return nil, false
	}
	sender, _ := anyAsMap(record[fieldMessageSender])
	isRead, _ := record[fieldMessageIsRead].(bool)
	text := stringOf(record[fieldMessageText])
	if !isRead {
		isRead = messageReadBy(now, sentAt, strconv.FormatInt(driverID, 10), text)
	}
	return Record{
		fieldMessageDriverID: driverID,
		fieldMessageText:     text,
		fieldMessageIsRead:   isRead,
		fieldMessageSender: map[string]any{
			keyName: stringutils.FirstNonEmptyTrimmed(stringOf(sender[keyName]), messageDispatchName),
			keyType: stringutils.FirstNonEmptyTrimmed(stringOf(sender[keyType]), messageSenderDispatch),
		},
		fieldMessageSentAtMs: sentAt,
	}, true
}

func messageReadBy(now time.Time, sentAtMs int64, parts ...string) bool {
	fraction := float64(fnvHash64("message-read|"+strconv.FormatInt(sentAtMs, 10)+"|"+strings.Join(parts, "|"))%10000) / 10000
	delay := messageReadDelayMin + time.Duration(fraction*float64(messageReadDelaySpread))
	return !now.Before(time.UnixMilli(sentAtMs).Add(delay))
}

type messageCreateRequest struct {
	DriverIDs []int64
	Text      string
}

func parseMessageCreate(body Record) (messageCreateRequest, error) {
	rawText, present := body[fieldMessageText]
	if !present || rawText == nil {
		return messageCreateRequest{}, invalidField(fieldMessageText, "is required")
	}
	text, ok := rawText.(string)
	if !ok {
		return messageCreateRequest{}, invalidField(fieldMessageText, "must be a string")
	}
	if strings.TrimSpace(text) == "" {
		return messageCreateRequest{}, invalidField(fieldMessageText, "must not be empty")
	}
	if utf8.RuneCountInString(text) > messageMaxTextLength {
		return messageCreateRequest{}, invalidField(
			fieldMessageText,
			fmt.Sprintf("must be at most %d characters", messageMaxTextLength),
		)
	}
	rawIDs, present := body[fieldMessageDriverIDList]
	if !present || rawIDs == nil {
		return messageCreateRequest{}, invalidField(fieldMessageDriverIDList, "is required")
	}
	items, ok := rawIDs.([]any)
	if !ok {
		return messageCreateRequest{}, invalidField(fieldMessageDriverIDList, "must be an array of driver IDs")
	}
	if len(items) == 0 {
		return messageCreateRequest{}, invalidField(fieldMessageDriverIDList, "must contain at least one driver ID")
	}
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for idx, item := range items {
		number, isNumber := item.(float64)
		if !isNumber || math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) ||
			number <= 0 || number > 1<<53 {
			return messageCreateRequest{}, invalidField(
				fmt.Sprintf("%s[%d]", fieldMessageDriverIDList, idx),
				"must be a positive integer driver ID",
			)
		}
		id := int64(number)
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return messageCreateRequest{DriverIDs: ids, Text: text}, nil
}

func (s *Server) handleMessageCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	parsed, err := parseMessageCreate(body)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	unknown := make([]string, 0)
	for _, id := range parsed.DriverIDs {
		driverID := strconv.FormatInt(id, 10)
		driver, exists := view.snap.driverByID[driverID]
		if !exists || driverDeactivated(driver) {
			unknown = append(unknown, driverID)
		}
	}
	if len(unknown) > 0 {
		s.writeError(writer, fmt.Errorf(
			"%w: driverIds contains IDs that do not match an active driver: %s",
			ErrInvalidBody,
			strings.Join(unknown, ", "),
		))
		return
	}
	nowMillis := s.simNow().UnixMilli()
	created := make([]any, 0, len(parsed.DriverIDs))
	persisted := make([]Record, 0, len(parsed.DriverIDs))
	for _, id := range parsed.DriverIDs {
		created = append(created, map[string]any{fieldMessageDriverID: id, fieldMessageText: parsed.Text})
		persisted = append(persisted, Record{
			fieldMessageDriverID: id,
			fieldMessageText:     parsed.Text,
			fieldMessageIsRead:   false,
			fieldMessageSender:   map[string]any{keyName: messageDispatchName, keyType: messageSenderDispatch},
			fieldMessageSentAtMs: nowMillis,
		})
	}
	if err = s.store.AppendMessages(persisted); err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|message-create", map[string]any{keyData: created})
}

func (l *LiveSimulator) GeneratedDriverMessages(now, start, end time.Time) []Record {
	if end.After(now) {
		end = now
	}
	if start.Before(telemetryEpoch) {
		start = telemetryEpoch
	}
	if end.Before(start) {
		return []Record{}
	}
	snapshot := l.fleet()
	roster := l.loadDriverRoster()
	routes := routeRefsByDriver(l.routeRefs(now))
	ids := make([]string, 0, len(roster))
	for driverID := range roster {
		if driver, ok := snapshot.driverByID[driverID]; ok && !driverDeactivated(driver) {
			ids = append(ids, driverID)
		}
	}
	sort.Strings(ids)
	startDay := start.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	endDay := end.Truncate(24 * time.Hour)
	out := make([]Record, 0, len(ids)*3)
	for _, driverID := range ids {
		numericDriver, err := strconv.ParseInt(driverID, 10, 64)
		if err != nil {
			continue
		}
		entry := roster[driverID]
		route := routes[driverID]
		for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
			for _, message := range l.driverDayMessages(driverID, entry, &route, day) {
				if message.At.Before(start) || message.At.After(end) {
					continue
				}
				sentAt := message.At.UnixMilli()
				out = append(out, Record{
					fieldMessageDriverID: numericDriver,
					fieldMessageText:     message.Text,
					fieldMessageIsRead:   messageReadBy(now, sentAt, driverID, message.Text),
					fieldMessageSender: map[string]any{
						keyName: stringutils.FirstNonEmptyTrimmed(entry.Name, driverID),
						keyType: messageSenderDriver,
					},
					fieldMessageSentAtMs: sentAt,
				})
			}
		}
	}
	return out
}

type driverMessage struct {
	At   time.Time
	Text string
}

func (l *LiveSimulator) driverDayMessages(
	driverID string,
	entry driverRoster,
	route *routeRef,
	day time.Time,
) []driverMessage {
	dayCtx := l.buildDailyEventContext(driverID, strings.TrimSpace(entry.VehicleID), day)
	span := dayCtx.DrivingEnd.Sub(dayCtx.DrivingStart)
	if span <= 0 {
		return nil
	}
	dayKey := day.Format(dateLayout)
	origin, destination := "the shipper", "the consignee"
	if stop, ok := route.firstStop(); ok && stop.Name != "" {
		origin = stop.Name
	}
	if stop, ok := route.lastStop(); ok && stop.Name != "" {
		destination = stop.Name
	}
	out := make([]driverMessage, 0, 3)
	pickupOffset := time.Duration((32 + 20*l.hashFraction("msg|pickup", driverID, dayKey)) * float64(time.Minute))
	out = append(out, driverMessage{
		At:   dayCtx.DrivingStart.Add(pickupOffset).Truncate(time.Second),
		Text: fmt.Sprintf("Loaded at %s, rolling to %s.", origin, destination),
	})
	if l.hashFraction("msg|delay", driverID, dayKey) < messageDelayRate {
		fraction := 0.35 + 0.3*l.hashFraction("msg|delay-at", driverID, dayKey)
		minutes := 15 + 5*int(math.Floor(7*l.hashFraction("msg|delay-minutes", driverID, dayKey)))
		out = append(out, driverMessage{
			At:   dayCtx.DrivingStart.Add(time.Duration(fraction * float64(span))).Truncate(time.Second),
			Text: fmt.Sprintf("Running about %d minutes behind schedule because of traffic.", minutes),
		})
	}
	deliveredOffset := time.Duration((2 + 6*l.hashFraction("msg|delivered", driverID, dayKey)) * float64(time.Minute))
	out = append(out, driverMessage{
		At:   dayCtx.DrivingEnd.Add(deliveredOffset).Truncate(time.Second),
		Text: fmt.Sprintf("Delivered at %s. Empty and available.", destination),
	})
	return out
}
