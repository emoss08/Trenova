package search

type Request struct {
	Query  string   `query:"query" validate:"required"`
	Types  []string `query:"types"`
	Limit  int      `query:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset int      `query:"offset,omitempty" validate:"omitempty,min=0"`
}
