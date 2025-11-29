package documenttemplate

import "errors"

var (
	ErrCannotDeleteSystemTemplate = errors.New("cannot delete system template")
	ErrCannotEditArchivedTemplate = errors.New("cannot edit archived template")
	ErrTemplateNotActive          = errors.New("template is not active")
)
