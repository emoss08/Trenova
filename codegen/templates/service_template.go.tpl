package services

import (
	"context"

    "github.com/emoss08/trenova/ent/{{.EntityNameLower}}"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
    "github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type {{.EntityName}}Ops struct {
	ctx    context.Context
	client *ent.Client
}

func New{{.EntityName}}Ops(ctx context.Context) *{{.EntityName}}Ops {
	return &{{.EntityName}}Ops{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

func (ops *{{.EntityName}}Ops) Get{{.EntityName}}s(limit, offset int, filters ...{{.EntityNameLower}}.Predicate) ([]*ent.{{.EntityName}}, int, error) {
	count, err := ops.client.{{.EntityName}}.Query().
		Where(filters...).
		Count(ops.ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := ops.client.{{.EntityName}}.Query().
		Limit(limit).
		Offset(offset).
		Where(filters...).
		All(ops.ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (ops *{{.EntityName}}Ops) Create{{.EntityName}}(item *ent.{{.EntityName}}) (*ent.{{.EntityName}}, error) {
	return ops.client.{{.EntityName}}.Create().
		SetStatus(item.Status).
		SetCode(item.Code).
		SetDescription(item.Description).
		Save(ops.ctx)
}

func (ops *{{.EntityName}}Ops) Update{{.EntityName}}(id int, item *ent.{{.EntityName}}) (*ent.{{.EntityName}}, error) {
	return ops.client.{{.EntityName}}.UpdateOneID(id).
		SetStatus(item.Status).
		SetCode(item.Code).
		SetDescription(item.Description).
		Save(ops.ctx)
}
