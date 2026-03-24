---
title: Golang Select Query [PostgreSQL MySQL]
---

<CoverImage title="Golang Select PostgreSQL MySQL" />

[[toc]]

## API

To see the full list of supported methods, see [SelectQuery](https://pkg.go.dev/github.com/uptrace/bun#SelectQuery).

```go
db.NewSelect().
 With("cte_name", subquery).

 Model(&strct).
 Model(&slice).

 Column("col1", "col2"). // quotes column names
 ColumnExpr("col1, col2"). // arbitrary unsafe expression
 ColumnExpr("count(*)").
 ColumnExpr("count(?)", bun.Ident("id")).
 ColumnExpr("(?) AS alias", subquery).
 ExcludeColumn("col1"). // all columns except col1
 ExcludeColumn("*"). // exclude all columns

 Table("table1", "table2"). // quotes table names
 TableExpr("table1 AS t1"). // arbitrary unsafe expression
 TableExpr("(?) AS alias", subquery).
 ModelTableExpr("table1 AS t1"). // overrides model table name

 Join("JOIN table2 AS t2 ON t2.id = t1.id").
 Join("LEFT JOIN table2 AS t2").JoinOn("t2.id = t1.id").

 WherePK(). // where using primary keys
 Where("id = ?", 123).
 Where("name LIKE ?", "my%").
 Where("? = 123", bun.Ident("id")).
 Where("id IN (?)", bun.In([]int64{1, 2, 3})).
 Where("id IN (?)", subquery).
 Where("FALSE").WhereOr("TRUE").
 WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
  return q.WhereOr("id = 1").
   WhereOr("id = 2")
 }).

 Group("col1", "col2"). // quotes column names
 GroupExpr("lower(col1)"). // arbitrary unsafe expression

 Order("col1 ASC", "col2 DESC"). // quotes column names
 OrderExpr("col1 ASC NULLS FIRST"). // arbitrary unsafe expression

    Having("column_name > ?", 123).

 Limit(100).
 Offset(100).

 For("UPDATE").
 For("SHARE").

 Scan(ctx)
```

## Example

To select into a struct, define a [model](models.md) and use [SelectQuery](https://pkg.go.dev/github.com/uptrace/bun#SelectQuery):

```go
book := new(Book)
err := db.NewSelect().Model(book).Where("id = ?", 123).Scan(ctx)
```

## Count rows

Bun provides [Count](https://pkg.go.dev/github.com/uptrace/bun#SelectQuery.Count) helper to generate `count(*)` queries:

```go
count, err := db.NewSelect().Model((*User)(nil)).Count(ctx)
```

Because selecting and counting rows is a common operation, Bun also provides [ScanAndCount](https://pkg.go.dev/github.com/uptrace/bun#SelectQuery.ScanAndCount):

```go
var users []User
count, err := db.NewSelect().Model(&users).Limit(20).ScanAndCount(ctx)
if err != nil {
 panic(err)
}
fmt.Println(users, count)
```

## EXISTS

You can also use [Exists](https://pkg.go.dev/github.com/uptrace/bun#SelectQuery.Exists) helper to use the corresponding `EXISTS` SQL operator:

```go
exists, err := db.NewSelect().Model((*User)(nil)).Where("name LIKE '%foo%'").Exists(ctx)
if err != nil {
 panic(err)
}
if !exists {
 fmt.Println("such user does not exist")
}
```

```sql
SELECT EXISTS (SELECT * FROM users WHERE name LIKE '%foo%')
```

## Joins

To select a book and manually join the book author:

```go
book := new(Book)
err := db.NewSelect().
    Model(book).
    ColumnExpr("book.*").
    ColumnExpr("a.id AS author__id, a.name AS author__name").
    Join("JOIN authors AS a ON a.id = book.author_id").
    OrderExpr("book.id ASC").
    Limit(1).
    Scan(ctx)
```

```sql
SELECT book.*, a.id AS author__id, a.name AS author__name
FROM books
JOIN authors AS a ON a.id = book.author_id
ORDER BY book.id ASC
LIMIT 1
```

To generate complex joins, use `JoinOn`:

```go
q = q.
    Join("JOIN authors AS a").
    JoinOn("a.id = book.author_id").
    JoinOn("a.deleted_at IS NULL")
```

```sql
JOIN authors AS a ON a.id = book.author_id AND a.deleted_at IS NULL
```

## Subqueries

You can use Bun queries (including `INSERT`, `UPDATE`, and `DELETE` queries) as a subquery:

```go
subq := db.NewSelect().Model((*Book)(nil)).Where("author_id = ?", 1)

err := db.NewSelect().Model().TableExpr("(?) AS book", subq).Scan(ctx, &books)
```

```sql
SELECT * FROM (
  SELECT "book"."id", "book"."title", "book"."text"
  FROM "books" AS "book" WHERE (author_id = 1)
) AS book
```

## Raw queries

Bun also allows you to execute and scan arbitrary raw queries:

```go
type User struct {
 ID int64
 Name string
}

users := make([]User, 0)

err := db.NewRaw(
 "SELECT id, name FROM ? LIMIT ?",
 bun.Ident("users"), 100,
).Scan(ctx, &users)
```

```sql
SELECT id, name FROM "users" LIMIT 100
```
