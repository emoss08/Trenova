---
title: Golang Common Table Expressions [PostgreSQL MySQL]
---

<CoverImage title="Golang Common Table Expressions PostgreSQL MySQL" />

[[toc]]

## WITH

Most Bun queries support CTEs via `With` method:

```go
q1 := db.NewSelect()
q2 := db.NewSelect()

q := db.NewInsert().
    With("q1", q1).
    With("q2", q2).
    Table("q1", "q2")
```

For example, you can use CTEs to bulk-delete rows that match some predicates:

```go
const limit = 1000

for {
 subq := db.NewSelect().
  Model((*Comment)(nil)).
  Where("created_at < now() - interval '90 day'").
  Limit(limit)

 res, err := db.NewDelete().
  With("todo", subq).
  Model((*Comment)(nil)).
  Table("todo").
  Where("comment.id = todo.id").
  Exec(ctx)
 if err != nil {
  panic(err)
 }

 num, err := res.RowsAffected()
 if err != nil {
  panic(err)
 }
 if num < limit {
  break
 }
}
```

```sql
WITH todo AS (
    SELECT * FROM comments
    WHERE created_at < now() - interval '90 day'
    LIMIT 1000
)
DELETE FROM comments AS comment USING todo
WHERE comment.id = todo.id
```

Or copy data between tables:

```go
src := db.NewSelect().Model((*Comment)(nil))

res, err := db.NewInsert().
    With("src", src).
    Table("comments_backup", "src").
    Exec(ctx)
```

```sql
WITH src AS (SELECT * FROM comments)
INSERT INTO comments_backups SELECT * FROM src
```

## VALUES

Bun also provides [ValuesQuery](https://pkg.go.dev/github.com/uptrace/bun#ValuesQuery) to help building CTEs:

```go
values := db.NewValues(&[]*Book{book1, book2})

res, err := db.NewUpdate().
    With("_data", values).
    Model((*Book)(nil)).
    Table("_data").
    Set("title = _data.title").
    Set("text = _data.text").
    Where("book.id = _data.id").
    Exec(ctx)
```

```sql
WITH _data (id, title, text) AS (VALUES (1, 'title1', 'text1'), (2, 'title2', 'text2'))
UPDATE books AS book
SET title = _data.title, text = _data.text
FROM _data
WHERE book.id = _data.id
```

## WithOrder

You can also use [WithOrder](https://pkg.go.dev/github.com/uptrace/bun#ValuesQuery.WithOrder) to include row rank in values:

```go
users := []User{
 {ID: 1, "one@my.com"},
 {ID: 2, "two@my.com"},
}

err := db.NewSelect().
 With("data", db.NewValues(&users).WithOrder()).
 Model(&users).
 Where("user.id = data.id").
 OrderExpr("data._order").
 Scan(ctx)
```

```sql
WITH "data" ("id", "email", _order) AS (
  VALUES
    (42::BIGINT, 'one@my.com'::VARCHAR, 0),
    (43::BIGINT, 'two@my.com'::VARCHAR, 1)
)
SELECT "user"."id", "user"."email"
FROM "users" AS "user"
WHERE (user.id = data.id)
ORDER BY data._order
```
