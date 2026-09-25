package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var _ serviceports.ToolPreviewer = (*createTableChangeAlertTool)(nil)

var alertViewLabels = map[string]string{
	"table":          "Watches",
	"events":         "On",
	"watchedColumns": "Only when these change",
	"conditions":     "When",
	"match":          "Conditions that must hold",
	fieldMessage:     "Message",
}

type alertView struct {
	Name           string   `json:"name"`
	Table          string   `json:"table"`
	Events         []string `json:"events"`
	WatchedColumns []string `json:"watchedColumns,omitempty"`
	Conditions     []string `json:"conditions,omitempty"`
	Match          string   `json:"match"`
	Message        string   `json:"message,omitempty"`
	Status         string   `json:"status"`
}

func alertViewOf(subscription *tablechangealert.TCASubscription) *alertView {
	view := &alertView{
		Name:           subscription.Name,
		Table:          subscription.TableName,
		Events:         subscription.EventTypes,
		WatchedColumns: subscription.WatchedColumns,
		Conditions:     make([]string, 0, len(subscription.Conditions)),
		Match:          subscription.ConditionMatch,
		Message:        subscription.CustomMessage,
		Status:         string(subscription.Status),
	}
	for idx := range subscription.Conditions {
		condition := &subscription.Conditions[idx]
		view.Conditions = append(view.Conditions, strings.TrimSpace(fmt.Sprintf(
			"%s %s %s",
			condition.Field,
			condition.Operator,
			agent.PreviewValueText(condition.Value),
		)))
	}

	return view
}

func (t *createTableChangeAlertTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	subscription := alertSubscription(&params)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceTableChangeAlert,
		Label:    subscription.Name,
	}, alertViewOf(subscription), toolpreview.Labels(alertViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would alert you when a %s record is %s, until the alert is paused or deleted.",
		subscription.TableName,
		alertEventWords(subscription.EventTypes),
	), change)

	if err = t.validateArgs(params.Params); err == nil {
		err = t.alerts.CheckSubscription(ctx, subscription)
	}

	return warnRefusal(preview, err)
}

func alertEventWords(events []string) string {
	words := make([]string, 0, len(events))
	for _, event := range events {
		switch event {
		case "INSERT":
			words = append(words, "added")
		case "UPDATE":
			words = append(words, "changed")
		case "DELETE":
			words = append(words, "deleted")
		default:
			words = append(words, strings.ToLower(event))
		}
	}

	return strings.Join(words, " or ")
}
