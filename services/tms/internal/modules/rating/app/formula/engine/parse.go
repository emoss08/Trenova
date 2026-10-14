package engine

import (
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/parser"
)

func languageOptions() []expr.Option {
	options := append(BuiltinFunctions(), limitOptions()...)
	options = append(options, disabledBuiltinOptions()...)
	return append(options, expr.MaxNodes(maxExpressionNodes))
}

var parseConfig = sync.OnceValue(func() *conf.Config {
	config := conf.CreateNew()
	for _, option := range languageOptions() {
		option(config)
	}
	for name := range config.Disabled {
		delete(config.Builtins, name)
	}
	return config
})

func parseExpression(expression string) (*parser.Tree, error) {
	return parser.ParseWithConfig(expression, parseConfig())
}
