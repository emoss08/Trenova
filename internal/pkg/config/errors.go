package config

import "github.com/rotisserie/eris"

var (
	ErrConfigNotLoaded      = eris.New("config not loaded")
	ErrInvalidAppName       = eris.New("invalid app name")
	ErrInvalidServerAddress = eris.New("invalid server address")

	// Database errors
	ErrInvalidDBHost = eris.New("invalid database host")
	ErrInvalidDBPort = eris.New("invalid database port")
	ErrInvalidDBName = eris.New("invalid database name")
	ErrInvalidDBUser = eris.New("invalid database username")
)
