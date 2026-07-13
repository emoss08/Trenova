import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import { rateTableSchema, type RateTable, type RateTableRow } from "@/types/rate-table";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { RateTableForm } from "./rate-table-form";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

const DEFAULT_VALUES = {
  active: true,
  name: "",
  key: "",
  description: "",
  lookupType: "Exact" as const,
  entries: [
    {
      matchKey: "",
      rangeMin: null,
      rangeMax: null,
      value: undefined,
      sortOrder: 0,
    },
  ],
};

export function RateTablePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RateTableRow>) {
  const form = useForm<RateTable>({
    resolver: zodResolver(rateTableSchema) as Resolver<RateTable>,
    defaultValues: DEFAULT_VALUES,
  });

  if (mode === "edit") {
    return <RateTableEditPanel open={open} onOpenChange={onOpenChange} row={row} form={form} />;
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/rate-tables/"
      queryKey="rate-table-list"
      title="Rate Table"
      size="xl"
      formComponent={<RateTableForm />}
    />
  );
}

type RateTableEditPanelProps = Pick<
  DataTablePanelProps<RateTableRow>,
  "open" | "onOpenChange" | "row"
> & {
  form: ReturnType<typeof useForm<RateTable>>;
};

function RateTableEditPanel({ open, onOpenChange, row, form }: RateTableEditPanelProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();

  type EditSubmitPayload = {
    action: EditPanelSaveAction;
    values: RateTable;
  };

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { data: record, isLoading: isRecordLoading } = useQuery({
    queryKey: ["rate-table", row?.id],
    queryFn: () => apiService.rateTableService.getById(row?.id ?? ""),
    enabled: open && !!row?.id,
  });

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  useEffect(() => {
    if (open && record) {
      reset(record, { keepDefaultValues: true });
    }
  }, [open, record, reset]);

  const { mutateAsync } = useApiMutation<RateTable, EditSubmitPayload, unknown, RateTable>({
    mutationFn: async ({ values }) => {
      const response = await api.put<RateTable>(`/rate-tables/${row?.id}/`, values);
      return response;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["rate-table-list"],
      });
      const previousRecord = queryClient.getQueryData(["rate-table-list"]);
      queryClient.setQueryData(["rate-table-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: "Rate Table updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: ["rate-table-list"] });
      void queryClient.invalidateQueries({ queryKey: ["rate-table", row?.id] });

      const action = variables.action;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: "Rate Table",
  });

  const onSubmit = useCallback(
    async (values: RateTable, action: EditPanelSaveAction) => {
      await mutateAsync({ values, action });
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: RateTable) => {
    return onSubmit(values, defaultAction);
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : undefined;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.name ?? "Rate Table"}
      description={panelDescription}
      size="xl"
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
      {!row || isRecordLoading ? (
        <ComponentLoader message="Loading Rate Table..." />
      ) : (
        <FormProvider {...form}>
          <Form id="panel-edit-form" onSubmit={handleSubmit(handleFormSubmit)}>
            <RateTableForm />
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
