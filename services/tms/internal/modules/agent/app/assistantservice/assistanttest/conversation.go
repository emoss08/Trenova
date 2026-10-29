package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
)

type StubConversations struct {
	conversationports.ConversationRepository

	Thread   *conversation.Thread
	Appended []conversation.Message
	// AppendCtxErr is the state of the context AppendTurn was given, so a test
	// can prove the save survives the request being cancelled.
	AppendCtxErr error
	AppendCalls  int
	// Messages is what ListMessages serves; lastList is what it was asked.
	Messages []conversation.Message
	LastList conversationports.ListMessagesRequest
	// Count is the thread's length as CountMessages reports it.
	Count int
	// Tainted is every time the thread's taint was written.
	Tainted []conversationports.MarkThreadTaintedRequest
	// Contexts is every time the thread's context use was written.
	Contexts []conversationports.UpdateThreadContextRequest
}

func (s *StubConversations) UpdateThreadContext(
	_ context.Context,
	req conversationports.UpdateThreadContextRequest,
) error {
	s.Contexts = append(s.Contexts, req)

	return nil
}

func (s *StubConversations) MarkThreadTainted(
	_ context.Context,
	req conversationports.MarkThreadTaintedRequest,
) error {
	s.Tainted = append(s.Tainted, req)

	return nil
}

func (s *StubConversations) CountMessages(
	context.Context,
	conversationports.CountMessagesRequest,
) (int, error) {
	return s.Count, nil
}

func (s *StubConversations) GetThread(
	context.Context,
	conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	return s.Thread, nil
}

func (s *StubConversations) ListMessages(
	_ context.Context,
	req conversationports.ListMessagesRequest,
) ([]conversation.Message, error) {
	s.LastList = req

	return s.Messages, nil
}

func (s *StubConversations) AppendTurn(
	ctx context.Context,
	req conversationports.AppendTurnRequest,
) ([]conversation.Message, error) {
	s.Appended = req.Messages
	s.AppendCtxErr = ctx.Err()
	s.AppendCalls++
	return req.Messages, nil
}

func (s *StubConversations) UpdateThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	return thread, nil
}

type StubDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	Definition *agentdefinition.Definition
}

func (s *StubDefinitions) GetByID(
	context.Context,
	agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.Definition, nil
}
