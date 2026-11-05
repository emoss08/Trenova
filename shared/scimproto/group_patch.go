package scimproto

import (
	"strings"

	"github.com/bytedance/sonic"
)

type MemberChange int

const (
	MemberAdd MemberChange = iota
	MemberRemove
	MemberReplace
)

type MemberOperation struct {
	Change MemberChange
	IDs    []string
}

type GroupPatch struct {
	DisplayName *string
	ExternalID  *string
	Members     []MemberOperation
}

func (p *GroupPatch) MemberCount() int {
	count := 0
	for _, op := range p.Members {
		count += len(op.IDs)
	}

	return count
}

func ParseGroupPatch(operations []PatchOperation) (*GroupPatch, error) {
	patch := new(GroupPatch)
	for i := range operations {
		if err := patch.apply(&operations[i]); err != nil {
			return nil, err
		}
	}

	return patch, nil
}

func (p *GroupPatch) apply(op *PatchOperation) error {
	if strings.TrimSpace(op.Path) == "" {
		if op.Op == OpRemove {
			return NoTarget("A remove operation needs a path")
		}
		object, err := decodeObject(op.Value)
		if err != nil {
			return err
		}
		for key, value := range object {
			if err = p.applyPath(op.Op, key, value); err != nil {
				return err
			}
		}
		return nil
	}

	return p.applyPath(op.Op, op.Path, op.Value)
}

func (p *GroupPatch) applyPath(op, rawPath string, value sonic.NoCopyRawMessage) error {
	path, err := parsePatchPath(rawPath)
	if err != nil {
		return err
	}

	switch path.attribute {
	case "displayname":
		if op == OpRemove {
			return Mutability("displayName cannot be removed")
		}
		name, decodeErr := decodeString(value)
		if decodeErr != nil {
			return decodeErr
		}
		p.DisplayName = &name
		return nil
	case "externalid":
		externalID := ""
		if op != OpRemove {
			decoded, decodeErr := decodeString(value)
			if decodeErr != nil {
				return decodeErr
			}
			externalID = decoded
		}
		p.ExternalID = &externalID
		return nil
	case "members":
		return p.applyMembers(op, path, value)
	case "id", "schemas", "meta":
		return nil
	default:
		if strings.HasPrefix(path.attribute, "urn:") {
			return nil
		}
		return InvalidPath("The Group attribute " + rawPath + " cannot be changed")
	}
}

func (p *GroupPatch) applyMembers(op string, path *patchPath, value sonic.NoCopyRawMessage) error {
	if !path.filter.Empty() {
		if op != OpRemove {
			return InvalidPath("Members can be added only through the members attribute")
		}
		id, ok := path.filter.Value("value")
		if !ok || strings.TrimSpace(id) == "" {
			return InvalidPath("A member filter must name the member's value")
		}
		p.Members = append(p.Members, MemberOperation{Change: MemberRemove, IDs: []string{id}})
		return nil
	}

	ids, err := decodeMembers(value)
	if err != nil {
		return err
	}

	switch op {
	case OpAdd:
		if len(ids) > 0 {
			p.Members = append(p.Members, MemberOperation{Change: MemberAdd, IDs: ids})
		}
	case OpReplace:
		p.Members = append(p.Members, MemberOperation{Change: MemberReplace, IDs: ids})
	case OpRemove:
		if len(ids) == 0 {
			p.Members = append(p.Members, MemberOperation{Change: MemberReplace, IDs: []string{}})
			return nil
		}
		p.Members = append(p.Members, MemberOperation{Change: MemberRemove, IDs: ids})
	}

	return nil
}
