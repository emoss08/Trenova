package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func rateconPages() []extractionPage {
	return []extractionPage{
		{Number: 2, Width: 1650, Height: 1275, Lines: []documentcontent.LayoutLine{
			{Text: "DELIVERY 2", X: 0.1, Y: 0.14, W: 0.12, H: 0.018},
			{Text: "Anyco Clothes #612", X: 0.1, Y: 0.18, W: 0.2, H: 0.018},
			{Text: "220 Market St, Fort Worth, TX 76102", X: 0.1, Y: 0.22, W: 0.34, H: 0.018},
			{Text: "10/22/2026 Appt 09:00", X: 0.1, Y: 0.26, W: 0.22, H: 0.018},
		}},
		{Number: 1, Width: 1275, Height: 1650, Lines: []documentcontent.LayoutLine{
			{Text: "NORTHLINE LOGISTICS", X: 0.08, Y: 0.05, W: 0.3, H: 0.02},
			{Text: "Load # NL-482913", X: 0.62, Y: 0.085, W: 0.2, H: 0.014},
			{
				Text: "Bill to: Northline Logistics, 400 Commerce St, Dallas TX 75202",
				X:    0.08, Y: 0.17, W: 0.66, H: 0.014,
			},
			{Text: "Ridgefield Paper Co.", X: 0.08, Y: 0.25, W: 0.22, H: 0.014},
			{
				Text: "Total $2,465.00", X: 0.52, Y: 0.5, W: 0.18, H: 0.018,
				Words: []float64{0.52, 0.58, 0.6, 0.7},
			},
		}},
	}
}

func rateconDraft() map[string]any {
	return map[string]any{
		"documentId":   "doc_1",
		"status":       "Ready",
		"reviewStatus": "NeedsReview",
		"documentKind": "RateConfirmation",
		"confidence":   0.91,
		"fields": map[string]any{
			"loadNumber": map[string]any{
				"label": "Load Number", "value": "NL-482913", "confidence": 0.97, "pageNumber": 1.0,
			},
			"billTo": map[string]any{
				"label": "Bill To", "value": "Northline Logistics", "confidence": 0.71,
				"pageNumber": 1.0, "reviewRequired": true,
			},
			"rate": map[string]any{
				"label": "Rate", "value": "2465.00", "confidence": 0.96, "pageNumber": 1.0,
			},
			"deliveryWindow": map[string]any{
				"label": "Delivery Window", "value": "10/21/2026", "confidence": 0.62,
				"conflict": true,
			},
			"commodity": map[string]any{
				"label": "Commodity", "value": "Paper products", "confidence": 0.78,
			},
			"notes": map[string]any{"label": "Notes", "value": "", "confidence": 0.9},
		},
		"stops": []any{
			map[string]any{
				"sequence": 1.0, "role": "pickup", "name": "Ridgefield Paper Co.",
				"confidence": 0.93, "pageNumber": 1.0,
			},
			map[string]any{
				"sequence": 3.0, "role": "Delivery", "name": "Anyco Clothes #612",
				"addressLine1": "220 Market St", "city": "Fort Worth", "state": "TX",
				"postalCode": "76102", "date": "10/22/2026", "timeWindow": "09:00",
				"appointmentRequired": true, "confidence": 0.74, "reviewRequired": true,
				"pageNumber": 2.0,
			},
		},
		"conflicts": []any{
			map[string]any{
				"key": "deliveryWindow", "label": "Delivery Window",
				"values": []any{"10/21/2026", "10/22/2026", "10/21/2026"},
			},
		},
		"missingFields": []any{"Pro Number", "Piece Count", "Pickup Stop"},
	}
}

func fieldsByKey(t *testing.T, artifact *assistantartifact.Artifact) map[string]map[string]any {
	t.Helper()

	fields, ok := artifact.Payload["fields"].([]map[string]any)
	require.True(t, ok)
	out := make(map[string]map[string]any, len(fields))
	for _, field := range fields {
		out[field["key"].(string)] = field
	}

	return out
}

func TestExtractionArtifact_BoxesTheLabelledLineOverTheLetterhead(t *testing.T) {
	t.Parallel()

	artifact := extractionArtifact(
		"call_1",
		rateconDraft(),
		extractionSource{pages: rateconPages()},
	)
	require.NotNil(t, artifact)

	billTo := fieldsByKey(t, artifact)["billTo"]
	assert.Equal(t, 1, billTo["page"])
	box := billTo["box"].(map[string]float64)
	assert.InDelta(t, 0.17, box["y"], 0.0001, "the Bill to line, not NORTHLINE LOGISTICS")
	assert.InDelta(t, 0.08+0.66*(9.0/62.0), box["x"], 0.0001)
	assert.InDelta(t, 0.66*(19.0/62.0), box["w"], 0.0001)
}

func TestExtractionArtifact_PrefersTheLineTheReadingQuoted(t *testing.T) {
	t.Parallel()

	source := extractionSource{
		pages:         rateconPages(),
		fieldEvidence: map[string]string{"billTo": "NORTHLINE LOGISTICS"},
	}
	artifact := extractionArtifact("call_1", rateconDraft(), source)
	require.NotNil(t, artifact)

	box := fieldsByKey(t, artifact)["billTo"]["box"].(map[string]float64)
	assert.InDelta(t, 0.05, box["y"], 0.0001)
	assert.InDelta(t, 0.08, box["x"], 0.0001)
}

func TestExtractionArtifact_MeasuresTheValueOnTheLineItself(t *testing.T) {
	t.Parallel()

	artifact := extractionArtifact(
		"call_1",
		rateconDraft(),
		extractionSource{pages: rateconPages()},
	)
	require.NotNil(t, artifact)

	assert.Equal(t, map[string]float64{
		"x": 0.7075, "y": 0.085, "w": 0.1125, "h": 0.014,
	}, fieldsByKey(t, artifact)["loadNumber"]["box"],
		"NL-482913 starts after \"Load # \", spaces and the hash counted")
}

func TestExtractionArtifact_PlacesWordsWhereTheReaderPutThem(t *testing.T) {
	t.Parallel()

	artifact := extractionArtifact(
		"call_1",
		rateconDraft(),
		extractionSource{pages: rateconPages()},
	)
	require.NotNil(t, artifact)

	box := fieldsByKey(t, artifact)["rate"]["box"].(map[string]float64)
	assert.InDelta(
		t,
		0.6+0.1*(1.0/9.0),
		box["x"],
		0.0001,
		"past the dollar sign of the second word",
	)
	assert.InDelta(t, 0.7-(0.6+0.1*(1.0/9.0)), box["w"], 0.0001)
	assert.InDelta(t, 0.5, box["y"], 0.0001)
}

func TestExtractionArtifact_SaysWhyAFieldNeedsALook(t *testing.T) {
	t.Parallel()

	artifact := extractionArtifact(
		"call_1",
		rateconDraft(),
		extractionSource{pages: rateconPages()},
	)
	require.NotNil(t, artifact)
	fields := fieldsByKey(t, artifact)

	assert.Equal(t, "conflict", fields["deliveryWindow"]["reason"])
	assert.Equal(t, []string{"10/21/2026", "10/22/2026"}, fields["deliveryWindow"]["readings"])
	assert.Equal(t, "review", fields["billTo"]["reason"])
	assert.Equal(t, "low", fields["commodity"]["reason"])
	assert.Equal(t, true, fields["commodity"]["needsLook"])

	_, hasReason := fields["loadNumber"]["reason"]
	assert.False(t, hasReason)
	assert.Equal(t, false, fields["loadNumber"]["needsLook"])
	assert.NotContains(t, fields, "notes", "an empty value is not a reading")
	assert.Equal(t, 6, artifact.Payload["needsLook"], "each part of a stop under review counts")
}

func TestExtractionArtifact_ReadsEachStopAsItsParts(t *testing.T) {
	t.Parallel()

	artifact := extractionArtifact(
		"call_1",
		rateconDraft(),
		extractionSource{pages: rateconPages()},
	)
	require.NotNil(t, artifact)
	fields := fieldsByKey(t, artifact)

	name := fields["stop3.name"]
	require.NotNil(t, name)
	assert.Equal(t, 3, name["stop"])
	assert.Equal(t, "delivery", name["role"])
	assert.Equal(t, "name", name["part"])
	assert.Equal(t, "Anyco Clothes #612", name["value"])
	assert.Equal(t, 2, name["page"])
	assert.Equal(t, "review", name["reason"])

	address := fields["stop3.address"]
	assert.Equal(t, "220 Market St, Fort Worth, TX 76102", address["value"])
	assert.InDelta(t, 0.22, address["box"].(map[string]float64)["y"], 0.0001)
	assert.InDelta(t, 0.34, address["box"].(map[string]float64)["w"], 0.0001,
		"a long value is boxed to its end, not only the part looked for")

	window := fields["stop3.window"]
	assert.Equal(t, "10/22/2026 09:00", window["value"])
	windowBox := window["box"].(map[string]float64)
	assert.InDelta(t, 0.26, windowBox["y"], 0.0001,
		"a value split by other words on the page is boxed on its leading words")
	assert.InDelta(t, 0.22*(10.0/21.0), windowBox["w"], 0.0001)
	assert.Equal(t, true, window["appointment"])
	_, nameAppointment := name["appointment"]
	assert.False(t, nameAppointment)

	for _, part := range []map[string]any{name, address, window} {
		assert.Equal(t, "Fort Worth, TX", part["place"], "every part of a stop names its place")
	}

	pickup := fields["stop1.name"]
	assert.Equal(t, "pickup", pickup["role"])
	_, pickupPlace := pickup["place"]
	assert.False(t, pickupPlace, "a stop read without a city or state names no place")
	assert.NotContains(t, fields, "stop1.address", "a part the reading lacks is not a field")
	assert.NotContains(t, fields, "stop2.name", "stops keep their own sequence")
}

func TestExtractionArtifact_DescribesTheFileAndItsPages(t *testing.T) {
	t.Parallel()

	source := extractionSource{
		fileName:       "Northline_RateCon.pdf",
		kindConfidence: 0.944,
		pages:          rateconPages(),
	}
	artifact := extractionArtifact("call_1", rateconDraft(), source)
	require.NotNil(t, artifact)

	assert.Equal(t, assistantartifact.KindExtraction, artifact.Kind)
	assert.Equal(t, "Rate confirmation · Northline_RateCon.pdf", artifact.Title)
	assert.Equal(t, "RateConfirmation", artifact.Payload["documentKind"])
	assert.Equal(t, "Rate confirmation", artifact.Payload["kindLabel"])
	assert.InDelta(t, 0.94, artifact.Payload["kindConfidence"], 0.0001)
	assert.InDelta(t, 0.91, artifact.Payload["confidence"], 0.0001)
	assert.Equal(t, "NeedsReview", artifact.Payload["reviewStatus"])
	assert.Equal(t, 2, artifact.Payload["pageCount"])
	assert.Equal(t, []map[string]int{
		{"number": 1, "width": 1275, "height": 1650},
		{"number": 2, "width": 1650, "height": 1275},
	}, artifact.Payload["pages"])
	assert.Equal(
		t,
		[]string{"Pro Number", "Piece Count", "Pickup Stop"},
		artifact.Payload["missing"],
	)
	assert.Equal(t, []map[string]string{
		{"key": "proNumber", "label": "Pro Number"},
		{"key": "pieceCount", "label": "Piece Count"},
		{"key": "pickupStop", "label": "Pickup Stop"},
	}, artifact.Payload["missingFields"])
	_, attached := artifact.Payload["attachedShipmentId"]
	assert.False(t, attached)
}

func TestExtractionArtifact_LooksOnOtherPagesWhenTheNamedOneLacksTheValue(t *testing.T) {
	t.Parallel()

	draft := map[string]any{
		"documentId": "doc_1",
		"fields": map[string]any{
			"loadNumber": map[string]any{
				"label": "Load Number", "value": "NL-482913", "confidence": 0.97, "pageNumber": 2.0,
			},
		},
	}
	artifact := extractionArtifact("call_1", draft, extractionSource{pages: rateconPages()})
	require.NotNil(t, artifact)

	assert.Equal(t, 1, fieldsByKey(t, artifact)["loadNumber"]["page"])
}

func TestExtractionArtifact_KeepsTheNamedPageOfAValueNotOnAnyPage(t *testing.T) {
	t.Parallel()

	draft := map[string]any{
		"documentId": "doc_1",
		"fields": map[string]any{
			"scac": map[string]any{
				"label":      "SCAC",
				"value":      "RWFL",
				"confidence": 0.9,
				"pageNumber": 2.0,
			},
		},
	}
	artifact := extractionArtifact("call_1", draft, extractionSource{pages: rateconPages()})
	require.NotNil(t, artifact)

	scac := fieldsByKey(t, artifact)["scac"]
	assert.Equal(t, 2, scac["page"])
	assert.NotContains(t, scac, "box")
}

func TestExtractionArtifact_NothingReadIsNoArtifact(t *testing.T) {
	t.Parallel()

	assert.Nil(t, extractionArtifact("call_1", map[string]any{
		"documentId": "doc_1", "status": "Pending", "fields": map[string]any{},
	}, extractionSource{}))
}

func TestExtractionArtifact_AttachedShipmentIsCarried(t *testing.T) {
	t.Parallel()

	draft := rateconDraft()
	draft["attachedShipmentId"] = "shp_1"
	artifact := extractionArtifact("call_1", draft, extractionSource{})
	require.NotNil(t, artifact)

	assert.Equal(t, "shp_1", artifact.Payload["attachedShipmentId"])
	assert.Equal(
		t,
		2,
		artifact.Payload["pageCount"],
		"the pages the readings name, with none stored",
	)
	assert.Equal(t, []map[string]int{}, artifact.Payload["pages"])
}

func TestDraftEvidence_ReadsWhatEachFieldAndStopQuoted(t *testing.T) {
	t.Parallel()

	fields, stops := draftEvidence(map[string]any{
		"fields": map[string]any{
			"billTo":     map[string]any{"evidenceExcerpt": "Bill to: Northline", "excerpt": "x"},
			"loadNumber": map[string]any{"excerpt": "Load # NL-482913"},
			"rate":       map[string]any{"value": "1"},
			"broken":     "text",
		},
		"stops": []any{
			map[string]any{"evidenceExcerpt": "PICKUP 1 Ridgefield"},
			"text",
			map[string]any{"name": "No quote"},
		},
	})

	assert.Equal(t, map[string]string{
		"billTo":     "Bill to: Northline",
		"loadNumber": "Load # NL-482913",
	}, fields)
	assert.Equal(t, map[int]string{0: "PICKUP 1 Ridgefield"}, stops)
}

func TestArtifactRecorder_ExtractionReadsTheFileItsPagesAndEvidence(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	documents := documentportsmock.NewMockDocumentRepository(t)
	documents.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&document.Document{
		ID: documentID, OriginalName: "Northline_RateCon.pdf", FileName: "stored.pdf",
		FileType: "Application/PDF",
	}, nil)

	contents := documentportsmock.NewMockDocumentContentService(t)
	contents.EXPECT().GetContent(mock.Anything, documentID, tenant).Return(&documentcontent.Content{
		ClassificationConfidence: 0.94,
		Pages: []*documentcontent.Page{{
			PageNumber: 1, Width: 1275, Height: 1650,
			Metadata: map[string]any{documentcontent.MetadataLines: []any{
				map[string]any{
					"t": "NORTHLINE LOGISTICS",
					"x": 0.08,
					"y": 0.05,
					"w": 0.3,
					"h": 0.02,
				},
				map[string]any{
					"t": "Northline Logistics remit",
					"x": 0.1,
					"y": 0.7,
					"w": 0.4,
					"h": 0.014,
				},
			}},
		}},
	}, nil)
	contents.EXPECT().GetShipmentDraft(mock.Anything, documentID, tenant).Return(
		&documentshipmentdraft.DocumentShipmentDraft{DraftData: map[string]any{
			"fields": map[string]any{
				"billTo": map[string]any{"evidenceExcerpt": "Northline Logistics remit to"},
			},
		}}, nil)

	recorder := &artifactRecorder{
		ctx:    t.Context(),
		logger: zap.NewNop(),
		tenant: tenant,
		svc:    &Service{documents: documents, contents: contents},
	}

	artifact := recorder.extraction("call_1", map[string]any{
		"documentId":   documentID.String(),
		"documentKind": "RateConfirmation",
		"fields": map[string]any{
			"billTo": map[string]any{
				"label": "Bill To", "value": "Northline Logistics", "confidence": 0.9, "pageNumber": 1.0,
			},
		},
	})
	require.NotNil(t, artifact)

	assert.Equal(t, "Northline_RateCon.pdf", artifact.Payload["fileName"])
	assert.Equal(t, "application/pdf", artifact.Payload["contentType"])
	assert.InDelta(t, 0.94, artifact.Payload["kindConfidence"], 0.0001)
	box := fieldsByKey(t, artifact)["billTo"]["box"].(map[string]float64)
	assert.InDelta(t, 0.7, box["y"], 0.0001, "the line the reading quoted")
}
