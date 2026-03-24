import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useCreatePanelActionPreference,
  type CreatePanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { API_ENDPOINTS } from "@/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { DataTablePanelContainer, type PanelSize } from "./data-table/data-table-panel";

type FormCreatePanelProps<T extends FieldValues, TData> = Pick<
  DataTablePanelProps<TData>,
  "open" | "onOpenChange"
> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  description?: string;
  form: UseFormReturn<T>;
  size?: PanelSize;
  notice?: React.ReactNode;
};

const SAVE_OPTIONS: SplitButtonOption<CreatePanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
  { id: "save-add-another", label: "Save & Add Another" },
];

export function FormCreatePanel<T extends FieldValues, TData>({
  open,
  onOpenChange,
  description,
  title,
  formComponent,
  form,
  url,
  size,
  queryKey,
  notice,
}: FormCreatePanelProps<T, TData>) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useCreatePanelActionPreference();
  const { isPopout, closePopout } = usePopoutWindow();

  type CreateSubmitPayload = {
    action: CreatePanelSaveAction;
    values: T;
  };

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open) {
      reset();
    }
  }, [open, reset]);

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  const { mutateAsync } = useApiMutation<T, CreateSubmitPayload, unknown, T>({
    mutationFn: async ({ values }) => {
      return api.post<T>(url, values);
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved.", {
        description: `${title} created successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [queryKey] });

      if (isPopout) {
        closePopout();
        return;
      }

      const action = variables.action;
      if (action === "save-close") {
        onOpenChange(false);
        reset();
      } else if (action === "save-add-another") {
        reset();
      }
    },
    setFormError: setError,
    resourceName: title,
  });

  const onSubmit = async (values: T, action: CreatePanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: CreatePanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: T) => {
    return onSubmit(values, defaultAction);
  };

  const handlePanelOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      reset();
    }
    onOpenChange(nextOpen);
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

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handlePanelOpenChange}
      title={`Add New ${title}`}
      description={description ?? `Fill out the form below to create a new ${title}.`}
      size={size}
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
            formId="panel-create-form"
          />
        </>
      }
    >
      {notice}
      <FormProvider {...form}>
        <Form id="panel-create-form" onSubmit={handleSubmit(handleFormSubmit)}>
          {formComponent}
        </Form>
      </FormProvider>
    </DataTablePanelContainer>
  );
}
