import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { documentTypeSchema, type DocumentType } from "@/types/document-type";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlertIcon } from "lucide-react";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { DocumentTypeForm } from "./document-type-form";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

export function DocumentTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DocumentType>) {
  const form = useForm({
    resolver: zodResolver(documentTypeSchema),
    defaultValues: {
      code: "",
      name: "",
      description: "",
      color: null,
      documentClassification: "Public" as const,
      documentCategory: "Other" as const,
    },
  });

  if (mode === "edit") {
    return (
      <DocumentTypeEditPanel
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
      url="/document-types/"
      queryKey="document-type-list"
      title="Document Type"
      formComponent={<DocumentTypeForm />}
    />
  );
}

type DocumentTypeEditPanelProps = Pick<
  DataTablePanelProps<DocumentType>,
  "open" | "onOpenChange" | "row"
> & {
  form: ReturnType<typeof useForm<DocumentType>>;
};

function DocumentTypeEditPanel({
  open,
  onOpenChange,
  row,
  form,
}: DocumentTypeEditPanelProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();

  type EditSubmitPayload = {
    action: EditPanelSaveAction;
    values: DocumentType;
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
      reset(row as DocumentType, { keepDefaultValues: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  const { mutateAsync } = useApiMutation<
    DocumentType,
    EditSubmitPayload,
    unknown,
    DocumentType
  >({
    mutationFn: async ({ values }) => {
      const response = await api.put<DocumentType>(
        `/document-types/${row?.id}/`,
        values,
      );
      return response;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["document-type-list"],
      });
      const previousRecord = queryClient.getQueryData(["document-type-list"]);
      queryClient.setQueryData(["document-type-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: "Document Type updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: ["document-type-list"] });

      const action = variables.action;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: "Document Type",
  });

  const onSubmit = useCallback(
    async (values: DocumentType, action: EditPanelSaveAction) => {
      await mutateAsync({ values, action });
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: DocumentType) => {
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

  const isSystem = row?.isSystem ?? false;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.name ?? "Document Type"}
      description={panelDescription}
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          {isSystem ? (
            <Button disabled>Save</Button>
          ) : (
            <SplitButton
              options={SAVE_OPTIONS}
              selectedOption={defaultAction}
              onOptionSelect={handleOptionSelect}
              isLoading={isSubmitting}
              loadingText="Saving..."
              formId="panel-edit-form"
            />
          )}
        </>
      }
    >
      {!row ? (
        <ComponentLoader message="Loading Document Type..." />
      ) : (
        <div className="flex flex-col gap-6">
          {isSystem && (
            <Alert variant="info">
              <CircleAlertIcon />
              <AlertTitle>System Document Type</AlertTitle>
              <AlertDescription>
                This is a system document type and cannot be modified.
              </AlertDescription>
            </Alert>
          )}
          <FormProvider {...form}>
            <Form
              id="panel-edit-form"
              onSubmit={handleSubmit(handleFormSubmit)}
            >
              <DocumentTypeForm disabled={isSystem} />
            </Form>
          </FormProvider>
        </div>
      )}
    </DataTablePanelContainer>
  );
}
