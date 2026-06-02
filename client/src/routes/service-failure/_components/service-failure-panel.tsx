import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  serviceFailureUpdateSchema,
  type ServiceFailure,
  type ServiceFailureUpdate,
} from "@/types/service-failure";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ServiceFailureForm } from "./service-failure-form";
import {
  ServiceFailureStopContext,
  serviceFailureStopSummaryFromFailure,
} from "./service-failure-stop-context";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

const defaultValues: ServiceFailureUpdate = {
  id: "",
  shipmentId: "",
  reasonCodeId: "",
  clearReasonCode: false,
  notes: "",
  internalNotes: "",
  x12StatusCodeOverride: "",
  x12ReasonCodeOverride: "",
  x12ExceptionCode: "",
  version: 0,
};

function toUpdate(row: ServiceFailure): ServiceFailureUpdate {
  return {
    id: row.id ?? "",
    shipmentId: row.shipmentId,
    reasonCodeId: row.reasonCodeId ?? "",
    clearReasonCode: false,
    notes: row.notes,
    internalNotes: row.internalNotes,
    x12StatusCodeOverride: row.x12StatusCodeOverride,
    x12ReasonCodeOverride: row.x12ReasonCodeOverride,
    x12ExceptionCode: row.x12ExceptionCode,
    version: row.version ?? 0,
  };
}

export function ServiceFailurePanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<ServiceFailure>) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const form = useForm<ServiceFailureUpdate>({
    resolver: zodResolver(serviceFailureUpdateSchema) as Resolver<ServiceFailureUpdate>,
    defaultValues,
  });
  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;
  const terminal = row?.status === "Resolved" || row?.status === "Voided";

  const handleClose = () => {
    onOpenChange(false);
    reset(defaultValues);
  };

  useEffect(() => {
    if (open && row) {
      reset(toUpdate(row), { keepDefaultValues: true });
    }
  }, [open, row, reset]);

  const { mutateAsync } = useApiMutation<
    ServiceFailure,
    { action: EditPanelSaveAction; values: ServiceFailureUpdate },
    unknown,
    ServiceFailureUpdate
  >({
    mutationFn: async ({ values }) => apiService.serviceFailureService.update(values.id, values),
    onSuccess: (_data, variables) => {
      toast.success("Service failure updated");
      void queryClient.invalidateQueries({ queryKey: ["service-failure-list"] });
      void queryClient.invalidateQueries({
        queryKey: ["serviceFailure", "list-by-shipment", row?.shipmentId],
      });

      if (variables.action === "save-close") {
        reset(defaultValues);
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: "Service Failure",
  });

  const onSubmit = async (values: ServiceFailureUpdate, action: EditPanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.number ?? "Service Failure"}
      description={row ? `${row.status} · ${row.lateMinutes} minute(s) after grace` : undefined}
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
        <ComponentLoader message="Loading service failure..." />
      ) : (
        <FormProvider {...form}>
          <Form
            id="panel-edit-form"
            onSubmit={handleSubmit((values) => onSubmit(values, defaultAction))}
          >
            <ServiceFailureStopContext
              summary={serviceFailureStopSummaryFromFailure(row)}
              variant="panel"
              className="mb-4"
            />
            <ServiceFailureForm disabled={terminal} stopType={row.stopType} />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
