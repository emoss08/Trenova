import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { FormSaveDock } from "./form-save-dock";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { useAuthStore } from "@/stores/auth-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { API_ENDPOINTS } from "@/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { ComponentLoader } from "./component-loader";
import { DataTablePanelContainer, type PanelSize } from "./data-table/data-table-panel";

type FormEditPanelProps<T extends FieldValues, TData extends Record<string, unknown>> = Pick<
  DataTablePanelProps<TData>,
  "open" | "onOpenChange" | "row"
> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  fieldKey?: keyof TData;
  size?: PanelSize;
  titleComponent?: (currentRecord: TData) => React.ReactNode;
  headerActions?: React.ReactNode;
  useDock?: boolean;
};

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

export function FormEditPanel<T extends FieldValues, TData extends Record<string, unknown>>({
  open,
  onOpenChange,
  row,
  url,
  title,
  queryKey,
  formComponent,
  size,
  form,
  fieldKey,
  titleComponent,
  headerActions,
  useDock = false,
}: FormEditPanelProps<T, TData>) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const { isPopout, closePopout } = usePopoutWindow();
  const user = useAuthStore((s) => s.user);

  type EditSubmitPayload = {
    action: EditPanelSaveAction;
    values: T;
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
      reset(row as unknown as T);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  const { mutateAsync } = useApiMutation<T, EditSubmitPayload, unknown, T>({
    mutationFn: async ({ values }) => {
      return api.put<T>(`${url}${row?.id as string}/`, values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: [queryKey] });
      const previousRecord = queryClient.getQueryData([queryKey]);
      queryClient.setQueryData([queryKey], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: `${title} updated successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [queryKey] });

      if (isPopout) {
        closePopout();
        return;
      }

      const action = variables.action;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: title,
  });

  const onSubmit = async (values: T, action: EditPanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: T) => {
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

  const resolvedTitle = fieldKey && row ? String(row[fieldKey]) : title;
  const resolvedTitleComponent = titleComponent && row ? titleComponent(row) : undefined;

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt as number, {
        timeFormat: user?.timeFormat || "24-hour",
      }, user?.timezone)}`
    : undefined;

  const splitButtonConfig = {
    options: SAVE_OPTIONS,
    selectedOption: defaultAction,
    onOptionSelect: handleOptionSelect,
    loadingText: "Saving...",
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={resolvedTitle}
      titleComponent={resolvedTitleComponent}
      description={panelDescription}
      headerActions={headerActions}
      size={size}
      footer={
        useDock ? undefined : (
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
        )
      }
    >
      {!row ? (
        <ComponentLoader message={`Loading ${title}...`} />
      ) : (
        <FormProvider {...form}>
          <Form id="panel-edit-form" onSubmit={handleSubmit(handleFormSubmit)}>
            {formComponent}
            {useDock && (
              <FormSaveDock
                splitButton={splitButtonConfig}
                formId="panel-edit-form"
                position="right"
                showReset={false}
              />
            )}
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
