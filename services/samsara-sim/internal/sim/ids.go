package sim

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

type idKind uint8

const (
	idKindNone idKind = iota
	idKindNumeric
	idKindUUID
	idKindToken
)

type idSpace string

const (
	idSpaceAddresses       idSpace = "addresses"
	idSpaceAssets          idSpace = "assets"
	idSpaceDrivers         idSpace = "drivers"
	idSpaceRoutes          idSpace = "routes"
	idSpaceWebhooks        idSpace = "webhooks"
	idSpaceFormTemplates   idSpace = "formTemplates"
	idSpaceFormSubmissions idSpace = "formSubmissions"
	idSpaceLiveShares      idSpace = "liveShares"
	idSpaceTags            idSpace = "tags"
	idSpaceAssignments     idSpace = "driverVehicleAssignments"
	idSpaceSignOuts        idSpace = "driverSignOuts"
	idSpaceContacts        idSpace = "contacts"
	idSpaceDocuments       idSpace = "documents"
	idSpaceDocumentPDFs    idSpace = "documentPdfs"
	idSpaceFormPDFExports  idSpace = "formSubmissionPdfExports"
	idSpaceDvirs           idSpace = "dvirs"
)

type idScheme struct {
	Kind   idKind
	Space  idSpace
	Floor  int64
	Stride int64
}

const (
	liveShareIDLength   = 19
	liveShareIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	tokenCharsPerBlock  = 12

	routeStopIDFloor  uint64 = 6_500_000_000
	routeStopIDBlocks uint64 = 15_625_000
	routeStopIDBlock  uint64 = 64
	dvirIDFloor       uint64 = 710_000_000
	dvirIDSpan        uint64 = 280_000_000
	dvirDefectIDFloor uint64 = 970_000_000
	dvirDefectIDSpan  uint64 = 29_000_000
	apiDvirIDFloor    int64  = 999_100_000
)

var idSchemedResources = []Resource{
	ResourceAddresses,
	ResourceAssets,
	ResourceVehicleStats,
	ResourceDrivers,
	ResourceRoutes,
	ResourceWebhooks,
	ResourceFormTemplates,
	ResourceFormSubmissions,
	ResourceLiveShares,
	ResourceTags,
	ResourceDriverVehicleAssignments,
	ResourceDriverSignOuts,
	ResourceContacts,
	ResourceDocuments,
	ResourceDocumentPDFs,
	ResourceFormPDFExports,
	ResourceDvirs,
}

func resourceIDScheme(resource Resource) idScheme {
	switch resource {
	case ResourceAddresses:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceAddresses,
			Floor:  41_226_300,
			Stride: 13,
		}
	case ResourceAssets, ResourceVehicleStats:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceAssets,
			Floor:  281_474_977_075_800,
			Stride: 17,
		}
	case ResourceDrivers:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceDrivers,
			Floor:  1_654_900,
			Stride: 23,
		}
	case ResourceRoutes:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceRoutes,
			Floor:  4_129_806_400,
			Stride: 19,
		}
	case ResourceWebhooks:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceWebhooks,
			Floor:  523_900,
			Stride: 7,
		}
	case ResourceFormTemplates:
		return idScheme{Kind: idKindUUID, Space: idSpaceFormTemplates}
	case ResourceFormSubmissions:
		return idScheme{Kind: idKindUUID, Space: idSpaceFormSubmissions}
	case ResourceLiveShares:
		return idScheme{Kind: idKindToken, Space: idSpaceLiveShares}
	case ResourceTags:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceTags,
			Floor:  342_400,
			Stride: 11,
		}
	case ResourceDriverVehicleAssignments:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceAssignments,
			Floor:  91_000_000,
			Stride: 1,
		}
	case ResourceDriverSignOuts:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceSignOuts,
			Floor:  92_000_000,
			Stride: 1,
		}
	case ResourceContacts:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceContacts,
			Floor:  22_400,
			Stride: 7,
		}
	case ResourceDvirs:
		return idScheme{
			Kind:   idKindNumeric,
			Space:  idSpaceDvirs,
			Floor:  apiDvirIDFloor,
			Stride: 7,
		}
	case ResourceAssetLocation,
		ResourceMessages,
		ResourceHOSClocks,
		ResourceHOSLogs,
		ResourceDriverTachograph,
		ResourceVehicleTachograph,
		ResourceDriverWorkflows,
		ResourceDocumentTypes:
		return idScheme{Kind: idKindNone}
	case ResourceDocuments:
		return idScheme{Kind: idKindUUID, Space: idSpaceDocuments}
	case ResourceDocumentPDFs:
		return idScheme{Kind: idKindUUID, Space: idSpaceDocumentPDFs}
	case ResourceFormPDFExports:
		return idScheme{Kind: idKindUUID, Space: idSpaceFormPDFExports}
	default:
		return idScheme{Kind: idKindNone}
	}
}

func numericID(value string) (int64, bool) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0, false
	}
	for idx := 0; idx < len(clean); idx++ {
		if clean[idx] < '0' || clean[idx] > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.ParseInt(clean, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func routeStopID(routeID string, sequence int) string {
	block := fnvHash64("route-stop|"+strings.TrimSpace(routeID)) % routeStopIDBlocks
	offset := uint64(max(sequence, 0)) % routeStopIDBlock
	return strconv.FormatUint(routeStopIDFloor+block*routeStopIDBlock+offset, 10)
}

func routeStopIDForAddress(routeID, addressID string) string {
	return deterministicNumericID(
		routeStopIDFloor,
		routeStopIDBlocks*routeStopIDBlock,
		"route-stop",
		strings.TrimSpace(routeID),
		"address",
		strings.TrimSpace(addressID),
	)
}

func dvirRecordID(dayKey, driverID, dvirType string) string {
	return deterministicNumericID(
		dvirIDFloor,
		dvirIDSpan,
		"dvir",
		dayKey,
		strings.TrimSpace(driverID),
		dvirType,
	)
}

func dvirDefectID(dvirID string, sequence int) string {
	return deterministicNumericID(
		dvirDefectIDFloor,
		dvirDefectIDSpan,
		"dvir-defect",
		dvirID,
		strconv.Itoa(sequence),
	)
}

func generatedRecordID(space idSpace, kind idKind, sequence int64) string {
	sequenceKey := strconv.FormatInt(sequence, 10)
	switch kind {
	case idKindUUID:
		return deterministicUUID("samsara-sim", string(space), sequenceKey)
	case idKindToken:
		return deterministicToken(liveShareIDLength, "samsara-sim", string(space), sequenceKey)
	case idKindNone, idKindNumeric:
		return ""
	default:
		return ""
	}
}

func deterministicNumericID(floor, span uint64, parts ...string) string {
	offset := fnvHash64(strings.Join(parts, "|")) % span
	return strconv.FormatUint(floor+offset, 10)
}

func deterministicUUID(parts ...string) string {
	key := strings.Join(parts, "|")
	var raw [16]byte
	binary.BigEndian.PutUint64(raw[:8], fnvHash64(key+"|event-id-high"))
	binary.BigEndian.PutUint64(raw[8:], fnvHash64(key+"|event-id-low"))
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[:4], raw[4:6], raw[6:8], raw[8:10], raw[10:])
}

func deterministicToken(length int, parts ...string) string {
	if length <= 0 {
		return ""
	}
	key := strings.Join(parts, "|")
	alphabetSize := uint64(len(liveShareIDAlphabet))
	out := make([]byte, length)
	var block uint64
	remaining := 0
	round := 0
	for idx := range out {
		if remaining == 0 {
			block = fnvHash64(key + "|token|" + strconv.Itoa(round))
			round++
			remaining = tokenCharsPerBlock
		}
		out[idx] = liveShareIDAlphabet[block%alphabetSize]
		block /= alphabetSize
		remaining--
	}
	return string(out)
}

func fnvHash64(value string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(value))
	return hasher.Sum64()
}
