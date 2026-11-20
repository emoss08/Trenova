package partner

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"time"
	"unicode/utf8"
)

var (
	qualifierPattern = regexp.MustCompile(`^[0-9A-Z]{2}$`)
	versionPattern   = regexp.MustCompile(`^00[0-9]{4}`)
	scacPattern      = regexp.MustCompile(`^[A-Z]{2,4}$`)
)

var knownSets = []string{"204", "210", "214", "990", "997", "999"}

func (p *Partner) validate(file, folder string) error {
	check := &problems{file: file}
	check.required("name", p.Name)
	if p.Name != "" && p.Name != folder {
		check.add("name", "%q must match its folder %q", p.Name, folder)
	}
	check.required("display", p.Display)
	check.oneOf("role", string(p.Role), string(RoleShipper), string(RoleCarrier))
	if _, err := time.LoadLocation(p.Zone); p.Zone == "" || err != nil {
		check.add("zone", "%q is not an IANA time zone", p.Zone)
	}
	validateIdentity(check, "them", p.Them)
	validateIdentity(check, "trenova", p.Trenova)
	if p.Role == RoleShipper && !scacPattern.MatchString(p.Trenova.SCAC) {
		check.add(
			"trenova.scac",
			"%q is not a SCAC: Trenova is the carrier for a shipper",
			p.Trenova.SCAC,
		)
	}
	if p.Role == RoleCarrier && !scacPattern.MatchString(p.Them.SCAC) {
		check.add("them.scac", "%q is not a SCAC: the partner is a carrier", p.Them.SCAC)
	}
	p.Envelope.validate(check)
	p.Transport.validate(check)
	if p.Batching.TransactionsPerGroup < 1 {
		check.add("batching.transactionsPerGroup", "must be at least 1")
	}
	if p.Batching.GroupsPerInterchange < 1 {
		check.add("batching.groupsPerInterchange", "must be at least 1")
	}
	p.Acknowledgment.validate(check)
	p.validateAnswers(check)
	validateSets(check, "sends", p.Sends)
	validateSets(check, "receives", p.Receives)
	for _, set := range p.Sends {
		if slices.Contains(p.Receives, set) && set != "997" && set != "999" {
			check.add(
				"receives",
				"%s is also in sends: a partner either sends or receives a set",
				set,
			)
		}
	}
	if p.Acknowledgment.Sends != AckNone &&
		!slices.Contains(p.Sends, string(p.Acknowledgment.Sends)) {
		check.add(
			"sends",
			"lacks %s, which acknowledgment.sends says it answers with",
			p.Acknowledgment.Sends,
		)
	}
	if p.Acknowledgment.Expects != AckNone &&
		!slices.Contains(p.Receives, string(p.Acknowledgment.Expects)) {
		check.add(
			"receives",
			"lacks %s, which acknowledgment.expects says it wants",
			p.Acknowledgment.Expects,
		)
	}
	return check.err()
}

func validateIdentity(check *problems, path string, identity Identity) {
	if !qualifierPattern.MatchString(identity.Qualifier) {
		check.add(path+".qualifier", "%q is not a two-character ISA qualifier", identity.Qualifier)
	}
	if identity.ID == "" || len(identity.ID) > 15 {
		check.add(path+".id", "%q must be 1 to 15 characters", identity.ID)
	}
	if len(identity.Group) < 2 || len(identity.Group) > 15 {
		check.add(path+".group", "%q must be 2 to 15 characters", identity.Group)
	}
	if identity.SCAC != "" && !scacPattern.MatchString(identity.SCAC) {
		check.add(path+".scac", "%q is not a SCAC", identity.SCAC)
	}
}

func (e *Envelope) validate(check *problems) {
	if !versionPattern.MatchString(e.Version) {
		check.add("envelope.version", "%q is not an X12 version such as 004010", e.Version)
	}
	for _, set := range slices.Sorted(maps.Keys(e.Versions)) {
		path := "envelope.versions." + set
		if !slices.Contains(knownSets, set) {
			check.add(path, "%q is not one of %v", set, knownSets)
		}
		if version := e.Versions[set]; !versionPattern.MatchString(version) {
			check.add(path, "%q is not an X12 version such as 005010X231A1", version)
		}
	}
	separators := map[string]string{
		"envelope.separators.element":    e.Separators.Element,
		"envelope.separators.segment":    e.Separators.Segment,
		"envelope.separators.component":  e.Separators.Component,
		"envelope.separators.repetition": e.Separators.Repetition,
	}
	seen := map[string]string{}
	for _, path := range slices.Sorted(maps.Keys(separators)) {
		value := separators[path]
		if value == "" && path == "envelope.separators.repetition" {
			continue
		}
		if utf8.RuneCountInString(value) != 1 || len(value) != 1 {
			check.add(path, "%q must be one ASCII character", value)
			continue
		}
		if value[0] == ' ' || isAlphanumeric(value[0]) {
			check.add(path, "%q cannot separate X12", value)
		}
		if other, ok := seen[value]; ok {
			check.add(path, "%q is also %s", value, other)
		}
		seen[value] = path
	}
	if e.Separators.Repetition == "" && slices.ContainsFunc(e.versions(), repeats) {
		check.add(
			"envelope.separators.repetition",
			"is required: a 00402 or later interchange carries its repetition separator in ISA11",
		)
	}
	check.oneOf(
		"envelope.terminator",
		string(e.Terminator),
		string(TerminatorNone),
		string(TerminatorLF),
		string(TerminatorCRLF),
	)
	check.oneOf("envelope.usage", e.Usage, "T", "P")
}

func (t *Transport) validate(check *problems) {
	switch {
	case t.AS2 == nil && t.SFTP == nil:
		check.add("transport", "needs as2 or sftp")
	case t.AS2 != nil && t.SFTP != nil:
		check.add("transport", "takes one of as2 and sftp, not both")
	case t.AS2 != nil:
		check.required("transport.as2.id", t.AS2.ID)
		check.oneOf("transport.as2.mdn", t.AS2.MDN, "sync", "async", "none")
	default:
		validateFileName(check, "transport.sftp.fileName", t.SFTP.FileName)
		check.required("transport.sftp.inbound", t.SFTP.Inbound)
		check.required("transport.sftp.outbound", t.SFTP.Outbound)
		if t.SFTP.Inbound == t.SFTP.Outbound {
			check.add("transport.sftp.outbound", "must differ from inbound")
		}
	}
}

func (a *Acknowledgment) validate(check *problems) {
	acks := []string{string(Ack997), string(Ack999), string(AckNone)}
	check.oneOf("acknowledgment.sends", string(a.Sends), acks...)
	check.oneOf("acknowledgment.expects", string(a.Expects), acks...)
	if a.Delay < 0 {
		check.add("acknowledgment.delay", "cannot be negative")
	}
	check.oneOf(
		"acknowledgment.reject.when",
		string(a.Reject.When),
		string(RejectNever),
		string(RejectErrors),
		string(RejectAny),
	)
	for index, set := range a.Reject.Sets {
		if !slices.Contains(knownSets, set) {
			check.add(fmt.Sprintf("acknowledgment.reject.sets[%d]", index), "%q is not a set", set)
		}
	}
	if a.Never < 0 || a.Never > 1 {
		check.add("acknowledgment.never", "%v is not a probability between 0 and 1", a.Never)
	}
}

func (p *Partner) validateAnswers(check *problems) {
	if p.Role != RoleCarrier {
		if p.Answers != nil {
			check.add("answers", "only a carrier answers tenders")
		}
		return
	}
	if p.Answers == nil {
		check.add("answers", "is required: a carrier answers the tenders Trenova sends it")
		return
	}
	check.oneOf(
		"answers.policy",
		string(p.Answers.Policy),
		string(AnswerAccept),
		string(AnswerDraw),
		string(AnswerDecline),
		string(AnswerNone),
	)
	if p.Answers.Policy == AnswerDecline {
		check.required("answers.decline", p.Answers.Decline)
	}
	if p.Answers.Policy != AnswerDecline && p.Answers.Decline != "" {
		check.add("answers.decline", "is only read when the policy is decline")
	}
	if p.Answers.Pace < 0 {
		check.add("answers.pace", "cannot be negative")
	}
}

func validateSets(check *problems, path string, sets []string) {
	if len(sets) == 0 {
		check.add(path, "lists no sets")
	}
	for index, set := range sets {
		if !slices.Contains(knownSets, set) {
			check.add(fmt.Sprintf("%s[%d]", path, index), "%q is not one of %v", set, knownSets)
		}
	}
}

func isAlphanumeric(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func (e *Envelope) versions() []string {
	return append(slices.Collect(maps.Values(e.Versions)), e.Version)
}

func repeats(version string) bool {
	return len(version) >= 5 && version[:5] >= "00402"
}
