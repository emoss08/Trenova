import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import {
  scimGroupRoleMappingFormSchema,
  type SCIMGroupRoleMapping,
  type SCIMGroupRoleMappingFormValues,
} from "@/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { type Resolver, useForm } from "react-hook-form";
import { emptyMapping, scimGroupMappingPanelQueryKey } from "./constants";
import { SCIMGroupMappingForm } from "./mapping-form";

export type SCIMGroupMappingPanelMode = "create" | "edit";

type SCIMGroupMappingRecord = SCIMGroupRoleMappingFormValues & Record<string, unknown>;

type SCIMGroupMappingPanelProps = {
  organizationId: string;
  directoryId: string;
  mode: SCIMGroupMappingPanelMode;
  open: boolean;
  mapping: SCIMGroupRoleMapping | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => Promise<void>;
};

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
  organizationId,
  directoryId,
  mode,
  open,
  mapping,
  onOpenChange,
  onSaved,
}: SCIMGroupMappingPanelProps) {
  const form = useForm<SCIMGroupRoleMappingFormValues>({
    resolver: zodResolver(
      scimGroupRoleMappingFormSchema,
    ) as Resolver<SCIMGroupRoleMappingFormValues>,
    defaultValues: toSCIMGroupMappingFormValues(emptyMapping),
    mode: "onChange",
  });
  const queryKey = scimGroupMappingPanelQueryKey(organizationId, directoryId);

  if (mode === "edit") {
    return (
      <FormEditPanel<SCIMGroupRoleMappingFormValues, SCIMGroupMappingRecord>
        open={open}
        onOpenChange={onOpenChange}
        row={mapping ? (toSCIMGroupMappingFormValues(mapping) as SCIMGroupMappingRecord) : null}
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
          await onSaved();
          return toSCIMGroupMappingFormValues(saved);
        }}
      />
    );
  }

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
        await onSaved();
        return toSCIMGroupMappingFormValues(saved);
      }}
    />
  );
}
