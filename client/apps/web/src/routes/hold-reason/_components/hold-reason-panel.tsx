import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import {
  SplitButton,
  type SplitButtonOption,
} from "@/components/ui/split-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import type { DataTablePanelProps } from "@/types/data-table";
import { holdReasonSchema, type HoldReason } from "@/types/hold-reason";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { HoldReasonForm } from "./hold-reason-form";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

export function HoldReasonPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<HoldReason>) {
  const form = useForm<HoldReason>({
    resolver: zodResolver(holdReasonSchema) as Resolver<HoldReason>,
    defaultValues: {
      active: true,
      type: "OperationalHold" as const,
      code: "",
      label: "",
      description: "",
      defaultSeverity: "Informational" as const,
      defaultBlocksDispatch: false,
      defaultBlocksDelivery: false,
      defaultBlocksBilling: false,
      defaultVisibleToCustomer: false,
      sortOrder: 100,
    },
  });

  if (mode === "edit") {
    return (
      <HoldReasonEditPanel
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
      url="/hold-reasons/"
      queryKey="hold-reason-list"
      title="Hold Reason"
      formComponent={<HoldReasonForm />}
    />
  );
}

type HoldReasonEditPanelProps = Pick<
  DataTablePanelProps<HoldReason>,
  "open" | "onOpenChange" | "row"
> & {
  form: ReturnType<typeof useForm<HoldReason>>;
};

function HoldReasonEditPanel({
  open,
  onOpenChange,
  row,
  form,
}: HoldReasonEditPanelProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();

  type EditSubmitPayload = {
    action: EditPanelSaveAction;
    values: HoldReason;
  };

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
      reset(row as HoldReason, { keepDefaultValues: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  const { mutateAsync } = useApiMutation<
    HoldReason,
    EditSubmitPayload,
    unknown,
    HoldReason
  >({
    mutationFn: async ({ values }) => {
      const response = await api.put<HoldReason>(
        `/hold-reasons/${row?.id}/`,
        values,
      );
      return response;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["hold-reason-list"],
      });
      const previousRecord = queryClient.getQueryData(["hold-reason-list"]);
      queryClient.setQueryData(["hold-reason-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: "Hold Reason updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: ["hold-reason-list"] });

      const action = variables.action;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: "Hold Reason",
  });

  const onSubmit = useCallback(
    async (values: HoldReason, action: EditPanelSaveAction) => {
      await mutateAsync({ values, action });
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: HoldReason) => {
    return onSubmit(values, defaultAction);
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt as number, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : undefined;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.label ?? "Hold Reason"}
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
        <ComponentLoader message="Loading Hold Reason..." />
      ) : (
        <FormProvider {...form}>
          <Form id="panel-edit-form" onSubmit={handleSubmit(handleFormSubmit)}>
            <HoldReasonForm />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
