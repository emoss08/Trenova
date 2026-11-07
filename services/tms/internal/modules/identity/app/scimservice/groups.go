package scimservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
)

func (s *Service) ListGroups(
	ctx context.Context,
	req *scimports.ListRequest,
) (*scimports.GroupPage, error) {
	filter, err := scimproto.ParseFilter(req.Filter, scimports.GroupFilters()...)
	if err != nil {
		return nil, err
	}

	start, size := pageBounds(req.StartIndex, req.Count)
	groups, total, err := s.repo.ListGroups(scopedTo(ctx, req.Directory), &scimports.ListQuery{
		DirectoryID: req.Directory.ID,
		Filter:      filter,
		Offset:      start - 1,
		Limit:       max(size, 1),
	})
	if err != nil {
		return nil, err
	}
	if size == 0 {
		groups = nil
	}

	resources := make([]*scimproto.Group, 0, len(groups))
	for _, group := range groups {
		resources = append(resources, groupResource(group, nil))
	}

	return &scimports.GroupPage{Groups: resources, Total: total, StartIndex: start}, nil
}

func (s *Service) GetGroup(
	ctx context.Context,
	req *scimports.GetGroupRequest,
) (*scimproto.Group, error) {
	ctx = scopedTo(ctx, req.Directory)
	group, err := s.repo.GetGroup(ctx, req.Directory.ID, req.GroupID)
	if err != nil {
		return nil, err
	}
	if !req.IncludeMembers {
		return groupResource(group, nil), nil
	}

	return s.groupWithMembers(ctx, req.Directory, group)
}

func (s *Service) CreateGroup(
	ctx context.Context,
	dir *scimports.Directory,
	resource *scimproto.Group,
) (*scimproto.Group, error) {
	ctx = scopedTo(ctx, dir)
	e := &event{
		action:       iam.ProvisioningActionCreate,
		resourceType: resourceTypeGroup,
		externalID:   strings.TrimSpace(resource.ExternalID),
	}

	group, err := groupFrom(resource)
	if err != nil {
		return nil, err
	}
	group.OrganizationID = dir.OrganizationID
	group.BusinessUnitID = dir.BusinessUnitID
	group.DirectoryID = dir.ID

	memberIDs := parseIDs(resource.MemberIDs())
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if createErr := s.repo.CreateGroup(txCtx, group); createErr != nil {
			return createErr
		}
		return s.addMembers(txCtx, dir, group.ID, memberIDs)
	})
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return nil, err
	}

	s.requestReconcileIf(ctx, dir, len(memberIDs) > 0)
	e.resourceID = group.ID
	e.status = statusSucceeded
	e.comment = "SCIM group created"
	e.after = group
	e.metadata = map[string]any{"memberCount": len(memberIDs)}
	s.record(ctx, dir, e)

	return s.groupWithMembers(ctx, dir, group)
}

func (s *Service) ReplaceGroup(
	ctx context.Context,
	dir *scimports.Directory,
	groupID pulid.ID,
	resource *scimproto.Group,
) (*scimproto.Group, error) {
	ctx = scopedTo(ctx, dir)
	group, err := s.repo.GetGroup(ctx, dir.ID, groupID)
	if err != nil {
		return nil, err
	}

	replacement, err := groupFrom(resource)
	if err != nil {
		return nil, err
	}

	before := *group
	e := &event{
		action:       iam.ProvisioningActionUpdate,
		resourceType: resourceTypeGroup,
		resourceID:   groupID,
		externalID:   replacement.ExternalID,
		before:       &before,
	}

	memberIDs := parseIDs(resource.MemberIDs())
	if len(memberIDs) > scimports.MaxMembersPerRequest {
		return nil, scimproto.PayloadTooLarge(
			"The group has more members than one request may carry",
		)
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if markErr := s.repo.MarkGroupPending(txCtx, dir.ID, groupID); markErr != nil {
			return markErr
		}
		group.DisplayName = replacement.DisplayName
		group.ExternalID = replacement.ExternalID
		if saveErr := s.repo.SaveGroup(txCtx, group); saveErr != nil {
			return saveErr
		}
		if clearErr := s.repo.ClearMembers(txCtx, dir.ID, groupID); clearErr != nil {
			return clearErr
		}
		return s.addMembers(txCtx, dir, groupID, memberIDs)
	})
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return nil, err
	}

	s.requestReconcileIf(ctx, dir, true)
	e.status = statusSucceeded
	e.comment = "SCIM group replaced"
	e.after = group
	e.metadata = map[string]any{"memberCount": len(memberIDs)}
	s.record(ctx, dir, e)

	return s.groupWithMembers(ctx, dir, group)
}

func (s *Service) PatchGroup(
	ctx context.Context,
	dir *scimports.Directory,
	groupID pulid.ID,
	operations []scimproto.PatchOperation,
) error {
	patch, err := scimproto.ParseGroupPatch(operations)
	if err != nil {
		return err
	}
	if patch.MemberCount() > scimports.MaxMembersPerRequest {
		return scimproto.PayloadTooLarge("The patch names more members than one request may carry")
	}

	ctx = scopedTo(ctx, dir)
	group, err := s.repo.GetGroup(ctx, dir.ID, groupID)
	if err != nil {
		return err
	}

	before := *group
	e := &event{
		action:       iam.ProvisioningActionUpdate,
		resourceType: resourceTypeGroup,
		resourceID:   groupID,
		externalID:   group.ExternalID,
		before:       &before,
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		return s.applyGroupPatch(txCtx, dir, group, patch)
	})
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return err
	}

	s.requestReconcileIf(ctx, dir, true)
	e.status = statusSucceeded
	e.comment = "SCIM group changed"
	e.after = group
	e.metadata = map[string]any{"memberChanges": patch.MemberCount()}
	s.record(ctx, dir, e)

	return nil
}

func (s *Service) applyGroupPatch(
	ctx context.Context,
	dir *scimports.Directory,
	group *scim.Group,
	patch *scimproto.GroupPatch,
) error {
	if patch.DisplayName != nil || patch.ExternalID != nil {
		if err := s.repo.MarkGroupPending(ctx, dir.ID, group.ID); err != nil {
			return err
		}
		if patch.DisplayName != nil {
			name, err := displayName(*patch.DisplayName)
			if err != nil {
				return err
			}
			group.DisplayName = name
		}
		if patch.ExternalID != nil {
			externalID, err := groupExternalID(*patch.ExternalID)
			if err != nil {
				return err
			}
			group.ExternalID = externalID
		}
		if err := s.repo.SaveGroup(ctx, group); err != nil {
			return err
		}
	}

	for _, op := range patch.Members {
		ids := parseIDs(op.IDs)
		switch op.Change {
		case scimproto.MemberAdd:
			if err := s.addMembers(ctx, dir, group.ID, ids); err != nil {
				return err
			}
		case scimproto.MemberRemove:
			if err := s.repo.MarkPending(ctx, dir.ID, ids); err != nil {
				return err
			}
			if err := s.repo.RemoveMembers(ctx, &scimports.MemberChange{
				DirectoryID: dir.ID,
				GroupID:     group.ID,
				UserIDs:     ids,
			}); err != nil {
				return err
			}
		case scimproto.MemberReplace:
			if err := s.repo.MarkGroupPending(ctx, dir.ID, group.ID); err != nil {
				return err
			}
			if err := s.repo.ClearMembers(ctx, dir.ID, group.ID); err != nil {
				return err
			}
			if err := s.addMembers(ctx, dir, group.ID, ids); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) DeleteGroup(
	ctx context.Context,
	dir *scimports.Directory,
	groupID pulid.ID,
) error {
	ctx = scopedTo(ctx, dir)
	group, err := s.repo.GetGroup(ctx, dir.ID, groupID)
	if err != nil {
		return err
	}

	e := &event{
		action:       iam.ProvisioningActionDelete,
		resourceType: resourceTypeGroup,
		resourceID:   groupID,
		externalID:   group.ExternalID,
		before:       group,
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if markErr := s.repo.MarkGroupPending(txCtx, dir.ID, groupID); markErr != nil {
			return markErr
		}
		return s.repo.DeleteGroup(txCtx, dir.ID, groupID)
	})
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return err
	}

	s.requestReconcileIf(ctx, dir, true)
	e.status = statusSucceeded
	e.comment = "SCIM group deleted"
	s.record(ctx, dir, e)

	return nil
}

func (s *Service) addMembers(
	ctx context.Context,
	dir *scimports.Directory,
	groupID pulid.ID,
	userIDs []pulid.ID,
) error {
	if len(userIDs) == 0 {
		return nil
	}
	if len(userIDs) > scimports.MaxMembersPerRequest {
		return scimproto.PayloadTooLarge("The group has more members than one request may carry")
	}

	linked, err := s.repo.CountUsers(ctx, dir.ID, userIDs)
	if err != nil {
		return err
	}
	if linked != len(userIDs) {
		return scimproto.InvalidValue("Every member must be a user this directory provisioned")
	}

	if err = s.repo.AddMembers(ctx, &scimports.MemberChange{
		DirectoryID: dir.ID,
		GroupID:     groupID,
		UserIDs:     userIDs,
	}); err != nil {
		return err
	}

	return s.repo.MarkPending(ctx, dir.ID, userIDs)
}

func (s *Service) requestReconcileIf(ctx context.Context, dir *scimports.Directory, changed bool) {
	if !changed {
		return
	}
	s.requester.RequestReconcile(ctx, dir.TenantInfo(), dir.ID)
}

func (s *Service) groupWithMembers(
	ctx context.Context,
	dir *scimports.Directory,
	group *scim.Group,
) (*scimproto.Group, error) {
	members, err := s.repo.ListMembers(ctx, dir.ID, group.ID, scimports.MaxMembersReturned)
	if err != nil {
		return nil, err
	}

	return groupResource(group, members), nil
}

func groupFrom(resource *scimproto.Group) (*scim.Group, error) {
	name, err := displayName(resource.DisplayName)
	if err != nil {
		return nil, err
	}
	externalID, err := groupExternalID(resource.ExternalID)
	if err != nil {
		return nil, err
	}

	return &scim.Group{DisplayName: name, ExternalID: externalID}, nil
}

func displayName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", scimproto.InvalidValue("displayName is required")
	}
	if len(name) > maxAttributeLength {
		return "", scimproto.InvalidValue("displayName cannot be longer than 255 characters")
	}

	return name, nil
}

func groupExternalID(raw string) (string, error) {
	externalID := strings.TrimSpace(raw)
	if len(externalID) > maxAttributeLength {
		return "", scimproto.InvalidValue("externalId cannot be longer than 255 characters")
	}

	return externalID, nil
}

func groupResource(group *scim.Group, members []*scimports.MemberRecord) *scimproto.Group {
	resource := &scimproto.Group{
		Schemas:     []string{scimproto.SchemaGroup},
		ID:          group.ID.String(),
		ExternalID:  group.ExternalID,
		DisplayName: group.DisplayName,
		Meta: &scimproto.Meta{
			ResourceType: scimproto.ResourceTypeGroup,
			Created:      timestamp(group.CreatedAt),
			LastModified: timestamp(group.UpdatedAt),
		},
	}

	if len(members) > 0 {
		resource.Members = make([]scimproto.MemberRef, 0, len(members))
		for _, member := range members {
			resource.Members = append(resource.Members, scimproto.MemberRef{
				Value:   member.UserID.String(),
				Display: member.Name,
				Type:    scimproto.ResourceTypeUser,
			})
		}
	}

	return resource
}
