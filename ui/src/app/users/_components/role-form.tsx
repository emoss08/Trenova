/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { EnhancedPermissionsSelector } from "@/components/permissions/enhanced-permissions-selector";
import { FormControl, FormGroup } from "@/components/ui/form";
import { roleTypeChoices, statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { PermissionSchema, RoleSchema } from "@/lib/schemas/user-schema";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFormContext } from "react-hook-form";

export function RoleForm() {
  const { control, watch, setValue } = useFormContext<RoleSchema>();

  const { data: permList, isLoading: permListLoading } = useQuery({
    // * TODO: We need to configure this better than just passing in 500 and 0
    // * On the backend, we could probably just do an `all` param to get all permissions at once.
    ...queries.permission.list(500, 0),
  });

  const currentPermissions = watch("permissions") || [];

  const availablePermissions = useMemo(() => {
    return permList?.results || [];
  }, [permList?.results]);

  const handlePermissionsChange = (permissions: PermissionSchema[]) => {
    setValue("permissions", permissions, { shouldDirty: true });
  };

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Current status of the role"
          options={statusChoices}
          isReadOnly
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique name of the role"
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="roleType"
          label="Type"
          placeholder="Type"
          description="Type of the role"
          options={roleTypeChoices}
          isReadOnly
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="description"
          label="Description"
          placeholder="Description"
          description="Description of the role"
        />
      </FormControl>
      <FormControl cols="full">
        <EnhancedPermissionsSelector
          permissions={availablePermissions}
          selectedPermissions={currentPermissions}
          onPermissionsChange={handlePermissionsChange}
          isLoading={permListLoading}
        />
      </FormControl>
    </FormGroup>
  );
}
