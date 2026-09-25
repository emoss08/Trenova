package aiaudit

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
)

const (
	// HashVersionSigned is an HMAC-SHA256 chain under a key held outside the
	// database.
	HashVersionSigned int16 = 1
	// HashVersionUnsigned is a plain SHA-256 chain, written when no key is
	// configured. It still shows a row changed by mistake, but not one
	// rewritten by somebody able to recompute the chain.
	HashVersionUnsigned int16 = 2

	chainDomain = "trenova.ai-audit/v1"
	costScale   = 6
)

// GenesisHash is the prev_hash of a tenant's first row.
var GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

var (
	ErrChainKeyRequired = errors.New("a signed row needs its chain key")
	ErrUnknownHashVer   = errors.New("unknown AI audit hash version")

	canonicalAPI = sonic.Config{SortMapKeys: true, UseNumber: true}.Froze()
)

// ChainKey is one secret the chain is signed with, named by its id so rows
// signed before a rotation can still be checked.
type ChainKey struct {
	ID     string
	Secret []byte
}

// canonicalRecord fixes the fields a row's hash covers and the order they are
// written in. Every content column is here; the hash, the key id and the
// version travel in the message header instead, and created_at is left out
// because the database writes it.
type canonicalRecord struct {
	ID                     string   `json:"id"`
	OrganizationID         string   `json:"organizationId"`
	BusinessUnitID         string   `json:"businessUnitId"`
	Seq                    int64    `json:"seq"`
	SourceKey              string   `json:"sourceKey"`
	OccurredAt             int64    `json:"occurredAt"`
	RecordedAt             int64    `json:"recordedAt"`
	Kind                   string   `json:"kind"`
	Outcome                string   `json:"outcome"`
	PrincipalType          string   `json:"principalType"`
	PrincipalID            string   `json:"principalId"`
	OnBehalfOfUserID       string   `json:"onBehalfOfUserId"`
	OnBehalfOfUserName     string   `json:"onBehalfOfUserName"`
	DecidedByUserID        string   `json:"decidedByUserId"`
	DecidedByUserName      string   `json:"decidedByUserName"`
	AgentDefinitionID      string   `json:"agentDefinitionId"`
	AgentDefinitionVersion *int64   `json:"agentDefinitionVersion"`
	AgentName              string   `json:"agentName"`
	OwnerKind              string   `json:"ownerKind"`
	OwnerID                string   `json:"ownerId"`
	RunID                  string   `json:"runId"`
	TurnID                 string   `json:"turnId"`
	ThreadID               string   `json:"threadId"`
	ProposalID             string   `json:"proposalId"`
	PlanID                 string   `json:"planId"`
	DecisionID             string   `json:"decisionId"`
	StepKey                string   `json:"stepKey"`
	CallID                 string   `json:"callId"`
	DelegateCallID         string   `json:"delegateCallId"`
	ParentOwnerID          string   `json:"parentOwnerId"`
	TraceID                string   `json:"traceId"`
	SpanID                 string   `json:"spanId"`
	ProviderID             string   `json:"providerId"`
	ProviderKind           string   `json:"providerKind"`
	Model                  string   `json:"model"`
	Attempt                *int     `json:"attempt"`
	Failover               bool     `json:"failover"`
	InputTokens            int      `json:"inputTokens"`
	OutputTokens           int      `json:"outputTokens"`
	ReasoningTokens        int      `json:"reasoningTokens"`
	CacheReadTokens        int      `json:"cacheReadTokens"`
	CacheWriteTokens       int      `json:"cacheWriteTokens"`
	CostUSD                *string  `json:"costUsd"`
	LatencyMs              *int64   `json:"latencyMs"`
	ToolName               string   `json:"toolName"`
	ToolEffect             string   `json:"toolEffect"`
	EgressClass            string   `json:"egressClass"`
	Tier                   string   `json:"tier"`
	TierSource             string   `json:"tierSource"`
	HeldBy                 []string `json:"heldBy"`
	Reason                 string   `json:"reason"`
	Arguments              any      `json:"arguments"`
	ArgumentSensitivity    any      `json:"argumentSensitivity"`
	RedactedPaths          []string `json:"redactedPaths"`
	ArgumentsTruncated     bool     `json:"argumentsTruncated"`
	ResultSummary          string   `json:"resultSummary"`
	EntityType             string   `json:"entityType"`
	EntityID               string   `json:"entityId"`
	VersionBefore          *int64   `json:"versionBefore"`
	VersionAfter           *int64   `json:"versionAfter"`
	WindowStart            int64    `json:"windowStart"`
	WindowEnd              int64    `json:"windowEnd"`
	Tainted                bool     `json:"tainted"`
	Taint                  any      `json:"taint"`
	ExternalContent        bool     `json:"externalContent"`
	Simulated              bool     `json:"simulated"`
	Purpose                string   `json:"purpose"`
	Reconstructed          bool     `json:"reconstructed"`
}

// CanonicalBytes is the row's content as the hash reads it: a fixed field
// order, map keys sorted, and every number written as its exact decimal so a
// value read back from jsonb hashes as it did before it was stored.
func CanonicalBytes(e *AIAuditEvent) ([]byte, error) {
	arguments, err := CanonicalJSONValue(e.Arguments)
	if err != nil {
		return nil, fmt.Errorf("canonicalize arguments: %w", err)
	}
	sensitivity, err := CanonicalJSONValue(e.ArgumentSensitivity)
	if err != nil {
		return nil, fmt.Errorf("canonicalize argument sensitivity: %w", err)
	}
	taint, err := CanonicalJSONValue(e.Taint)
	if err != nil {
		return nil, fmt.Errorf("canonicalize taint: %w", err)
	}

	record := canonicalRecord{
		ID:                     e.ID.String(),
		OrganizationID:         e.OrganizationID.String(),
		BusinessUnitID:         e.BusinessUnitID.String(),
		Seq:                    e.Seq,
		SourceKey:              e.SourceKey,
		OccurredAt:             e.OccurredAt,
		RecordedAt:             e.RecordedAt,
		Kind:                   string(e.Kind),
		Outcome:                string(e.Outcome),
		PrincipalType:          string(e.PrincipalType),
		PrincipalID:            e.PrincipalID,
		OnBehalfOfUserID:       e.OnBehalfOfUserID.String(),
		OnBehalfOfUserName:     e.OnBehalfOfUserName,
		DecidedByUserID:        e.DecidedByUserID.String(),
		DecidedByUserName:      e.DecidedByUserName,
		AgentDefinitionID:      e.AgentDefinitionID.String(),
		AgentDefinitionVersion: e.AgentDefinitionVersion,
		AgentName:              e.AgentName,
		OwnerKind:              e.OwnerKind,
		OwnerID:                e.OwnerID.String(),
		RunID:                  e.RunID.String(),
		TurnID:                 e.TurnID.String(),
		ThreadID:               e.ThreadID.String(),
		ProposalID:             e.ProposalID.String(),
		PlanID:                 e.PlanID.String(),
		DecisionID:             e.DecisionID.String(),
		StepKey:                e.StepKey,
		CallID:                 e.CallID,
		DelegateCallID:         e.DelegateCallID,
		ParentOwnerID:          e.ParentOwnerID.String(),
		TraceID:                e.TraceID,
		SpanID:                 e.SpanID,
		ProviderID:             e.ProviderID.String(),
		ProviderKind:           e.ProviderKind,
		Model:                  e.Model,
		Attempt:                e.Attempt,
		Failover:               e.Failover,
		InputTokens:            e.InputTokens,
		OutputTokens:           e.OutputTokens,
		ReasoningTokens:        e.ReasoningTokens,
		CacheReadTokens:        e.CacheReadTokens,
		CacheWriteTokens:       e.CacheWriteTokens,
		CostUSD:                canonicalCost(e.CostUSD),
		LatencyMs:              e.LatencyMs,
		ToolName:               e.ToolName,
		ToolEffect:             e.ToolEffect,
		EgressClass:            e.EgressClass,
		Tier:                   e.Tier,
		TierSource:             e.TierSource,
		HeldBy:                 nonNilStrings(e.HeldBy),
		Reason:                 e.Reason,
		Arguments:              arguments,
		ArgumentSensitivity:    sensitivity,
		RedactedPaths:          nonNilStrings(e.RedactedPaths),
		ArgumentsTruncated:     e.ArgumentsTruncated,
		ResultSummary:          e.ResultSummary,
		EntityType:             e.EntityType,
		EntityID:               e.EntityID,
		VersionBefore:          e.VersionBefore,
		VersionAfter:           e.VersionAfter,
		WindowStart:            e.WindowStart,
		WindowEnd:              e.WindowEnd,
		Tainted:                e.Tainted,
		Taint:                  taint,
		ExternalContent:        e.ExternalContent,
		Simulated:              e.Simulated,
		Purpose:                string(e.Purpose),
		Reconstructed:          e.Reconstructed,
	}

	return canonicalAPI.Marshal(&record)
}

// ComputeHash is the row's hash under its own prev_hash, key id and version.
// A signed row needs the key it names; an unsigned row takes none.
func ComputeHash(e *AIAuditEvent, key *ChainKey) (string, error) {
	var mac hash.Hash
	switch e.HashVersion {
	case HashVersionSigned:
		if key == nil || len(key.Secret) == 0 || key.ID != e.HashKeyID {
			return "", ErrChainKeyRequired
		}
		mac = hmac.New(sha256.New, key.Secret)
	case HashVersionUnsigned:
		mac = sha256.New()
	default:
		return "", fmt.Errorf("%w: %d", ErrUnknownHashVer, e.HashVersion)
	}

	content, err := CanonicalBytes(e)
	if err != nil {
		return "", err
	}

	writeHeader(mac, e)
	_, _ = mac.Write(content)

	return hex.EncodeToString(mac.Sum(nil)), nil
}

func writeHeader(mac hash.Hash, e *AIAuditEvent) {
	_, _ = mac.Write([]byte(chainDomain))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write(strconv.AppendInt(nil, int64(e.HashVersion), 10))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(e.HashKeyID))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(e.PrevHash))
	_, _ = mac.Write([]byte{'\n'})
}

// Seal links a row to the one before it and signs it: with the key when one
// is given, as an unsigned SHA-256 link when it is nil.
func Seal(e *AIAuditEvent, prevHash string, key *ChainKey) error {
	e.PrevHash = prevHash
	if key != nil {
		e.HashVersion = HashVersionSigned
		e.HashKeyID = key.ID
	} else {
		e.HashVersion = HashVersionUnsigned
		e.HashKeyID = ""
	}

	sum, err := ComputeHash(e, key)
	if err != nil {
		return err
	}
	e.Hash = sum

	return nil
}

// VerifyHash reports whether a row still hashes to what it recorded.
func VerifyHash(e *AIAuditEvent, key *ChainKey) (bool, error) {
	sum, err := ComputeHash(e, key)
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare([]byte(sum), []byte(e.Hash)) == 1, nil
}

// CanonicalJSONValue turns any JSON-shaped value into the form the hash and
// the jsonb column both hold: maps of string keys, lists, strings, booleans,
// nil, and numbers as exact decimals.
func CanonicalJSONValue(value any) (any, error) {
	return jsonutils.CanonicalValue(value)
}

func canonicalCost(cost *decimal.Decimal) *string {
	if cost == nil {
		return nil
	}
	fixed := cost.StringFixed(costScale)

	return &fixed
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
