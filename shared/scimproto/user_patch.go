package scimproto

import (
	"strings"

	"github.com/bytedance/sonic"
)

func ApplyUserPatch(user *User, operations []PatchOperation) error {
	for i := range operations {
		if err := applyUserOperation(user, &operations[i]); err != nil {
			return err
		}
	}

	return nil
}

func applyUserOperation(user *User, op *PatchOperation) error {
	if strings.TrimSpace(op.Path) == "" {
		if op.Op == OpRemove {
			return NoTarget("A remove operation needs a path")
		}
		object, err := decodeObject(op.Value)
		if err != nil {
			return err
		}
		for key, value := range object {
			if err = applyUserPath(user, op.Op, key, value); err != nil {
				return err
			}
		}
		return nil
	}

	return applyUserPath(user, op.Op, op.Path, op.Value)
}

func applyUserPath(user *User, op, rawPath string, value sonic.NoCopyRawMessage) error {
	path, err := parsePatchPath(rawPath)
	if err != nil {
		return err
	}
	if strings.HasPrefix(path.attribute, "urn:") {
		return nil
	}

	switch path.attribute {
	case "active":
		return patchActive(user, op, value)
	case "username":
		if op == OpRemove {
			return Mutability("userName cannot be removed")
		}
		return patchString(&user.UserName, op, value)
	case "externalid":
		return patchString(&user.ExternalID, op, value)
	case "displayname":
		return patchString(&user.DisplayName, op, value)
	case "title":
		return patchString(&user.Title, op, value)
	case "timezone":
		return patchString(&user.Timezone, op, value)
	case "locale":
		return patchString(&user.Locale, op, value)
	case "preferredlanguage":
		return patchString(&user.PreferredLanguage, op, value)
	case "name":
		return patchName(user, op, path.sub, value)
	case "emails":
		return patchMultiValue(&user.Emails, op, path, value)
	case "phonenumbers":
		return patchMultiValue(&user.PhoneNumbers, op, path, value)
	default:
		return nil
	}
}

func patchActive(user *User, op string, value sonic.NoCopyRawMessage) error {
	if op == OpRemove {
		return nil
	}
	active, err := decodeBool(value)
	if err != nil {
		return err
	}
	user.Active = BoolOf(active)

	return nil
}

func patchString(target *string, op string, value sonic.NoCopyRawMessage) error {
	if op == OpRemove {
		*target = ""
		return nil
	}
	decoded, err := decodeString(value)
	if err != nil {
		return err
	}
	*target = decoded

	return nil
}

func patchName(user *User, op, sub string, value sonic.NoCopyRawMessage) error {
	if user.Name == nil {
		user.Name = &Name{}
	}

	if sub == "" {
		if op == OpRemove {
			user.Name = &Name{}
			return nil
		}
		var name Name
		if err := sonic.Unmarshal(value, &name); err != nil {
			return InvalidValue("The name value must be an object")
		}
		if op == OpReplace {
			*user.Name = name
			return nil
		}
		mergeName(user.Name, &name)
		return nil
	}

	var target *string
	switch sub {
	case "formatted":
		target = &user.Name.Formatted
	case "givenname":
		target = &user.Name.GivenName
	case "familyname":
		target = &user.Name.FamilyName
	case "middlename":
		target = &user.Name.MiddleName
	case "honorificprefix":
		target = &user.Name.HonorificPrefix
	case "honorificsuffix":
		target = &user.Name.HonorificSuffix
	default:
		return nil
	}

	return patchString(target, op, value)
}

func mergeName(target, source *Name) {
	for _, pair := range []struct {
		dst *string
		src string
	}{
		{&target.Formatted, source.Formatted},
		{&target.GivenName, source.GivenName},
		{&target.FamilyName, source.FamilyName},
		{&target.MiddleName, source.MiddleName},
		{&target.HonorificPrefix, source.HonorificPrefix},
		{&target.HonorificSuffix, source.HonorificSuffix},
	} {
		if pair.src != "" {
			*pair.dst = pair.src
		}
	}
}

func patchMultiValue(
	target *[]MultiValue,
	op string,
	path *patchPath,
	value sonic.NoCopyRawMessage,
) error {
	if path.filter.Empty() {
		if op == OpRemove {
			*target = nil
			return nil
		}
		values, err := decodeMultiValues(value)
		if err != nil {
			return err
		}
		if op == OpReplace {
			*target = values
			return nil
		}
		*target = mergeMultiValues(*target, values)
		return nil
	}

	matches := func(candidate *MultiValue) bool {
		for _, condition := range path.filter.Conditions {
			switch condition.Attribute {
			case "type":
				if !strings.EqualFold(candidate.Type, condition.Value) {
					return false
				}
			case "value":
				if !strings.EqualFold(candidate.Value, condition.Value) {
					return false
				}
			}
		}
		return true
	}

	if op == OpRemove {
		kept := (*target)[:0]
		for i := range *target {
			if !matches(&(*target)[i]) {
				kept = append(kept, (*target)[i])
			}
		}
		*target = kept
		return nil
	}

	index := -1
	for i := range *target {
		if matches(&(*target)[i]) {
			index = i
			break
		}
	}
	if index < 0 {
		entry := MultiValue{}
		if kind, ok := path.filter.Value("type"); ok {
			entry.Type = kind
		}
		*target = append(*target, entry)
		index = len(*target) - 1
	}

	entry := &(*target)[index]
	switch path.sub {
	case "", "value":
		if path.sub == "" {
			var replacement MultiValue
			if err := sonic.Unmarshal(value, &replacement); err == nil &&
				replacement.Value != "" {
				replacement.Type = entry.Type
				*entry = replacement
				return nil
			}
		}
		return patchString(&entry.Value, op, value)
	case "primary":
		primary, err := decodeBool(value)
		if err != nil {
			return err
		}
		entry.Primary = Bool(primary)
		return nil
	case "display":
		return patchString(&entry.Display, op, value)
	default:
		return nil
	}
}

func mergeMultiValues(existing, added []MultiValue) []MultiValue {
	merged := append([]MultiValue(nil), existing...)
	for _, value := range added {
		replaced := false
		for i := range merged {
			if value.Type != "" && strings.EqualFold(merged[i].Type, value.Type) {
				merged[i] = value
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, value)
		}
	}

	return merged
}
