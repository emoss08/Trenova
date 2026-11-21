package values

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const ImportKeysKey = "archiveKeys"

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Import struct {
	Spec           *instanceapi.SpecImport
	Settled        bool
	KeysSecret     string
	ImportedBefore bool
}

func ImportKeysSecret(release string) string {
	return Fullname(release) + "-import-keys"
}

func ImportRecord(release string) string {
	return Fullname(release) + "-import"
}

func CheckImport(spec *instanceapi.SpecImport, storage *Storage) error {
	id := spec.Id.String()
	switch {
	case spec.Prefix != "imports/"+id+"/":
		return fmt.Errorf("%w: the import's prefix must be imports/%s/", ErrInvalidSpec, id)
	case !sha256Pattern.MatchString(spec.ExpectSha256):
		return fmt.Errorf("%w: the import's expectSha256", ErrInvalidSpec)
	case len(spec.ArchiveKeys) != 1:
		return fmt.Errorf(
			"%w: the import must carry exactly one archive key, not %d",
			ErrInvalidSpec,
			len(spec.ArchiveKeys),
		)
	case !spec.ArchiveKeys[0].Current || strings.TrimSpace(spec.ArchiveKeys[0].Id) == "":
		return fmt.Errorf(
			"%w: the import's archive key must have an id and be current",
			ErrInvalidSpec,
		)
	case !storage.Configured():
		return fmt.Errorf("%w: an import needs the instance's backups bucket", ErrInvalidSpec)
	case spec.Bucket != BackupsBucket(storage):
		return fmt.Errorf(
			"%w: the import's bucket is not the instance's backups bucket",
			ErrInvalidSpec,
		)
	default:
		return nil
	}
}

func BackupsBucket(storage *Storage) string {
	bucket, _, _ := strings.Cut(strings.TrimPrefix(storage.BackupsPath, "s3://"), "/")
	return bucket
}

func ApplyImport(vals map[string]any, storage *Storage, in *Import) error {
	if in.Spec == nil && !in.ImportedBefore {
		return nil
	}
	merge(vals, map[string]any{"bootstrap": map[string]any{
		"adminSetup": AdminSetupLink,
		"orgName":    "",
		"adminName":  "",
		"adminEmail": "",
	}})
	if in.Spec == nil {
		return nil
	}
	if err := CheckImport(in.Spec, storage); err != nil {
		return err
	}
	merge(vals, map[string]any{keyPostgres: map[string]any{"hibernate": false}})
	if in.Settled {
		merge(vals, map[string]any{"import": map[string]any{
			"id":      in.Spec.Id.String(),
			"settled": true,
		}})
		return nil
	}
	merge(vals, map[string]any{"import": map[string]any{
		"id":           in.Spec.Id.String(),
		"bucket":       in.Spec.Bucket,
		"prefix":       in.Spec.Prefix,
		"expectSha256": in.Spec.ExpectSha256,
		"keysSecret":   in.KeysSecret,
		"settled":      false,
	}})
	return nil
}
