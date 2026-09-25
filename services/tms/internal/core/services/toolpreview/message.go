package toolpreview

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

// Send previews something the write would send: an email, a driver message,
// an EDI document, a comment a customer sees. The message is what would go
// out, rendered by the code that sends it, with the recipients it would
// really reach.
func Send(rec Record, message *agent.MessagePreview) *agent.RecordChange {
	change := newChange(agent.PreviewOperationSend, &rec)
	if message == nil {
		return change
	}

	sent := *message
	sent.From = strings.TrimSpace(sent.From)
	sent.To = stringutils.FilterEmpty(sent.To)
	sent.Cc = stringutils.FilterEmpty(sent.Cc)
	sent.Bcc = stringutils.FilterEmpty(sent.Bcc)
	sent.Attachments = stringutils.FilterEmpty(sent.Attachments)
	sent.Subject = strings.TrimSpace(sent.Subject)
	change.Message = &sent

	return change
}

// MoneyBlock is an amount the write would move, line by line. The totals are
// the sums of the lines' known figures, and the delta is after less before;
// a side with no known figure has no total.
func MoneyBlock(currency string, lines ...agent.MoneyLine) *agent.MoneyPreview {
	block := &agent.MoneyPreview{
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
		Lines:    make([]agent.MoneyLine, 0, len(lines)),
	}

	before, after := decimal.Zero, decimal.Zero
	var anyBefore, anyAfter bool
	for _, line := range lines {
		line.Label = strings.TrimSpace(line.Label)
		block.Lines = append(block.Lines, line)
		if line.Before.Valid {
			before = before.Add(line.Before.Decimal)
			anyBefore = true
		}
		if line.After.Valid {
			after = after.Add(line.After.Decimal)
			anyAfter = true
		}
	}

	if anyBefore {
		block.TotalBefore = decimal.NewNullDecimal(before)
	}
	if anyAfter {
		block.TotalAfter = decimal.NewNullDecimal(after)
	}
	block.Delta = MoneyDelta(block.TotalBefore, block.TotalAfter)

	return block
}

// MoneyDelta is after less before, where a missing side counts as zero; it
// is unknown only when both sides are.
func MoneyDelta(before, after decimal.NullDecimal) decimal.NullDecimal {
	if !before.Valid && !after.Valid {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(after.Decimal.Sub(before.Decimal))
}

// Money attaches an amount to a change of the record, stamped with the
// sensitivity of the fields it is drawn from (SensitiveAs), or the resource's
// default. An amount as sensitive as Confidential is never carried: the change
// is returned without it.
func Money(rec Record, block *agent.MoneyPreview, opts ...Option) *agent.RecordChange {
	change := newChange(agent.PreviewOperationUpdate, &rec)
	AttachMoney(change, block, opts...)

	return change
}

// AttachMoney puts an amount on a change another builder made, with the same
// sensitivity rule as Money.
func AttachMoney(change *agent.RecordChange, block *agent.MoneyPreview, opts ...Option) {
	if change == nil || block == nil {
		return
	}

	o := newOptions(opts)
	sensitivity := moneySensitivity(o, change.Resource)
	if sensitivity == permission.SensitivityConfidential {
		change.Money = nil

		return
	}

	attached := *block
	attached.Lines = append([]agent.MoneyLine(nil), block.Lines...)
	attached.Sensitivity = sensitivity
	change.Money = &attached
}

func moneySensitivity(o *options, resource permission.Resource) permission.FieldSensitivity {
	if len(o.sensitiveAs) == 0 {
		return fieldsensitivity.Level(o.registry, resource, "")
	}

	highest := permission.SensitivityPublic
	for _, path := range o.sensitiveAs {
		level := fieldsensitivity.Level(o.registry, resource, topLevel(path))
		if level.Level() > highest.Level() {
			highest = level
		}
	}

	return highest
}
