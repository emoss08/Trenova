package agent

import "errors"

func isValidEnum(check func() bool, message string) func(any) error {
	return func(_ any) error {
		if !check() {
			return errors.New(message)
		}
		return nil
	}
}
