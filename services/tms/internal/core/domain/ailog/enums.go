package ailog

import "github.com/openai/openai-go/v2"

type Operation string

const (
	OperationClassifyLocation = Operation("ClassifyLocation")
)

func (o Operation) String() string {
	return string(o)
}

type Model openai.ChatModel
