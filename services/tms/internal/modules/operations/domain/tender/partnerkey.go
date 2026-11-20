package tender

import (
	"strconv"
	"strings"
)

const partnerKeyMoveSeparator = "-"

type PartnerKey struct {
	ProNumber string
	Move      int64
}

func MovePartnerKey(proNumber string, moveSequence int64, moves int) PartnerKey {
	key := PartnerKey{ProNumber: strings.TrimSpace(proNumber)}
	if moves > 1 {
		key.Move = moveSequence + 1
	}
	return key
}

func (k PartnerKey) String() string {
	if k.ProNumber == "" || k.Move < 1 {
		return k.ProNumber
	}
	return k.ProNumber + partnerKeyMoveSeparator + strconv.FormatInt(k.Move, 10)
}

func (k PartnerKey) MoveSequence() int64 {
	return k.Move - 1
}

func ParsePartnerKeys(references ...string) []PartnerKey {
	keys := make([]PartnerKey, 0, len(references)*2)
	seen := make(map[PartnerKey]struct{}, len(references)*2)
	add := func(key PartnerKey) {
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for _, reference := range references {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			continue
		}
		add(PartnerKey{ProNumber: reference})
		at := strings.LastIndex(reference, partnerKeyMoveSeparator)
		if at < 1 {
			continue
		}
		move, err := strconv.ParseInt(reference[at+1:], 10, 64)
		if err != nil || move < 1 {
			continue
		}
		add(PartnerKey{ProNumber: reference[:at], Move: move})
	}
	return keys
}

func ProNumbersOf(keys []PartnerKey) []string {
	numbers := make([]string, 0, len(keys))
	for _, key := range keys {
		numbers = append(numbers, key.ProNumber)
	}
	return numbers
}
