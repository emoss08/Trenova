package dbtype

type RelationshipType string

const (
	RelationshipTypeBelongsTo  = RelationshipType("BelongsTo")
	RelationshipTypeHasOne     = RelationshipType("HasOne")
	RelationshipTypeHasMany    = RelationshipType("HasMany")
	RelationshipTypeManyToMany = RelationshipType("ManyToMany")
	RelationshipTypeCustom     = RelationshipType("Custom")
)

type JoinType string

const (
	JoinTypeLeft  = JoinType("Left")
	JoinTypeRight = JoinType("Right")
	JoinTypeInner = JoinType("Inner")
)
