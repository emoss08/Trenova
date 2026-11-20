package credential

import "context"

const BundleKey = "enrolment.json"

type Store interface {
	Bundle(ctx context.Context) (Bundle, bool, error)
	Load(ctx context.Context) (Identity, bool, error)
	Save(ctx context.Context, ident *Identity) error
}
