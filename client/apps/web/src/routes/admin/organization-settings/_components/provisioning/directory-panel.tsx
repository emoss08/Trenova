import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import {
  scimDirectoryFormSchema,
  type SCIMDirectory,
  type SCIMDirectoryFormValues,
} from "@trenova/shared/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { type Resolver, useForm } from "react-hook-form";
import { emptyDirectory, scimDirectoryPanelQueryKey } from "./constants";
import { SCIMDirectoryForm } from "./directory-form";
import type { TableSheetProps } from "@trenova/shared/types/data-table";

export type SCIMDirectoryPanelMode = "create" | "edit";

type SCIMDirectoryRecord = SCIMDirectoryFormValues & Record<string, unknown>;

type SCIMDirectoryPanelProps = {
  organizationId: string;
  mode: SCIMDirectoryPanelMode;
  directory: SCIMDirectory | null;
  onSaved: (directory: SCIMDirectory) => Promise<void>;
} & TableSheetProps;

function toSCIMDirectoryFormValues(directory: SCIMDirectory): SCIMDirectoryFormValues {
  return {
    ...directory,
  };
}

function toSCIMDirectory(values: SCIMDirectoryFormValues): SCIMDirectory {
  return {
    ...emptyDirectory,
    ...values,
  };
}

export function SCIMDirectoryPanel({
  organizationId,
  mode,
  open,
  directory,
  onOpenChange,
  onSaved,
}: SCIMDirectoryPanelProps) {
  const form = useForm<SCIMDirectoryFormValues>({
    resolver: zodResolver(scimDirectoryFormSchema) as Resolver<SCIMDirectoryFormValues>,
    defaultValues: toSCIMDirectoryFormValues(emptyDirectory),
    mode: "onChange",
  });
  const queryKey = scimDirectoryPanelQueryKey(organizationId);

  if (mode === "edit") {
    return (
      <FormEditPanel<SCIMDirectoryFormValues, SCIMDirectoryRecord>
        open={open}
        onOpenChange={onOpenChange}
        row={directory ? (toSCIMDirectoryFormValues(directory) as SCIMDirectoryRecord) : null}
        form={form}
        queryKey={queryKey}
        title="SCIM Directory"
        fieldKey="tenantSlug"
        size="md"
        formComponent={<SCIMDirectoryForm />}
        mutationFn={async (values) => {
          const saved = await apiService.organizationService.updateSCIMDirectory(
            organizationId,
            toSCIMDirectory(values),
          );
          await onSaved(saved);
          return toSCIMDirectoryFormValues(saved);
        }}
      />
    );
  }

  return (
    <FormCreatePanel<SCIMDirectoryFormValues, SCIMDirectoryRecord>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={queryKey}
      title="SCIM Directory"
      description="Configure a SCIM tenant before issuing tokens or mapping directory groups."
      size="md"
      formComponent={<SCIMDirectoryForm />}
      mutationFn={async (values) => {
        const saved = await apiService.organizationService.createSCIMDirectory(
          organizationId,
          toSCIMDirectory(values),
        );
        await onSaved(saved);
        return toSCIMDirectoryFormValues(saved);
      }}
    />
  );
}
