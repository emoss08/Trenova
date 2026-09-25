package jsonutils

import (
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
)

// ObjectKeyOrder reads an encoded object's top-level keys in the order they
// were written, which for an encoded struct is the order its fields are
// declared in. A decoded map has no order of its own.
func ObjectKeyOrder(encoded []byte) []string {
	root, err := sonic.Get(encoded)
	if err != nil {
		return nil
	}

	keys := make([]string, 0, 32)
	if err = root.ForEach(func(path ast.Sequence, _ *ast.Node) bool {
		if path.Key != nil {
			keys = append(keys, *path.Key)
		}

		return true
	}); err != nil {
		return nil
	}

	return keys
}
