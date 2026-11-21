package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
)

const sealDomain = "trenova.audit-seal/v1"

var sealVersions = map[Origin][2]int16{
	OriginHere:           {SealHashVersionSigned, SealHashVersionUnsigned},
	OriginSourceVerified: {SealHashVersionSourceVerified, SealHashVersionSourceVerifiedUnsigned},
	OriginUnverified:     {SealHashVersionImported, SealHashVersionImportedUnsigned},
}

func (s *Seal) Sign(keyID string, secret []byte) {
	s.SignFrom(OriginHere, keyID, secret)
}

func (s *Seal) SignFrom(origin Origin, keyID string, secret []byte) {
	versions := sealVersions[origin]
	if keyID != "" {
		s.HashKeyID = keyID
		s.HashVersion = versions[0]
	} else {
		s.HashKeyID = ""
		s.HashVersion = versions[1]
		secret = nil
	}
	s.Hash = s.ComputeHash(secret)
}

func (s *Seal) Unsigned() bool {
	switch s.HashVersion {
	case SealHashVersionUnsigned, SealHashVersionImportedUnsigned,
		SealHashVersionSourceVerifiedUnsigned:
		return true
	default:
		return false
	}
}

func (s *Seal) Origin() Origin {
	switch s.HashVersion {
	case SealHashVersionImported, SealHashVersionImportedUnsigned:
		return OriginUnverified
	case SealHashVersionSourceVerified, SealHashVersionSourceVerifiedUnsigned:
		return OriginSourceVerified
	default:
		return OriginHere
	}
}

func (s *Seal) ComputeHash(secret []byte) string {
	message := s.message()
	if s.Unsigned() {
		sum := sha256.Sum256([]byte(message))
		return hex.EncodeToString(sum[:])
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Seal) message() string {
	classes := make([]string, 0, len(s.ClassDigests))
	for class := range s.ClassDigests {
		classes = append(classes, class)
	}
	slices.Sort(classes)

	var b strings.Builder
	b.WriteString(sealDomain)
	for _, part := range []string{
		strconv.Itoa(int(s.HashVersion)),
		s.HashKeyID,
		s.PrevHash,
		s.OrganizationID.String(),
		s.BusinessUnitID.String(),
		strconv.FormatInt(s.Seq, 10),
		strconv.FormatInt(s.FromTS, 10),
		strconv.FormatInt(s.ToTS, 10),
		strconv.FormatInt(s.RowCount, 10),
	} {
		b.WriteByte('\n')
		b.WriteString(part)
	}
	for _, class := range classes {
		digest := s.ClassDigests[class]
		b.WriteByte('\n')
		b.WriteString(class)
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(digest.Count, 10))
		b.WriteByte(':')
		b.WriteString(digest.Digest)
	}
	return b.String()
}
