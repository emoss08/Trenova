package sim

import (
	"log/slog"
	"net/http"
	"time"
)

type fleetView struct {
	server *Server
	live   *LiveSimulator
	snap   *fleetSnapshot
	now    time.Time
}

func (s *Server) fleetView() *fleetView {
	return &fleetView{
		server: s,
		live:   s.live,
		snap:   s.live.fleet(),
		now:    s.simNow(),
	}
}

func (v *fleetView) tagRecordView(tag Record) Record {
	id := recordID(tag)
	out := Record{keyID: id, keyName: stringValue(tag, keyName)}
	if parentID := stringValue(tag, fieldParentTagID); parentID != "" {
		out[fieldParentTagID] = parentID
		if parent, ok := v.snap.tags.byID[parentID]; ok {
			out["parentTag"] = map[string]any{
				keyID:   parentID,
				keyName: stringValue(parent, keyName),
			}
		}
	}
	if externalIDs := externalIDsOf(tag); len(externalIDs) > 0 {
		out[fieldExternalIDs] = renderExternalIDs(tag, nil)
	}
	for _, kind := range tagMemberKinds {
		members := stringListValues(tag[string(kind)])
		objects := make([]any, 0, len(members))
		for _, member := range members {
			objects = append(objects, v.taggedObject(kind, member))
		}
		out[string(kind)] = objects
	}
	return out
}

func (v *fleetView) taggedObject(kind tagMemberKind, id string) map[string]any {
	out := map[string]any{keyID: id}
	var record Record
	switch kind {
	case tagMembersAddresses:
		record = v.snap.addressByID[id]
	case tagMembersDrivers:
		record = v.snap.driverByID[id]
	case tagMembersAssets, tagMembersVehicles:
		record = v.snap.assetByID[id]
	case tagMembersMachines, tagMembersSensors:
	}
	if name := stringValue(record, keyName); name != "" {
		out[keyName] = name
	}
	return out
}

func (v *fleetView) driverRef(driverID string) map[string]any {
	out := map[string]any{keyID: driverID}
	if driver, ok := v.snap.driverByID[driverID]; ok {
		out[keyName] = stringValue(driver, keyName)
		out[fieldExternalIDs] = renderExternalIDs(driver, nil)
	}
	return out
}

func (v *fleetView) driverTiny(driverID string) map[string]any {
	out := map[string]any{keyID: driverID}
	if driver, ok := v.snap.driverByID[driverID]; ok {
		out[keyName] = stringValue(driver, keyName)
	}
	return out
}

func (v *fleetView) vehicleRef(vehicleID string) map[string]any {
	out := map[string]any{keyID: vehicleID}
	if vehicle, ok := v.snap.assetByID[vehicleID]; ok {
		out[keyName] = stringValue(vehicle, keyName)
		out[fieldExternalIDs] = renderExternalIDs(vehicle, vehicleAutoExternalIDs)
	}
	return out
}

func (v *fleetView) assetsOfType(assetType string) []Record {
	out := make([]Record, 0, len(v.snap.assets))
	for _, asset := range v.snap.assets {
		if stringValue(asset, keyType) == assetType {
			out = append(out, asset)
		}
	}
	return out
}

func (v *fleetView) vehicleRecords() []Record {
	return v.assetsOfType(assetTypeVehicle)
}

func (s *Server) respondPage(
	writer http.ResponseWriter,
	request *http.Request,
	records []Record,
	signature string,
) {
	page, pagination, err := paginate(records, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{keyData: recordsAsAny(page), keyPagination: pagination}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}

func (s *Server) respondJSONStatus(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	signature string,
	payload any,
) {
	profile := s.profileFromContext(request.Context())
	applied := s.scenarios.Apply(profile, signature, payload)
	if err := writeJSON(writer, status, applied); err != nil {
		s.logger.Error("failed to write JSON response", slog.String("error", err.Error()))
	}
}
