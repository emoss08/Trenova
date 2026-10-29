package ediprofilesecret

import (
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Encryption *encryptionservice.Service
}

type Service struct {
	encryption *encryptionservice.Service
}

func New(p Params) *Service {
	return &Service{
		encryption: p.Encryption,
	}
}
