package workflowrepository

import "errors"

var ErrOnlyDraftVersionsCanBePublished = errors.New("only draft versions can be published")
