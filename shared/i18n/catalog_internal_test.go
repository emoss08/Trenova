package i18n

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCatalogsLayersEditionCatalogsOverTheBase(t *testing.T) {
	t.Parallel()

	base := fstest.MapFS{}
	for _, locale := range supported {
		base["catalogs/"+string(locale)+".json"] = &fstest.MapFile{
			Data: []byte(`{"Shared": "shared-` + string(locale) + `"}`),
		}
	}
	edition := fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{"Signup closed": "Registro cerrado", "Shared": "override"}`)},
	}

	loaded, err := readCatalogs(base, "catalogs", []fs.FS{edition})
	require.NoError(t, err)

	assert.Equal(t, "Registro cerrado", loaded[ES]["Signup closed"])
	assert.Equal(t, "override", loaded[ES]["Shared"])
	assert.Equal(t, "shared-zh-TW", loaded[ZhTW]["Shared"])
	assert.NotContains(t, loaded[ZhTW], "Signup closed")
}

func TestReadCatalogsRejectsAMalformedEditionCatalog(t *testing.T) {
	t.Parallel()

	base := fstest.MapFS{}
	for _, locale := range supported {
		base["catalogs/"+string(locale)+".json"] = &fstest.MapFile{Data: []byte(`{}`)}
	}
	edition := fstest.MapFS{"es.json": &fstest.MapFile{Data: []byte(`{`)}}

	_, err := readCatalogs(base, "catalogs", []fs.FS{edition})
	require.ErrorContains(t, err, "parsing es.json")
}

func TestRegisterCatalogFSRefusesNil(t *testing.T) {
	t.Parallel()

	require.Error(t, RegisterCatalogFS(nil))
}
