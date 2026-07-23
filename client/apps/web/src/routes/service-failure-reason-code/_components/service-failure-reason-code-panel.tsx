import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@trenova/shared/lib/api";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  serviceFailureReasonCodeSchema,
  type ServiceFailureReasonCode,
} from "@/types/service-failure-reason-code";
import { TimeFormat } from "@trenova/shared/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ServiceFailureReasonCodeForm } from "./service-failure-reason-code-form";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

const defaultValues: ServiceFailureReasonCode = {
  active: true,
  code: "",
  label: "",
  description: "",
  category: "Carrier",
  appliesTo: "Both",
  defaultStatusCode: "SD",
  defaultReasonCode: "NS",
  defaultExceptionCode: "",
  defaultNote: "",
  sortOrder: 100,
};

export function ServiceFailureReasonCodePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ServiceFailureReasonCode>) {
  const form = useForm<ServiceFailureReasonCode>({
    resolver: zodResolver(serviceFailureReasonCodeSchema) as Resolver<ServiceFailureReasonCode>,
    defaultValues,
  });

  if (mode === "edit") {
    return (
      <ServiceFailureReasonCodeEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/service-failure-reason-codes/"
      queryKey="service-failure-reason-code-list"
      title="Service Failure Reason Code"
      formComponent={<ServiceFailureReasonCodeForm />}
    />
  );
}

type ServiceFailureReasonCodeEditPanelProps = Pick<
  DataTablePanelProps<ServiceFailureReasonCode>,
  "open" | "onOpenChange" | "row"
> & {
  form: ReturnType<typeof useForm<ServiceFailureReasonCode>>;
};

function ServiceFailureReasonCodeEditPanel({
  open,
  onOpenChange,
  row,
  form,
}: ServiceFailureReasonCodeEditPanelProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  useEffect(() => {
    if (open && row) {
      reset(row, { keepDefaultValues: true });
    }
  }, [open, row, reset]);

  const { mutateAsync } = useApiMutation<
    ServiceFailureReasonCode,
    { action: EditPanelSaveAction; values: ServiceFailureReasonCode },
    unknown,
    ServiceFailureReasonCode
  >({
    mutationFn: async ({ values }) =>
      api.put<ServiceFailureReasonCode>(`/service-failure-reason-codes/${row?.id}/`, values),
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: "Service failure reason code updated successfully",
      });
      void queryClient.invalidateQueries({
        queryKey: ["service-failure-reason-code-list"],
      });

      if (variables.action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: "Service Failure Reason Code",
  });

  const onSubmit = async (values: ServiceFailureReasonCode, action: EditPanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : undefined;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.label ?? "Service Failure Reason Code"}
      description={panelDescription}
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <SplitButton
            options={SAVE_OPTIONS}
            selectedOption={defaultAction}
            onOptionSelect={handleOptionSelect}
            isLoading={isSubmitting}
            loadingText="Saving..."
            formId="panel-edit-form"
          />
        </>
      }
    >
      {!row ? (
        <ComponentLoader message="Loading reason code..." />
      ) : (
        <FormProvider {...form}>
          <Form
            id="panel-edit-form"
            onSubmit={handleSubmit((values) => onSubmit(values, defaultAction))}
          >
            <ServiceFailureReasonCodeForm />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
