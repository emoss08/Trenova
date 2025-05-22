package resourcesqltype

import "github.com/rotisserie/eris"

type SQLKeyword string

const (
	Select      = SQLKeyword("SELECT")
	From        = SQLKeyword("FROM")
	Where       = SQLKeyword("WHERE")
	And         = SQLKeyword("AND")
	Join        = SQLKeyword("JOIN")
	LeftJoin    = SQLKeyword("LEFT JOIN")
	RightJoin   = SQLKeyword("RIGHT JOIN")
	InnerJoin   = SQLKeyword("INNER JOIN")
	On          = SQLKeyword("ON")
	GroupBy     = SQLKeyword("GROUP BY")
	OrderBy     = SQLKeyword("ORDER BY")
	Limit       = SQLKeyword("LIMIT")
	Offset      = SQLKeyword("OFFSET")
	InsertInto  = SQLKeyword("INSERT INTO")
	Values      = SQLKeyword("VALUES")
	Update      = SQLKeyword("UPDATE")
	Delete      = SQLKeyword("DELETE")
	Create      = SQLKeyword("CREATE")
	Table       = SQLKeyword("TABLE")
	As          = SQLKeyword("AS")
	Null        = SQLKeyword("NULL")
	Not         = SQLKeyword("NOT")
	Is          = SQLKeyword("IS")
	Between     = SQLKeyword("BETWEEN")
	Exists      = SQLKeyword("EXISTS")
	All         = SQLKeyword("ALL")
	Any         = SQLKeyword("ANY")
	Some        = SQLKeyword("SOME")
	Count       = SQLKeyword("COUNT")
	Sum         = SQLKeyword("SUM")
	Avg         = SQLKeyword("AVG")
	Max         = SQLKeyword("MAX")
	Min         = SQLKeyword("MIN")
	Distinct    = SQLKeyword("DISTINCT")
	Set         = SQLKeyword("SET")
	DeleteFrom  = SQLKeyword("DELETE FROM")
	CreateTable = SQLKeyword("CREATE TABLE")
	AlterTable  = SQLKeyword("ALTER TABLE")
	DropTable   = SQLKeyword("DROP TABLE")
	Or          = SQLKeyword("OR")
	In          = SQLKeyword("IN")
	IsNull      = SQLKeyword("IS NULL")
	IsNotNull   = SQLKeyword("IS NOT NULL")
	Like        = SQLKeyword("LIKE")
	Case        = SQLKeyword("CASE")
	When        = SQLKeyword("WHEN")
	Then        = SQLKeyword("THEN")
	Else        = SQLKeyword("ELSE")
	End         = SQLKeyword("END")
	Into        = SQLKeyword("INTO")
	Check       = SQLKeyword("CHECK")
)

func (k SQLKeyword) String() string {
	return string(k)
}

func SQLKeywordFromString(s string) (SQLKeyword, error) {
	return SQLKeyword(s), nil
}

var AvailableKeywords = []SQLKeyword{
	Select,
	From,
	Where,
	Join,
	LeftJoin,
	RightJoin,
	InnerJoin,
	On,
	GroupBy,
	OrderBy,
	Limit,
	Offset,
	InsertInto,
	Values,
	Update,
	Set,
	DeleteFrom,
	CreateTable,
	AlterTable,
	DropTable,
	And,
	Or,
	Not,
	As,
	In,
	IsNull,
	IsNotNull,
	Like,
	Between,
	Exists,
	All,
	Any,
	Count,
	Sum,
	Avg,
	Min,
	Max,
	Distinct,
	Case,
	When,
	Then,
	Else,
	End,
}

type Keyword string

const (
	KeywordYes = Keyword("YES")
)

func (k Keyword) String() string {
	return string(k)
}

func KeywordFromString(s string) (Keyword, error) {
	if s == KeywordYes.String() {
		return KeywordYes, nil
	}

	return "", eris.New("invalid keyword")
}

// isColumnFocusedContext checks if the context focuses on column suggestions
func IsColumnFocusedContext(lastKeyword string) bool {
	return lastKeyword == Select.String() ||
		lastKeyword == Where.String() ||
		lastKeyword == On.String() ||
		lastKeyword == GroupBy.String() ||
		lastKeyword == OrderBy.String() ||
		lastKeyword == Set.String() ||
		lastKeyword == And.String() ||
		lastKeyword == Or.String()
}

// isTableFocusedContext checks if the context focuses on table suggestions
func IsTableFocusedContext(lastKeyword string) bool {
	return lastKeyword == From.String() ||
		lastKeyword == Join.String() ||
		lastKeyword == Update.String() ||
		lastKeyword == Into.String()
}
