package docling

import (
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	contentLayerFurniture = "furniture"

	coordOriginBottomLeft = "BOTTOMLEFT"

	labelTitle              = "title"
	labelSectionHeader      = "section_header"
	labelListItem           = "list_item"
	labelCode               = "code"
	labelFormula            = "formula"
	labelCheckboxSelected   = "checkbox_selected"
	labelCheckboxUnselected = "checkbox_unselected"
	labelPageHeader         = "page_header"
	labelPageFooter         = "page_footer"

	conversionStatusSuccess        = "success"
	conversionStatusPartialSuccess = "partial_success"
)

type submitResponse struct {
	TaskID string `json:"task_id"`
}

type statusResponse struct {
	TaskID       string `json:"task_id"`
	TaskStatus   string `json:"task_status"`
	ErrorMessage string `json:"error_message"`
}

type convertResponse struct {
	Document   exportedDocument  `json:"document"`
	Status     string            `json:"status"`
	Errors     []conversionError `json:"errors"`
	Confidence *confidenceScores `json:"confidence"`
}

type exportedDocument struct {
	Filename    string                 `json:"filename"`
	JSONContent sonic.NoCopyRawMessage `json:"json_content"`
}

type conversionError struct {
	ComponentType string `json:"component_type"`
	ErrorMessage  string `json:"error_message"`
	PageNo        *int   `json:"page_no"`
}

type confidenceScores struct {
	ParseScore *float64 `json:"parse_score"`
	OCRScore   *float64 `json:"ocr_score"`
}

type versionResponse struct {
	DoclingServe string `json:"docling-serve"`
}

type doclingDocument struct {
	SchemaName string          `json:"schema_name"`
	Origin     *documentOrigin `json:"origin"`
	Body       node            `json:"body"`
	Furniture  node            `json:"furniture"`
	Groups     []node          `json:"groups"`
	Texts      []textItem      `json:"texts"`
	Tables     []tableItem     `json:"tables"`
	Pictures   []node          `json:"pictures"`
	Pages      map[string]page `json:"pages"`
}

type documentOrigin struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
}

type ref struct {
	Ref string `json:"$ref"`
}

type node struct {
	SelfRef      string `json:"self_ref"`
	Children     []ref  `json:"children"`
	ContentLayer string `json:"content_layer"`
	Label        string `json:"label"`
	Prov         []prov `json:"prov"`
}

type textItem struct {
	node

	Text       string `json:"text"`
	Level      int    `json:"level"`
	Enumerated bool   `json:"enumerated"`
	Marker     string `json:"marker"`
}

type tableItem struct {
	node

	Data tableData `json:"data"`
}

type tableData struct {
	NumRows    int         `json:"num_rows"`
	NumCols    int         `json:"num_cols"`
	TableCells []tableCell `json:"table_cells"`
}

type tableCell struct {
	BBox         *bbox  `json:"bbox"`
	StartRow     int    `json:"start_row_offset_idx"`
	EndRow       int    `json:"end_row_offset_idx"`
	StartCol     int    `json:"start_col_offset_idx"`
	EndCol       int    `json:"end_col_offset_idx"`
	ColumnHeader bool   `json:"column_header"`
	Text         string `json:"text"`
}

type prov struct {
	PageNo   int   `json:"page_no"`
	BBox     bbox  `json:"bbox"`
	Charspan []int `json:"charspan"`
}

type bbox struct {
	L           float64 `json:"l"`
	T           float64 `json:"t"`
	R           float64 `json:"r"`
	B           float64 `json:"b"`
	CoordOrigin string  `json:"coord_origin"`
}

type page struct {
	PageNo int      `json:"page_no"`
	Size   pageSize `json:"size"`
}

type pageSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func decodeDocument(raw []byte) (*doclingDocument, error) {
	var doc doclingDocument
	if err := sonic.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

type itemKind int

const (
	itemUnknown itemKind = iota
	itemText
	itemTable
	itemGroup
	itemPicture
)

func (d *doclingDocument) resolve(reference string) (kind itemKind, index int) {
	collection, position, ok := strings.Cut(strings.TrimPrefix(reference, "#/"), "/")
	if !ok {
		return itemUnknown, 0
	}
	index, err := strconv.Atoi(position)
	if err != nil || index < 0 {
		return itemUnknown, 0
	}

	switch collection {
	case "texts":
		if index < len(d.Texts) {
			return itemText, index
		}
	case "tables":
		if index < len(d.Tables) {
			return itemTable, index
		}
	case "groups":
		if index < len(d.Groups) {
			return itemGroup, index
		}
	case "pictures":
		if index < len(d.Pictures) {
			return itemPicture, index
		}
	}

	return itemUnknown, 0
}
