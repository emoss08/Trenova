package document

import "fmt"

func errInvalidProcessingProfile(profile ProcessingProfile) error {
	return fmt.Errorf("invalid document processing profile %q", profile)
}
