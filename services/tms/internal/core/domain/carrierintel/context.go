package carrierintel

import "context"

type purposeContextKey struct{}

func WithPurpose(ctx context.Context, purpose Purpose) context.Context {
	return context.WithValue(ctx, purposeContextKey{}, purpose)
}

func PurposeFromContext(ctx context.Context) (Purpose, bool) {
	purpose, ok := ctx.Value(purposeContextKey{}).(Purpose)
	return purpose, ok && purpose.IsValid()
}

func IsInteractiveContext(ctx context.Context) bool {
	purpose, ok := PurposeFromContext(ctx)
	return ok && purpose.IsInteractive()
}
