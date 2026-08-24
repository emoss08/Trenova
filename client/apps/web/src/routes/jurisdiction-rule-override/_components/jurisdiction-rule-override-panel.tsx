import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@trenova/shared/lib/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  jurisdictionRuleOverrideSchema,
  type JurisdictionRuleOverride,
} from "@/types/jurisdiction-rule-override";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { JurisdictionRuleOverrideForm } from "./jurisdiction-rule-override-form";

const QUERY_KEY = "jurisdiction-rule-override-list";

// Every limit starts null, meaning defer to the statute. Seeding numbers here
// would turn a blank form into an override of everything.
const EMPTY: Partial<JurisdictionRuleOverride> = {
  stateId: "",
  maxWidthFeet: null,
  maxHeightFeet: null,
  maxLengthFeet: null,
  maxWeightPounds: null,
  permitLeadTimeDays: null,
  daylightOnly: null,
  holidayRestricted: null,
  reason: "",
};

export function JurisdictionRuleOverridePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<JurisdictionRuleOverride>) {
  const form = useForm<JurisdictionRuleOverride>({
    resolver: zodResolver(jurisdictionRuleOverrideSchema) as Resolver<JurisdictionRuleOverride>,
    defaultValues: EMPTY as JurisdictionRuleOverride,
  });

  if (mode === "edit") {
    return <OverrideEditPanel open={open} onOpenChange={onOpenChange} row={row} form={form} />;
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/jurisdiction-rule-overrides/"
      queryKey={QUERY_KEY}
      title="Carrier Override"
      formComponent={<JurisdictionRuleOverrideForm />}
    />
  );
}

function OverrideEditPanel({
  open,
  onOpenChange,
  row,
  form,
}: Pick<DataTablePanelProps<JurisdictionRuleOverride>, "open" | "onOpenChange" | "row"> & {
  form: ReturnType<typeof useForm<JurisdictionRuleOverride>>;
}) {
  const queryClient = useQueryClient();
  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open && row) {
      reset(row as JurisdictionRuleOverride, { keepDefaultValues: true });
    }
  }, [open, row, reset]);

  const { mutateAsync } = useApiMutation<
    JurisdictionRuleOverride,
    JurisdictionRuleOverride,
    unknown,
    JurisdictionRuleOverride
  >({
    mutationFn: async (values) =>
      api.put<JurisdictionRuleOverride>(`/jurisdiction-rule-overrides/${row?.id}/`, values),
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: "Carrier override updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
    },
    form,
    resourceName: "Carrier Override",
  });

  const removeMutation = useApiMutation({
    mutationFn: async () => api.delete(`/jurisdiction-rule-overrides/${row?.id}/`),
    onSuccess: () => {
      toast.success("Override removed", {
        description: "This jurisdiction reverts to its statutory limits.",
      });
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      onOpenChange(false);
    },
    resourceName: "Carrier Override",
  });

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.state?.name ?? "Carrier Override"}
      description="Applies to your organization only"
      footer={
        <>
          {/* Removing an override loosens this jurisdiction back to the
              statutory limits, so it sits apart from Cancel and Save. */}
          <Button
            type="button"
            variant="destructive"
            onClick={() => removeMutation.mutate(undefined)}
            disabled={removeMutation.isPending}
          >
            {removeMutation.isPending ? "Removing..." : "Remove Override"}
          </Button>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="panel-edit-form" disabled={isSubmitting}>
            {isSubmitting ? "Saving..." : "Save"}
          </Button>
        </>
      }
    >
      {!row ? (
        <ComponentLoader message="Loading Carrier Override..." />
      ) : (
        <FormProvider {...form}>
          <Form
            id="panel-edit-form"
            onSubmit={handleSubmit(async (values) => {
              await mutateAsync(values);
            })}
          >
            <JurisdictionRuleOverrideForm />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
