package sim

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

type tagMemberKind string

const (
	tagMembersAddresses tagMemberKind = "addresses"
	tagMembersAssets    tagMemberKind = "assets"
	tagMembersDrivers   tagMemberKind = "drivers"
	tagMembersMachines  tagMemberKind = "machines"
	tagMembersSensors   tagMemberKind = "sensors"
	tagMembersVehicles  tagMemberKind = "vehicles"

	fieldParentTagID  = "parentTagId"
	paramTagIDs       = "tagIds"
	paramParentTagIDs = "parentTagIds"
	maxTagNameLength  = 191
)

var tagMemberKinds = []tagMemberKind{
	tagMembersAddresses,
	tagMembersAssets,
	tagMembersDrivers,
	tagMembersMachines,
	tagMembersSensors,
	tagMembersVehicles,
}

var driverGroupTagFields = []string{keyPeerGroupTagID, keyVehicleGroupTagID, keyTrailerGroupTagID}

type tagIndex struct {
	byID     map[string]Record
	order    []string
	children map[string][]string
	members  map[tagMemberKind]map[string][]string
}

type tagFilter struct {
	active  bool
	allowed map[string]struct{}
}

func newTagIndex(tags []Record) *tagIndex {
	index := &tagIndex{
		byID:     make(map[string]Record, len(tags)),
		order:    make([]string, 0, len(tags)),
		children: make(map[string][]string, len(tags)),
		members:  make(map[tagMemberKind]map[string][]string, len(tagMemberKinds)),
	}
	for _, kind := range tagMemberKinds {
		index.members[kind] = map[string][]string{}
	}
	for _, tag := range tags {
		id := recordID(tag)
		if id == "" {
			continue
		}
		index.byID[id] = tag
		index.order = append(index.order, id)
		if parent := stringValue(tag, fieldParentTagID); parent != "" {
			index.children[parent] = append(index.children[parent], id)
		}
		for _, kind := range tagMemberKinds {
			for _, member := range stringListValues(tag[string(kind)]) {
				index.members[kind][member] = append(index.members[kind][member], id)
			}
		}
	}
	return index
}

func (t *tagIndex) has(id string) bool {
	_, ok := t.byID[strings.TrimSpace(id)]
	return ok
}

func (t *tagIndex) tagIDsFor(kind tagMemberKind, entityID string) []string {
	return t.members[kind][entityID]
}

func (t *tagIndex) tiny(id string) map[string]any {
	tag, ok := t.byID[id]
	if !ok {
		return nil
	}
	out := map[string]any{keyID: id, keyName: stringValue(tag, keyName)}
	if parent := stringValue(tag, fieldParentTagID); parent != "" {
		out[fieldParentTagID] = parent
	}
	return out
}

func (t *tagIndex) tinyTags(kind tagMemberKind, entityID string) []any {
	ids := t.tagIDsFor(kind, entityID)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if tiny := t.tiny(id); tiny != nil {
			out = append(out, tiny)
		}
	}
	return out
}

func (t *tagIndex) descendants(id string) []string {
	out := []string{}
	stack := []string{id}
	seen := map[string]struct{}{}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, visited := seen[current]; visited {
			continue
		}
		seen[current] = struct{}{}
		out = append(out, current)
		stack = append(stack, t.children[current]...)
	}
	return out
}

func (t *tagIndex) isAncestor(candidate, of string) bool {
	current := of
	seen := map[string]struct{}{}
	for current != "" {
		if current == candidate {
			return true
		}
		if _, loop := seen[current]; loop {
			return true
		}
		seen[current] = struct{}{}
		tag, ok := t.byID[current]
		if !ok {
			return false
		}
		current = stringValue(tag, fieldParentTagID)
	}
	return false
}

func (t *tagIndex) filter(tagIDs, parentTagIDs []string) tagFilter {
	if len(tagIDs) == 0 && len(parentTagIDs) == 0 {
		return tagFilter{}
	}
	allowed := make(map[string]struct{}, len(tagIDs)+len(parentTagIDs))
	for _, id := range tagIDs {
		allowed[strings.TrimSpace(id)] = struct{}{}
	}
	for _, id := range parentTagIDs {
		clean := strings.TrimSpace(id)
		if !t.has(clean) {
			continue
		}
		for _, descendant := range t.descendants(clean) {
			allowed[descendant] = struct{}{}
		}
	}
	return tagFilter{active: true, allowed: allowed}
}

func (f tagFilter) matches(index *tagIndex, kind tagMemberKind, entityID string) bool {
	if !f.active {
		return true
	}
	for _, id := range index.tagIDsFor(kind, entityID) {
		if _, ok := f.allowed[id]; ok {
			return true
		}
	}
	return false
}

func standardTagParams(values url.Values) (tagIDs, parentTagIDs []string) {
	return tagFilterParams(values, paramTagIDs, paramParentTagIDs)
}

func tagFilterParams(
	values url.Values,
	tagParam, parentParam string,
) (tagIDs, parentTagIDs []string) {
	return csvQueryValues(values, tagParam), csvQueryValues(values, parentParam)
}

func assetTagKind(record Record) tagMemberKind {
	if stringValue(record, keyType) == assetTypeVehicle {
		return tagMembersVehicles
	}
	return tagMembersAssets
}

func setEntityTagsTx(tx *storeTx, kind tagMemberKind, entityID string, tagIDs []string) error {
	tags := tx.records(ResourceTags)
	known := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		known[recordID(tag)] = struct{}{}
	}
	wanted := make(map[string]struct{}, len(tagIDs))
	for _, id := range tagIDs {
		clean := strings.TrimSpace(id)
		if _, ok := known[clean]; !ok {
			return missingReference("tag", clean)
		}
		wanted[clean] = struct{}{}
	}
	for idx, tag := range tags {
		_, include := wanted[recordID(tag)]
		members := stringListValues(tag[string(kind)])
		present := slices.Contains(members, entityID)
		if include == present {
			continue
		}
		next := cloneRecord(tag)
		if include {
			members = append(members, entityID)
		} else {
			members = slices.DeleteFunc(
				members,
				func(member string) bool { return member == entityID },
			)
		}
		setTagMembers(next, kind, members)
		tags[idx] = next
	}
	return nil
}

func moveEntityTagKindTx(tx *storeTx, from, to tagMemberKind, entityID string) {
	if from == to {
		return
	}
	tags := tx.records(ResourceTags)
	for idx, tag := range tags {
		members := stringListValues(tag[string(from)])
		if !slices.Contains(members, entityID) {
			continue
		}
		next := cloneRecord(tag)
		setTagMembers(
			next,
			from,
			slices.DeleteFunc(members, func(member string) bool { return member == entityID }),
		)
		target := stringListValues(next[string(to)])
		if !slices.Contains(target, entityID) {
			setTagMembers(next, to, append(target, entityID))
		}
		tags[idx] = next
	}
}

func removeEntityFromTagsTx(tx *storeTx, kind tagMemberKind, entityID string) {
	tags := tx.records(ResourceTags)
	for idx, tag := range tags {
		members := stringListValues(tag[string(kind)])
		if !slices.Contains(members, entityID) {
			continue
		}
		next := cloneRecord(tag)
		setTagMembers(
			next,
			kind,
			slices.DeleteFunc(members, func(member string) bool { return member == entityID }),
		)
		tags[idx] = next
	}
}

func setTagMembers(tag Record, kind tagMemberKind, members []string) {
	if len(members) == 0 {
		delete(tag, string(kind))
		return
	}
	sorted := append([]string(nil), members...)
	sort.Strings(sorted)
	values := make([]any, 0, len(sorted))
	for _, member := range sorted {
		values = append(values, member)
	}
	tag[string(kind)] = values
}

func (s *Server) registerTagRoutes() {
	s.mux.HandleFunc("GET /tags", s.handleTagList)
	s.mux.HandleFunc("POST /tags", s.handleTagCreate)
	s.mux.HandleFunc("GET /tags/{id}", s.handleTagGet)
	s.mux.HandleFunc("PATCH /tags/{id}", s.handleTagPatch)
	s.mux.HandleFunc("PUT /tags/{id}", s.handleTagReplace)
	s.mux.HandleFunc("DELETE /tags/{id}", s.handleTagDelete)
}

func tagMemberRules() []fieldRule {
	rules := make([]fieldRule, 0, len(tagMemberKinds))
	for _, kind := range tagMemberKinds {
		rules = append(rules, fieldRule{Name: string(kind), Kind: kindStringList})
	}
	return rules
}

func tagBodyRules(mode bodyMode, withExternalIDs bool) []fieldRule {
	rules := []fieldRule{
		{
			Name:     keyName,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   maxTagNameLength,
		},
		{Name: fieldParentTagID, Kind: kindString, Nullable: true},
	}
	if withExternalIDs {
		rules = append(
			rules,
			fieldRule{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		)
	}
	return append(rules, tagMemberRules()...)
}

func (s *Server) handleTagList(writer http.ResponseWriter, request *http.Request) {
	view := s.fleetView()
	tags := make([]Record, 0, len(view.snap.tags.order))
	for _, id := range view.snap.tags.order {
		tags = append(tags, view.tagRecordView(view.snap.tags.byID[id]))
	}
	s.respondPage(writer, request, tags, "|tag-list")
}

func (s *Server) handleTagGet(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	tag, idx := findRecordByRef(view.snap.tagRecords, ref, tagAutoExternalIDs)
	if idx < 0 {
		s.writeError(writer, notFound("tag", ref))
		return
	}
	payload := map[string]any{keyData: view.tagRecordView(tag)}
	s.respondJSON(writer, request, requestSignature(request)+"|tag-get", payload)
}

func (s *Server) handleTagCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, tagBodyRules(bodyCreate, true), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	createdID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return createTagTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondTag(writer, request, createdID, "|tag-create")
}

func (s *Server) handleTagPatch(writer http.ResponseWriter, request *http.Request) {
	s.handleTagUpdate(writer, request, false)
}

func (s *Server) handleTagReplace(writer http.ResponseWriter, request *http.Request) {
	s.handleTagUpdate(writer, request, true)
}

func (s *Server) handleTagUpdate(writer http.ResponseWriter, request *http.Request, replace bool) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, tagBodyRules(bodyPatch, !replace), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	tagID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return updateTagTx(tx, ref, sanitized, replace, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondTag(writer, request, tagID, "|tag-update")
}

func (s *Server) handleTagDelete(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	err = s.store.Transact(func(tx *storeTx) error {
		return deleteTagTx(tx, ref)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func createTagTx(tx *storeTx, body Record, now time.Time) (string, error) {
	if err := validateTagWriteTx(tx, "", body); err != nil {
		return "", err
	}
	record := Record{}
	applyTagFields(record, body, false)
	created, err := tx.insert(ResourceTags, record, now)
	if err != nil {
		return "", err
	}
	return recordID(created), nil
}

func updateTagTx(
	tx *storeTx,
	ref string,
	body Record,
	replace bool,
	now time.Time,
) (string, error) {
	current, idx := findRecordByRef(tx.records(ResourceTags), ref, tagAutoExternalIDs)
	if idx < 0 {
		return "", notFound("tag", ref)
	}
	tagID := recordID(current)
	if err := validateTagWriteTx(tx, tagID, body); err != nil {
		return "", err
	}
	_, err := tx.update(ResourceTags, tagID, func(record Record) error {
		applyTagFields(record, body, replace)
		return nil
	}, now)
	return tagID, err
}

func deleteTagTx(tx *storeTx, ref string) error {
	tags := tx.records(ResourceTags)
	current, idx := findRecordByRef(tags, ref, tagAutoExternalIDs)
	if idx < 0 {
		return notFound("tag", ref)
	}
	removed := make(map[string]struct{})
	for _, id := range newTagIndex(tags).descendants(recordID(current)) {
		removed[id] = struct{}{}
	}
	kept := make([]Record, 0, len(tags)-len(removed))
	for _, tag := range tags {
		if _, drop := removed[recordID(tag)]; !drop {
			kept = append(kept, tag)
		}
	}
	if err := tx.replace(ResourceTags, kept); err != nil {
		return err
	}
	clearDriverGroupTagsTx(tx, removed)
	return nil
}

func clearDriverGroupTagsTx(tx *storeTx, removed map[string]struct{}) {
	drivers := tx.records(ResourceDrivers)
	for idx, driver := range drivers {
		var next Record
		for _, field := range driverGroupTagFields {
			if _, gone := removed[stringValue(driver, field)]; !gone {
				continue
			}
			if next == nil {
				next = cloneRecord(driver)
			}
			delete(next, field)
		}
		if next != nil {
			drivers[idx] = next
		}
	}
}

func (s *Server) respondTag(
	writer http.ResponseWriter,
	request *http.Request,
	tagID string,
	signature string,
) {
	view := s.fleetView()
	tag, ok := view.snap.tags.byID[tagID]
	if !ok {
		s.writeError(writer, notFound("tag", tagID))
		return
	}
	payload := map[string]any{keyData: view.tagRecordView(tag)}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}

func validateTagWriteTx(tx *storeTx, selfID string, body Record) error {
	tags := tx.records(ResourceTags)
	index := newTagIndex(tags)
	parentID := ""
	if raw, ok := body[fieldParentTagID]; ok && raw != nil {
		parentID = strings.TrimSpace(stringOf(raw))
	} else if !ok && selfID != "" {
		parentID = stringValue(index.byID[selfID], fieldParentTagID)
	}
	if parentID != "" {
		if !index.has(parentID) {
			return missingReference("parent tag", parentID)
		}
		if selfID != "" && index.isAncestor(selfID, parentID) {
			return fmt.Errorf(
				"%w: tag %s cannot be nested under itself or one of its descendants",
				ErrTagCycle,
				selfID,
			)
		}
	}
	name := ""
	if raw, ok := body[keyName].(string); ok {
		name = strings.TrimSpace(raw)
	} else if selfID != "" {
		name = stringValue(index.byID[selfID], keyName)
	}
	for _, tag := range tags {
		if recordID(tag) == selfID || stringValue(tag, fieldParentTagID) != parentID {
			continue
		}
		if strings.EqualFold(stringValue(tag, keyName), name) {
			return fmt.Errorf(
				"%w: a tag named %q already exists at this level of the tag tree",
				ErrUniqueConflict,
				name,
			)
		}
	}
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		if err := tx.ensureExternalIDs(externalIDClaim{
			Kind:  externalIDKindTags,
			Owner: externalIDOwner(ResourceTags, selfID),
			IDs:   mapOf(raw),
		}); err != nil {
			return err
		}
	}
	return validateTagMembersTx(tx, body)
}

func validateTagMembersTx(tx *storeTx, body Record) error {
	assets := tx.records(ResourceAssets)
	assetTypes := make(map[string]string, len(assets))
	for _, asset := range assets {
		assetTypes[recordID(asset)] = stringValue(asset, keyType)
	}
	lookups := map[tagMemberKind]func(id string) bool{
		tagMembersAddresses: recordExists(tx.records(ResourceAddresses)),
		tagMembersDrivers:   recordExists(tx.records(ResourceDrivers)),
		tagMembersVehicles: func(id string) bool {
			return assetTypes[id] == assetTypeVehicle
		},
		tagMembersAssets: func(id string) bool {
			assetType, ok := assetTypes[id]
			return ok && assetType != assetTypeVehicle
		},
		tagMembersMachines: func(string) bool { return false },
		tagMembersSensors:  func(string) bool { return false },
	}
	for _, kind := range tagMemberKinds {
		raw, ok := body[string(kind)]
		if !ok || raw == nil {
			continue
		}
		exists := lookups[kind]
		for _, id := range stringListValues(raw) {
			if !exists(id) {
				return missingReference(strings.TrimSuffix(string(kind), "s"), id)
			}
		}
	}
	return nil
}

func recordExists(records []Record) func(id string) bool {
	ids := make(map[string]struct{}, len(records))
	for _, record := range records {
		ids[recordID(record)] = struct{}{}
	}
	return func(id string) bool {
		_, ok := ids[id]
		return ok
	}
}

func applyTagFields(record, body Record, replace bool) {
	if name, ok := body[keyName].(string); ok {
		record[keyName] = strings.TrimSpace(name)
	}
	if raw, ok := body[fieldParentTagID]; ok {
		if parent, isString := raw.(string); isString && strings.TrimSpace(parent) != "" {
			record[fieldParentTagID] = strings.TrimSpace(parent)
		} else {
			delete(record, fieldParentTagID)
		}
	} else if replace {
		delete(record, fieldParentTagID)
	}
	if raw, ok := body[fieldExternalIDs]; ok {
		if raw == nil {
			delete(record, fieldExternalIDs)
		} else {
			record[fieldExternalIDs] = raw
		}
	}
	for _, kind := range tagMemberKinds {
		raw, ok := body[string(kind)]
		if !ok && !replace {
			continue
		}
		setTagMembers(record, kind, stringListValues(raw))
	}
}
