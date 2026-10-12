import { useT } from "@trenova/shared/i18n/use-t";
import { Form } from "@trenova/shared/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { rememberPristineDefaults } from "@/lib/form-defaults";
import {
  useCreatePanelActionPreference,
  type CreatePanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@trenova/shared/lib/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { API_ENDPOINTS } from "@trenova/shared/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { DataTablePanelContainer, type PanelSize } from "./data-table/data-table-panel";
import {
  FormPanelFooter,
  PanelDirtyTracker,
  usePanelCloseGuard,
  type ChangeFields,
} from "./form-changes";
import { createFieldRegistry, FormFieldsProvider } from "@trenova/shared/lib/form-field-registry";
import { FormSaveDock } from "./form-save-dock";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";

type FormCreatePanelProps<
  TFieldValues extends FieldValues,
  TData,
  TSubmitValues = TFieldValues,
  TMutationData = TSubmitValues,
> = Pick<DataTablePanelProps<TData>, "open" | "onOpenChange"> & {
  url?: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  description?: string;
  form: UseFormReturn<TFieldValues, any, TSubmitValues>;
  size?: PanelSize;
  notice?: React.ReactNode;
  useDock?: boolean;
  /**
   * Labels for the change review. Given, the footer counts unsaved changes, lists
   * each one with its own undo, and says why the form cannot be saved.
   */
  changeFields?: ChangeFields;
  /** Why the form cannot be saved yet, shown in the footer beside the changes. */
  footerProblem?: string | null;
  /** More at the left of the footer, such as a test button. */
  footerLeading?: React.ReactNode;
  mutationFn?: (values: TSubmitValues) => Promise<TMutationData>;
};

const SAVE_OPTIONS: SplitButtonOption<CreatePanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & close" },
  { id: "save-add-another", label: "Save & add another" },
];

export function FormCreatePanel<
  TFieldValues extends FieldValues,
  TData,
  TSubmitValues = TFieldValues,
  TMutationData = TSubmitValues,
>({
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
  useDock = false,
  mutationFn,
  changeFields,
  footerProblem,
  footerLeading,
}: FormCreatePanelProps<TFieldValues, TData, TSubmitValues, TMutationData>) {
  const t = useT();

  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useCreatePanelActionPreference();
  const { isPopout, closePopout } = usePopoutWindow();

  type CreateSubmitPayload = {
    action: CreatePanelSaveAction;
    values: TSubmitValues;
  };

  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  // Reset to the route's own defaults rather than whatever the form currently calls its
  // defaults. The edit panel shares this form and replaces defaultValues with the record it
  // loads, so a bare reset() here would open the create panel pre-filled with the last
  // record edited.
  const pristineDefaults = rememberPristineDefaults<TFieldValues>(form);

  useEffect(() => {
    if (open) {
      reset(pristineDefaults);
    }
  }, [open, reset, pristineDefaults]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset(pristineDefaults);
  }, [onOpenChange, reset, pristineDefaults]);
  const guard = usePanelCloseGuard({ open, onClose: handleClose });
  const [fieldRegistry] = useState(createFieldRegistry);
  const dockClosePrompt = guard.confirmingClose
    ? { count: guard.changedCount, onKeepEditing: guard.keepEditing, onDiscard: guard.discard }
    : null;

  const { mutateAsync } = useApiMutation<TMutationData, CreateSubmitPayload, unknown, TFieldValues>(
    {
      mutationFn: async ({ values }) => {
        if (mutationFn) {
          return mutationFn(values);
        }

        if (!url) {
          throw new Error(`No URL configured for ${title}`);
        }

        return api.post<TMutationData>(url, values);
      },
      onSuccess: (_data, variables) => {
        toast.success(t("Changes have been saved."), {
          description: t("{0} created successfully", title),
        });
        void queryClient.invalidateQueries({ queryKey: [queryKey] });

        if (isPopout) {
          closePopout();
          return;
        }

        const action = variables.action;
        if (action === "save") {
          reset(form.getValues());
        }
        if (action === "save-close") {
          onOpenChange(false);
          reset(pristineDefaults);
        } else if (action === "save-add-another") {
          reset(pristineDefaults);
        }
      },
      form,
      resourceName: title,
    },
  );

  const onSubmit = async (values: TSubmitValues, action: CreatePanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: CreatePanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: TSubmitValues) => {
    return onSubmit(values, defaultAction);
  };

  const handlePanelOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      reset(pristineDefaults);
      onOpenChange(true);
      return;
    }
    guard.requestClose();
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const saveKey = event.key === "Enter" || event.key.toLowerCase() === "s";
      if (open && (event.ctrlKey || event.metaKey) && saveKey && !isSubmitting) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  const splitButtonConfig = {
    options: SAVE_OPTIONS,
    selectedOption: defaultAction,
    onOptionSelect: handleOptionSelect,
    loadingText: t("Saving..."),
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handlePanelOpenChange}
      title={t("New {0}", toSentenceFragment(title))}
      description={
        description ?? t("Fill out the form below to create a new {0}.", toSentenceFragment(title))
      }
      size={size}
      footer={
        useDock ? undefined : (
          <FormPanelFooter
            form={form}
            create
            guard={guard}
            registry={fieldRegistry}
            changeFields={changeFields}
            problem={footerProblem}
            leading={footerLeading}
            save={
              <SplitButton
                options={SAVE_OPTIONS}
                selectedOption={defaultAction}
                onOptionSelect={handleOptionSelect}
                isLoading={isSubmitting}
                loadingText={t("Saving...")}
                formId="panel-create-form"
                shortcut={formatShortcut("S")}
                tone="primary"
              />
            }
          />
        )
      }
    >
      <div className="flex flex-col gap-4">
        {notice}
        <FormFieldsProvider registry={fieldRegistry}>
          <FormProvider {...form}>
            <PanelDirtyTracker form={form} guard={guard} />
            <Form id="panel-create-form" onSubmit={handleSubmit(handleFormSubmit)}>
              {formComponent}
              {useDock && (
                <FormSaveDock
                  closePrompt={dockClosePrompt}
                  changeFields={changeFields}
                  splitButton={splitButtonConfig}
                  formId="panel-create-form"
                  position="right"
                  showReset={false}
                />
              )}
            </Form>
          </FormProvider>
        </FormFieldsProvider>
      </div>
    </DataTablePanelContainer>
  );
}
