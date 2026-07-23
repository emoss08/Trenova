import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps, TableSheetProps } from "@/types/data-table";
import {
  scimGroupRoleMappingFormSchema,
  type SCIMGroupRoleMapping,
  type SCIMGroupRoleMappingFormValues,
} from "@/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { type Resolver, useForm } from "react-hook-form";
import { emptyMapping, scimGroupMappingPanelQueryKey } from "./constants";
import { SCIMGroupMappingForm } from "./mapping-form";

type SCIMGroupMappingRecord = SCIMGroupRoleMappingFormValues & Record<string, unknown>;

type SCIMGroupMappingPanelContextProps = {
  organizationId: string;
  directoryId: string;
};

type SCIMGroupMappingCreatePanelProps = TableSheetProps & SCIMGroupMappingPanelContextProps;

type SCIMGroupMappingEditPanelProps = Pick<
  DataTablePanelProps<SCIMGroupRoleMapping>,
  "open" | "onOpenChange" | "row"
> &
  SCIMGroupMappingPanelContextProps;

type SCIMGroupMappingPanelProps = DataTablePanelProps<SCIMGroupRoleMapping> &
  SCIMGroupMappingPanelContextProps;

function toSCIMGroupMappingFormValues(
  mapping: SCIMGroupRoleMapping,
): SCIMGroupRoleMappingFormValues {
  return {
    ...mapping,
  };
}

function toSCIMGroupMapping(
  values: SCIMGroupRoleMappingFormValues,
  directoryId: string,
): SCIMGroupRoleMapping {
  return {
    ...emptyMapping,
    ...values,
    directoryId,
  };
}

export function SCIMGroupMappingPanel({
  mode,
  open,
  onOpenChange,
  row,
  organizationId,
  directoryId,
}: SCIMGroupMappingPanelProps) {
  if (mode === "edit") {
    return (
      <SCIMGroupMappingEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        organizationId={organizationId}
        directoryId={directoryId}
      />
    );
  }

  return (
    <SCIMGroupMappingCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      organizationId={organizationId}
      directoryId={directoryId}
    />
  );
}

export function SCIMGroupMappingCreatePanel({
  open,
  onOpenChange,
  organizationId,
  directoryId,
}: SCIMGroupMappingCreatePanelProps) {
  const form = useForm<SCIMGroupRoleMappingFormValues>({
    resolver: zodResolver(
      scimGroupRoleMappingFormSchema,
    ) as Resolver<SCIMGroupRoleMappingFormValues>,
    defaultValues: toSCIMGroupMappingFormValues(emptyMapping),
    mode: "onChange",
  });
  const queryKey = scimGroupMappingPanelQueryKey(organizationId, directoryId);

  return (
    <FormCreatePanel<SCIMGroupRoleMappingFormValues, SCIMGroupMappingRecord>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={queryKey}
      title="Group Mapping"
      description="Map an external SCIM group to a Trenova role."
      size="md"
      formComponent={<SCIMGroupMappingForm />}
      mutationFn={async (values) => {
        const saved = await apiService.organizationService.createSCIMGroupRoleMapping(
          organizationId,
          directoryId,
          toSCIMGroupMapping(values, directoryId),
        );
        return toSCIMGroupMappingFormValues(saved);
      }}
    />
  );
}

function SCIMGroupMappingEditPanel({
  open,
  onOpenChange,
  row,
  organizationId,
  directoryId,
}: SCIMGroupMappingEditPanelProps) {
  const form = useForm<SCIMGroupRoleMappingFormValues>({
    resolver: zodResolver(
      scimGroupRoleMappingFormSchema,
    ) as Resolver<SCIMGroupRoleMappingFormValues>,
    defaultValues: toSCIMGroupMappingFormValues(emptyMapping),
    mode: "onChange",
  });
  const queryKey = scimGroupMappingPanelQueryKey(organizationId, directoryId);

  return (
    <FormEditPanel<SCIMGroupRoleMappingFormValues, SCIMGroupMappingRecord>
      open={open}
      onOpenChange={onOpenChange}
      row={row ? (toSCIMGroupMappingFormValues(row) as SCIMGroupMappingRecord) : null}
      form={form}
      queryKey={queryKey}
      title="Group Mapping"
      fieldKey="displayName"
      size="md"
      formComponent={<SCIMGroupMappingForm />}
      mutationFn={async (values) => {
        const saved = await apiService.organizationService.updateSCIMGroupRoleMapping(
          organizationId,
          directoryId,
          toSCIMGroupMapping(values, directoryId),
        );
        return toSCIMGroupMappingFormValues(saved);
      }}
    />
  );
}
