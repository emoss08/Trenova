package keeltest

//go:generate go tool keel gen -tags integration -out internal/testutil/keeltest/{name}db -kinds internal/testutil/keeltest/keelkinds -tables internal/testutil/keeltest/keeltables .
//go:generate go run buildtag.go keeltestdb keeltables
