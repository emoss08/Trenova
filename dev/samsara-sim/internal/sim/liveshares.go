package sim

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	liveShareTypeAll                = "all"
	liveShareTypeAssetsLocation     = "assetsLocation"
	liveShareTypeAssetsNearLocation = "assetsNearLocation"
	liveShareTypeAssetsOnRoute      = "assetsOnRoute"
	fieldLiveShareURL               = "liveSharingUrl"
	fieldExpiresAtTime              = "expiresAtTime"
	fieldAssetID                    = "assetId"
	fieldAddressID                  = "addressId"
	fieldRecurringRouteID           = "recurringRouteId"
	maxLiveShareTextLength          = 255
	liveShareURLBase                = "https://cloud.samsara.com/o/"
	labelLiveShare                  = "live sharing link"
)

var (
	liveShareTypes = []string{
		liveShareTypeAssetsLocation,
		liveShareTypeAssetsNearLocation,
		liveShareTypeAssetsOnRoute,
	}
	liveShareFilterTypes = []string{
		liveShareTypeAll,
		liveShareTypeAssetsLocation,
		liveShareTypeAssetsNearLocation,
		liveShareTypeAssetsOnRoute,
	}
	liveShareConfigFields = map[string]string{
		liveShareTypeAssetsLocation:     "assetsLocationLinkConfig",
		liveShareTypeAssetsNearLocation: "assetsNearLocationLinkConfig",
		liveShareTypeAssetsOnRoute:      "assetsOnRouteLinkConfig",
	}
	liveShareURLSegments = map[string]string{
		liveShareTypeAssetsLocation:     "asset",
		liveShareTypeAssetsNearLocation: "address",
		liveShareTypeAssetsOnRoute:      "route",
	}
)

func (s *Server) registerLiveShareRoutes() {
	s.mux.HandleFunc("GET /live-shares", s.handleLiveShareList)
	s.mux.HandleFunc("POST /live-shares", s.handleLiveShareCreate)
	s.mux.HandleFunc("PATCH /live-shares", s.handleLiveSharePatch)
	s.mux.HandleFunc("DELETE /live-shares", s.handleLiveShareDelete)
}

func liveShareBodyRules(mode bodyMode) []fieldRule {
	rules := []fieldRule{
		{Name: keyDescription, Kind: kindString, MaxLen: maxLiveShareTextLength},
		{Name: fieldExpiresAtTime, Kind: kindTime, Nullable: true},
		{
			Name:     keyName,
			Kind:     kindString,
			Required: true,
			MinLen:   1,
			MaxLen:   maxLiveShareTextLength,
			Check:    nonBlank(keyName),
		},
	}
	if mode == bodyCreate {
		rules = append(
			rules,
			fieldRule{Name: keyType, Kind: kindString, Required: true, Enum: liveShareTypes},
			fieldRule{
				Name: liveShareConfigFields[liveShareTypeAssetsLocation],
				Kind: kindObject,
				Fields: []fieldRule{
					{Name: fieldAssetID, Kind: kindString, MinLen: 1},
					{Name: "location", Kind: kindObject, Fields: []fieldRule{
						{Name: fieldFormattedAddress, Kind: kindString, Required: true, MinLen: 1},
						{
							Name:     keyLatitude,
							Kind:     kindNumber,
							Required: true,
							Check: coordinateRange(
								"assetsLocationLinkConfig.location.latitude",
								90,
							),
						},
						{
							Name:     keyLongitude,
							Kind:     kindNumber,
							Required: true,
							Check: coordinateRange(
								"assetsLocationLinkConfig.location.longitude",
								180,
							),
						},
						{Name: keyName, Kind: kindString, Required: true, MinLen: 1},
					}},
					{Name: fieldTagIDs, Kind: kindStringList},
				},
			},
			fieldRule{
				Name: liveShareConfigFields[liveShareTypeAssetsNearLocation],
				Kind: kindObject,
				Fields: []fieldRule{
					{Name: fieldAddressID, Kind: kindString, Required: true, MinLen: 1},
				},
			},
			fieldRule{
				Name: liveShareConfigFields[liveShareTypeAssetsOnRoute],
				Kind: kindObject,
				Fields: []fieldRule{
					{Name: fieldRecurringRouteID, Kind: kindString, Required: true, MinLen: 1},
				},
			},
		)
	}
	return rules
}

func liveShareURL(shareType, id string) string {
	segment := liveShareURLSegments[shareType]
	token := deterministicToken(liveShareIDLength, "live-share-url", id)
	return liveShareURLBase + strconv.FormatInt(
		webhookOrgID,
		10,
	) + "/fleet/viewer/" + segment + "/" + token
}

func liveShareExpired(record Record, now time.Time) bool {
	raw := stringValue(record, fieldExpiresAtTime)
	if raw == "" {
		return false
	}
	expires, err := parseRFC3339(raw)
	return err == nil && !expires.After(now)
}

func validateLiveShareExpiry(body Record, now time.Time) error {
	raw, ok := body[fieldExpiresAtTime].(string)
	if !ok {
		return nil
	}
	expires, err := parseRFC3339(raw)
	if err != nil {
		return invalidField(fieldExpiresAtTime, "must be an RFC 3339 timestamp")
	}
	if !expires.After(now) {
		return invalidField(fieldExpiresAtTime, "can't be set in the past")
	}
	return nil
}

func resolveLiveShareConfigTx(tx *storeTx, body Record) (Record, error) {
	shareType := stringOf(body[keyType])
	expected := liveShareConfigFields[shareType]
	for _, other := range liveShareConfigFields {
		if other != expected && body[other] != nil {
			return nil, invalidField(other, fmt.Sprintf("does not apply to type %q", shareType))
		}
	}
	config := mapOf(body[expected])
	if config == nil {
		return nil, invalidField(expected, fmt.Sprintf("is required for type %q", shareType))
	}
	if shareType == liveShareTypeAssetsOnRoute && body[keyDescription] != nil {
		return nil, invalidField(keyDescription, "does not apply to assetsOnRoute links")
	}
	resolved := Record{}
	switch shareType {
	case liveShareTypeAssetsLocation:
		assetID := stringOf(config[fieldAssetID])
		tagIDs := stringListValues(config[fieldTagIDs])
		if (assetID == "") == (len(tagIDs) == 0) {
			return nil, invalidField(expected, "must set exactly one of assetId or tagIds")
		}
		if assetID != "" {
			if !recordExists(tx.records(ResourceAssets))(assetID) {
				return nil, missingReference("asset", assetID)
			}
			resolved[fieldAssetID] = assetID
		}
		if len(tagIDs) > 0 {
			exists := recordExists(tx.records(ResourceTags))
			for _, tagID := range tagIDs {
				if !exists(tagID) {
					return nil, missingReference("tag", tagID)
				}
			}
			resolved[fieldTagIDs] = stringsAsAny(tagIDs)
		}
		if location := mapOf(config["location"]); location != nil {
			resolved["location"] = cloneMap(location)
		}
	case liveShareTypeAssetsNearLocation:
		ref := stringOf(config[fieldAddressID])
		address, idx := findRecordByRef(tx.records(ResourceAddresses), ref, nil)
		if idx < 0 {
			return nil, missingReference(labelAddress, ref)
		}
		resolved[fieldAddressID] = recordID(address)
	case liveShareTypeAssetsOnRoute:
		routeID := stringOf(config[fieldRecurringRouteID])
		if !recordExists(tx.records(ResourceRoutes))(routeID) {
			return nil, missingReference("recurring route", routeID)
		}
		resolved[fieldRecurringRouteID] = routeID
	}
	return resolved, nil
}

func liveShareView(record Record, tags *tagIndex) Record {
	shareType := stringValue(record, keyType)
	id := recordID(record)
	out := Record{
		keyID:             id,
		keyName:           stringValue(record, keyName),
		keyType:           shareType,
		fieldLiveShareURL: stringValue(record, fieldLiveShareURL),
	}
	if out[fieldLiveShareURL] == "" {
		out[fieldLiveShareURL] = liveShareURL(shareType, id)
	}
	if description := stringValue(record, keyDescription); description != "" {
		out[keyDescription] = description
	}
	if expires := stringValue(record, fieldExpiresAtTime); expires != "" {
		out[fieldExpiresAtTime] = expires
	}
	field, ok := liveShareConfigFields[shareType]
	if !ok {
		return out
	}
	config := mapOf(record[field])
	if config == nil {
		return out
	}
	rendered := map[string]any{}
	for key, value := range config {
		if key == fieldTagIDs {
			continue
		}
		rendered[key] = cloneAny(value)
	}
	if tagIDs := stringListValues(config[fieldTagIDs]); len(tagIDs) > 0 {
		tiny := make([]any, 0, len(tagIDs))
		for _, tagID := range tagIDs {
			if tag := tags.tiny(tagID); tag != nil {
				tiny = append(tiny, tag)
			}
		}
		rendered[keyTags] = tiny
	}
	out[field] = rendered
	return out
}

func (s *Server) handleLiveShareList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	filterType := liveShareTypeAll
	if raw := strings.TrimSpace(values.Get(keyType)); raw != "" {
		if !containsString(liveShareFilterTypes, raw) {
			s.writeError(
				writer,
				invalidParameter(
					keyType,
					"must be one of "+strings.Join(liveShareFilterTypes, ", "),
				),
			)
			return
		}
		filterType = raw
	}
	ids := csvQueryValues(values, "ids")
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	records, err := s.store.List(ResourceLiveShares)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	tags := s.fleetView().snap.tags
	views := make([]Record, 0, len(records))
	for _, record := range records {
		if liveShareExpired(record, now) ||
			(filterType != liveShareTypeAll && stringValue(record, keyType) != filterType) {
			continue
		}
		if len(wanted) > 0 {
			if _, ok := wanted[recordID(record)]; !ok {
				continue
			}
		}
		views = append(views, liveShareView(record, tags))
	}
	s.respondPage(writer, request, views, "|live-share-list")
}

func (s *Server) handleLiveShareCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, liveShareBodyRules(bodyCreate), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	if err = validateLiveShareExpiry(sanitized, now); err != nil {
		s.writeError(writer, err)
		return
	}
	var created Record
	err = s.store.Transact(func(tx *storeTx) error {
		config, configErr := resolveLiveShareConfigTx(tx, sanitized)
		if configErr != nil {
			return configErr
		}
		shareType := stringOf(sanitized[keyType])
		record := Record{keyName: sanitized[keyName], keyType: shareType}
		record[liveShareConfigFields[shareType]] = config
		for _, field := range []string{keyDescription, fieldExpiresAtTime} {
			if value, ok := sanitized[field]; ok && value != nil {
				record[field] = value
			}
		}
		inserted, insertErr := tx.insert(ResourceLiveShares, record, now)
		if insertErr != nil {
			return insertErr
		}
		inserted[fieldLiveShareURL] = liveShareURL(shareType, recordID(inserted))
		created = inserted
		return nil
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondLiveShare(writer, request, created, "|live-share-create")
}

func (s *Server) handleLiveSharePatch(writer http.ResponseWriter, request *http.Request) {
	id, err := queryID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, liveShareBodyRules(bodyPatch), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if _, ok := body[keyName]; !ok {
		s.writeError(writer, invalidField(keyName, "is required"))
		return
	}
	now := s.simNow()
	if err = validateLiveShareExpiry(sanitized, now); err != nil {
		s.writeError(writer, err)
		return
	}
	var updated Record
	err = s.store.Transact(func(tx *storeTx) error {
		current, idx := tx.find(ResourceLiveShares, id)
		if idx < 0 || liveShareExpired(current, now) {
			return notFound(labelLiveShare, id)
		}
		if stringValue(current, keyType) == liveShareTypeAssetsOnRoute &&
			sanitized[keyDescription] != nil {
			return invalidField(keyDescription, "does not apply to assetsOnRoute links")
		}
		var updateErr error
		updated, updateErr = tx.update(ResourceLiveShares, id, func(record Record) error {
			for _, field := range []string{keyName, keyDescription, fieldExpiresAtTime} {
				value, ok := sanitized[field]
				if !ok {
					continue
				}
				if value == nil {
					delete(record, field)
					continue
				}
				record[field] = value
			}
			return nil
		}, now)
		return updateErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondLiveShare(writer, request, updated, "|live-share-patch")
}

func (s *Server) handleLiveShareDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := queryID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	err = s.store.Transact(func(tx *storeTx) error {
		current, idx := tx.find(ResourceLiveShares, id)
		if idx < 0 || liveShareExpired(current, now) {
			return notFound(labelLiveShare, id)
		}
		return tx.remove(ResourceLiveShares, id)
	})
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			s.writeError(writer, notFound(labelLiveShare, id))
			return
		}
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) respondLiveShare(
	writer http.ResponseWriter,
	request *http.Request,
	record Record,
	signature string,
) {
	payload := map[string]any{keyData: liveShareView(record, s.fleetView().snap.tags)}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}
