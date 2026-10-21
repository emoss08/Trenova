package conversationfileservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

const maxConversationFileBytes = 64 << 20

type Params struct {
	fx.In

	Repo       documentports.DocumentRepository
	Storage    kernelports.Client
	Encryption *encryptionservice.Service
}

// Service reads a file a person attached to a conversation so the system can
// process it, as document intelligence reads an upload. It is not a person
// viewing the file, so no view is recorded. It stands apart from the document
// service, whose dependencies reach the model router that reads through it.
type Service struct {
	repo       documentports.DocumentRepository
	storage    kernelports.Client
	encryption *encryptionservice.Service
}

func New(p Params) documentports.ConversationFileReader {
	return &Service{repo: p.Repo, storage: p.Storage, encryption: p.Encryption}
}

func (s *Service) ReadConversationFile(
	ctx context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.ConversationFile, error) {
	doc, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if !doc.OwnedByConversation() {
		return nil, errortypes.NewNotFoundError("Document not found")
	}

	data, err := storedobject.Read(
		ctx,
		s.storage,
		s.encryption,
		doc.StoragePath,
		storedobject.AAD(req.TenantInfo, doc.StoragePath),
		maxConversationFileBytes,
	)
	if err != nil {
		return nil, err
	}

	return &documentports.ConversationFile{Document: doc, Data: data}, nil
}
