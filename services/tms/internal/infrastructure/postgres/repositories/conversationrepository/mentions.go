package conversationrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

const humanizedStatus = `regexp_replace(%s::text, '([a-z])([A-Z])', '\1 \2', 'g')`

func humanize(column string) string {
	return fmt.Sprintf(humanizedStatus, column)
}

var mentionQueries = map[string]string{
	"shipment": `
SELECT 'shipment' AS type, s.id::text AS id, s.pro_number AS label,
	concat_ws(' · ', NULLIF(concat_ws(' → ', o.place, d.place), ''), ` + humanize("s.status") + `) AS subtitle
FROM shipments AS s
LEFT JOIN LATERAL (
	SELECT concat_ws(', ', l.city, st.abbreviation) AS place
	FROM shipment_moves AS m
	JOIN stops AS sp ON sp.shipment_move_id = m.id
		AND sp.organization_id = m.organization_id AND sp.business_unit_id = m.business_unit_id
	JOIN locations AS l ON l.id = sp.location_id
		AND l.organization_id = sp.organization_id AND l.business_unit_id = sp.business_unit_id
	LEFT JOIN us_states AS st ON st.id = l.state_id
	WHERE m.shipment_id = s.id
		AND m.organization_id = s.organization_id AND m.business_unit_id = s.business_unit_id
	ORDER BY m.sequence, sp.sequence
	LIMIT 1
) AS o ON true
LEFT JOIN LATERAL (
	SELECT concat_ws(', ', l.city, st.abbreviation) AS place
	FROM shipment_moves AS m
	JOIN stops AS sp ON sp.shipment_move_id = m.id
		AND sp.organization_id = m.organization_id AND sp.business_unit_id = m.business_unit_id
	JOIN locations AS l ON l.id = sp.location_id
		AND l.organization_id = sp.organization_id AND l.business_unit_id = sp.business_unit_id
	LEFT JOIN us_states AS st ON st.id = l.state_id
	WHERE m.shipment_id = s.id
		AND m.organization_id = s.organization_id AND m.business_unit_id = s.business_unit_id
	ORDER BY m.sequence DESC, sp.sequence DESC
	LIMIT 1
) AS d ON true
WHERE s.organization_id = ? AND s.business_unit_id = ?
	AND (s.pro_number ILIKE ? OR s.bol ILIKE ?)
ORDER BY (s.pro_number ILIKE ?) DESC, s.updated_at DESC
LIMIT ?`,
	"customer": `
SELECT 'customer' AS type, c.id::text AS id, c.name AS label,
	concat_ws(' · ', c.code, NULLIF(c.city, '')) AS subtitle
FROM customers AS c
WHERE c.organization_id = ? AND c.business_unit_id = ?
	AND (c.name ILIKE ? OR c.code ILIKE ?)
ORDER BY (c.name ILIKE ?) DESC, c.name
LIMIT ?`,
	"invoice": `
SELECT 'invoice' AS type, i.id::text AS id, i.number AS label,
	concat_ws(' · ', NULLIF(i.bill_to_name, ''), ` + humanize("i.status") + `) AS subtitle
FROM invoices AS i
WHERE i.organization_id = ? AND i.business_unit_id = ?
	AND (i.number ILIKE ? OR i.bill_to_name ILIKE ?)
ORDER BY (i.number ILIKE ?) DESC, i.updated_at DESC
LIMIT ?`,
	"invoice_dispute": `
SELECT 'invoice_dispute' AS type, d.id::text AS id, i.number AS label,
	concat_ws(' · ', NULLIF(i.bill_to_name, ''), ` + humanize("d.reason_code") + `, ` + humanize("d.status") + `) AS subtitle
FROM invoice_disputes AS d
JOIN invoices AS i ON i.id = d.invoice_id
	AND i.organization_id = d.organization_id AND i.business_unit_id = d.business_unit_id
WHERE d.organization_id = ? AND d.business_unit_id = ?
	AND (i.number ILIKE ? OR i.bill_to_name ILIKE ?)
ORDER BY (i.number ILIKE ?) DESC, d.updated_at DESC
LIMIT ?`,
	"billing_queue_item": `
SELECT 'billing_queue_item' AS type, q.id::text AS id, q.number AS label,
	concat_ws(' · ', NULLIF(s.pro_number, ''), ` + humanize("q.status") + `) AS subtitle
FROM billing_queue_items AS q
LEFT JOIN shipments AS s ON s.id = q.shipment_id
	AND s.organization_id = q.organization_id AND s.business_unit_id = q.business_unit_id
WHERE q.organization_id = ? AND q.business_unit_id = ?
	AND (q.number ILIKE ? OR s.pro_number ILIKE ?)
ORDER BY (q.number ILIKE ?) DESC, q.updated_at DESC
LIMIT ?`,
	"worker": `
SELECT 'worker' AS type, w.id::text AS id, concat_ws(' ', w.first_name, w.last_name) AS label,
	concat_ws(' · ', ` + humanize("w.type") + `, NULLIF(w.city, '')) AS subtitle
FROM workers AS w
WHERE w.organization_id = ? AND w.business_unit_id = ?
	AND (concat_ws(' ', w.first_name, w.last_name) ILIKE ? OR w.last_name ILIKE ?)
ORDER BY (w.first_name ILIKE ?) DESC, w.first_name, w.last_name
LIMIT ?`,
	"carrier": `
SELECT 'carrier' AS type, c.id::text AS id, c.name AS label,
	concat_ws(' · ', 'MC ' || NULLIF(c.mc_number, ''), ` + humanize("c.compliance_status") + `) AS subtitle
FROM carriers AS c
WHERE c.organization_id = ? AND c.business_unit_id = ?
	AND (c.name ILIKE ? OR c.code ILIKE ? OR c.mc_number ILIKE ?)
ORDER BY (c.name ILIKE ?) DESC, c.name
LIMIT ?`,
}

func likeEscape(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func (r *repository) SearchMentions(
	ctx context.Context,
	req repositories.SearchMentionsRequest,
) ([]repositories.MentionRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.MentionRow, error) {
		db := r.db.DBForContext(ctx)
		term := likeEscape(strings.TrimSpace(req.Query))
		contains := "%" + term + "%"
		prefix := term + "%"
		out := make([]repositories.MentionRow, 0, len(req.Kinds)*req.LimitPerKind)
		for _, kind := range req.Kinds {
			query, ok := mentionQueries[kind]
			if !ok {
				continue
			}
			args := []any{req.TenantInfo.OrgID, req.TenantInfo.BuID, contains, contains}
			if kind == "carrier" {
				args = append(args, contains)
			}
			args = append(args, prefix, req.LimitPerKind, max(req.Offset, 0))
			rows := make([]repositories.MentionRow, 0, req.LimitPerKind)
			if err := db.NewRaw(query+"\nOFFSET ?", args...).Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("search %s mentions: %w", kind, err)
			}
			out = append(out, rows...)
		}
		return out, nil
	})
}
