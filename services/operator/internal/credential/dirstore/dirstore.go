package dirstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/atomicfile"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/securefile"
)

const (
	IdentityFile = "identity.json"
	dirMode      = 0o700
)

type identityFile struct {
	Kind         string `json:"kind"`
	InstanceID   string `json:"instanceId"`
	APIURL       string `json:"apiUrl"`
	Certificate  string `json:"certificate"`
	PrivateKey   string `json:"privateKey"`
	CABundle     string `json:"caBundle"`
	EnrolledFrom string `json:"enrolledFrom"`
}

type Store struct {
	dir       string
	enrolment string
}

type Params struct {
	Dir       string
	Enrolment string
}

var _ credential.Store = Store{}

func New(p Params) (Store, error) {
	if p.Dir == "" {
		return Store{}, errors.New("dirstore: the credential directory is required")
	}
	if p.Enrolment == "" {
		p.Enrolment = filepath.Join(p.Dir, credential.BundleKey)
	}
	if err := os.MkdirAll(p.Dir, dirMode); err != nil {
		return Store{}, fmt.Errorf("make the credential directory: %w", err)
	}
	info, err := os.Lstat(p.Dir)
	if err != nil {
		return Store{}, fmt.Errorf("read the credential directory: %w", err)
	}
	if !info.IsDir() {
		return Store{}, fmt.Errorf("%w: %s is not a directory", securefile.ErrInsecure, p.Dir)
	}
	if err = os.Chmod(p.Dir, dirMode); err != nil {
		return Store{}, fmt.Errorf("restrict the credential directory: %w", err)
	}
	if err = securefile.CheckOwnDir(p.Dir); err != nil {
		return Store{}, fmt.Errorf("the credential directory: %w", err)
	}
	return Store{dir: p.Dir, enrolment: p.Enrolment}, nil
}

func (s Store) Bundle(context.Context) (credential.Bundle, bool, error) {
	body, err := securefile.ReadSupplied(s.enrolment)
	if errors.Is(err, fs.ErrNotExist) {
		return credential.Bundle{}, false, nil
	}
	if err != nil {
		return credential.Bundle{}, false, fmt.Errorf("read the enrolment bundle: %w", err)
	}
	b, err := credential.ParseBundle(body)
	if err != nil {
		return credential.Bundle{}, false, err
	}
	return b, true, nil
}

func (s Store) Load(context.Context) (credential.Identity, bool, error) {
	path := filepath.Join(s.dir, IdentityFile)
	body, err := securefile.ReadOwn(path)
	if errors.Is(err, fs.ErrNotExist) {
		return credential.Identity{}, false, nil
	}
	if err != nil {
		return credential.Identity{}, false, fmt.Errorf("read the identity: %w", err)
	}
	var stored identityFile
	if err = sonic.Unmarshal(body, &stored); err != nil {
		return credential.Identity{}, false, fmt.Errorf(
			"%w: %w",
			credential.ErrIdentityInvalid,
			err,
		)
	}
	id, err := uuid.Parse(stored.InstanceID)
	if err != nil {
		return credential.Identity{}, false, fmt.Errorf(
			"%w: instanceId: %w",
			credential.ErrIdentityInvalid,
			err,
		)
	}
	ident, err := credential.NewIdentity(&credential.IdentityParams{
		Kind:         credential.KindOf(stored.Kind),
		InstanceID:   id,
		APIURL:       stored.APIURL,
		CertPEM:      []byte(stored.Certificate),
		KeyPEM:       []byte(stored.PrivateKey),
		CAPEM:        []byte(stored.CABundle),
		EnrolledFrom: stored.EnrolledFrom,
	})
	if err != nil {
		return credential.Identity{}, false, fmt.Errorf(
			"%w: %w",
			credential.ErrIdentityInvalid,
			err,
		)
	}
	return ident, true, nil
}

func (s Store) Save(_ context.Context, ident *credential.Identity) error {
	body, err := sonic.Marshal(identityFile{
		Kind:         string(ident.Kind),
		InstanceID:   ident.InstanceID.String(),
		APIURL:       ident.APIURL,
		Certificate:  string(ident.CertPEM),
		PrivateKey:   string(ident.KeyPEM),
		CABundle:     string(ident.CAPEM),
		EnrolledFrom: ident.EnrolledFrom,
	})
	if err != nil {
		return fmt.Errorf("encode the identity: %w", err)
	}
	return atomicfile.Write(filepath.Join(s.dir, IdentityFile), body)
}
