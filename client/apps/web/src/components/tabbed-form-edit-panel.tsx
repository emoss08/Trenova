import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { OverflowTabsList } from "@trenova/shared/components/ui/overflow-tabs-list";
import { Tabs, TabsContent } from "@trenova/shared/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@trenova/shared/lib/api";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { API_ENDPOINTS } from "@trenova/shared/types/server";
import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { parseAsString, useQueryState } from "nuqs";
import {
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
  type LazyExoticComponent,
} from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { FormCopyButton } from "./form-copy-button";
import {
  FormPanelFooter,
  PanelDirtyTracker,
  usePanelCloseGuard,
  type ChangeFields,
} from "./form-changes";
import { createFieldRegistry, FormFieldsProvider } from "@trenova/shared/lib/form-field-registry";
import { FormSaveDock } from "./form-save-dock";

const PANEL_SIZES = {
  sm: 400,
  md: 500,
  lg: 650,
  xl: 800,
} as const;

type PanelSize = keyof typeof PANEL_SIZES;

interface TabConfig {
  value: string;
  label: string;
  icon?: React.ComponentType<{ className?: string }>;
  manageScroll?: boolean;
  hideFooter?: boolean;
  content: LazyExoticComponent<React.ComponentType<any>>;
  contentProps?: Record<string, unknown>;
}

export interface FormTabConfig {
  value: string;
  label: string;
  icon?: React.ComponentType<{ className?: string }>;
  content: React.ReactNode;
}

type TabbedFormEditPanelProps<T extends FieldValues, TData extends Record<string, unknown>> = Pick<
  DataTablePanelProps<TData>,
  "open" | "onOpenChange" | "row"
> & {
  url?: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent?: React.ReactNode;
  form: UseFormReturn<T>;
  fieldKey?: keyof TData;
  titleComponent?: (currentRecord: TData) => React.ReactNode;
  headerActions?: React.ReactNode;
  descriptionExtra?: React.ReactNode;
  tabs?: TabConfig[];
  formTabs?: FormTabConfig[];
  size?: PanelSize;
  useDock?: boolean;
  /**
   * Whether the form still holds only the bare table row.
   *
   * A panel whose record has children fetches them separately, and until they
   * land the form has none. Saving in that window tells the server the record
   * has no children, which it reads as the user having deleted them — so the
   * panel refuses to submit rather than writing that emptiness back.
   */
  isRecordLoading?: boolean;
  recordFailed?: boolean;
  mutationFn?: (values: T, row: TData) => Promise<T>;
  /** Labels for the change review; see FormEditPanel. */
  changeFields?: ChangeFields;
  /** Why the form cannot be saved yet, shown in the footer beside the changes. */
  footerProblem?: string | null;
  /** More at the left of the footer, such as a test button. */
  footerLeading?: React.ReactNode;
};

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & close" },
];

function TabFallback() {
  const t = useT();

  return (
    <div className="flex items-center justify-center py-12">
      <ComponentLoader message={t("Loading...")} />
    </div>
  );
}

export function TabbedFormEditPanel<T extends FieldValues, TData extends Record<string, unknown>>({
  open,
  onOpenChange,
  row,
  url,
  title,
  queryKey,
  formComponent,
  form,
  fieldKey,
  titleComponent,
  headerActions,
  descriptionExtra,
  tabs = [],
  formTabs = [],
  size = "md",
  useDock = false,
  isRecordLoading = false,
  recordFailed = false,
  mutationFn,
  changeFields,
  footerProblem,
  footerLeading,
}: TabbedFormEditPanelProps<T, TData>) {
  const t = useT();

  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const pendingActionRef = useRef<EditPanelSaveAction>(defaultAction);
  const hasFormTabs = formTabs.length > 0;
  const defaultTab = hasFormTabs ? formTabs[0].value : "details";
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault(defaultTab));
  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
    void setActiveTab(defaultTab);
  }, [defaultTab, onOpenChange, reset, setActiveTab]);
  const guard = usePanelCloseGuard({ open, onClose: handleClose });
  const [fieldRegistry] = useState(createFieldRegistry);
  const dockClosePrompt = guard.confirmingClose
    ? { count: guard.changedCount, onKeepEditing: guard.keepEditing, onDiscard: guard.discard }
    : null;
  const handleDialogOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      onOpenChange(true);
      return;
    }
    guard.requestClose();
  };

  useEffect(() => {
    if (open && row) {
      reset(row as unknown as T);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, row?.version, reset]);

  useEffect(() => {
    if (!open) {
      void setActiveTab(defaultTab);
    }
  }, [open, defaultTab, setActiveTab]);

  const { mutateAsync } = useApiMutation<T, T, unknown, T>({
    mutationFn: async (values: T) => {
      if (mutationFn && row) {
        return mutationFn(values, row);
      }

      return api.put<T>(`${url}${row?.id as string}/`, values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: [queryKey] });
      const previousRecord = queryClient.getQueryData([queryKey]);
      queryClient.setQueryData([queryKey], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: () => {
      toast.success(t("Changes have been saved"), {
        description: t("{0} updated successfully", title),
      });
      void queryClient.invalidateQueries({ queryKey: [queryKey] });

      const action = pendingActionRef.current;
      if (action === "save") {
        reset(form.getValues());
      }
      if (action === "save-close") {
        reset();
        onOpenChange(false);
        void setActiveTab(defaultTab);
      }
    },
    form,
    resourceName: title,
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  // The record has to be in the form before any of it can be written back.
  const saveBlocked = isRecordLoading || recordFailed;

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    if (saveBlocked) return;

    pendingActionRef.current = action;
    setDefaultAction(action);
    void handleSubmit(onSubmit)();
  };

  const handleFormSubmit = (values: T) => {
    if (saveBlocked) return;

    pendingActionRef.current = defaultAction;
    return onSubmit(values);
  };

  const splitButtonConfig = {
    options: SAVE_OPTIONS,
    selectedOption: defaultAction,
    onOptionSelect: handleOptionSelect,
    disabled: saveBlocked,
    loadingText: t("Saving..."),
  };

  const hasTabs = tabs.length > 0;
  const activeTabConfig = tabs.find((tab) => tab.value === activeTab);
  const activeTabManagesScroll = activeTabConfig?.manageScroll ?? false;
  const activeTabHidesFooter = activeTabConfig?.hideFooter ?? false;

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
        !activeTabHidesFooter &&
        !saveBlocked &&
        (event.ctrlKey || event.metaKey) &&
        (event.key === "Enter" || event.key.toLowerCase() === "s") &&
        !isSubmitting
      ) {
        event.preventDefault();
        pendingActionRef.current = defaultAction;
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [
    open,
    isSubmitting,
    handleSubmit,
    defaultAction,
    onSubmit,
    activeTabHidesFooter,
    saveBlocked,
  ]);

  const panelTitle = titleComponent
    ? row
      ? titleComponent(row)
      : title
    : fieldKey && row
      ? String(row[fieldKey])
      : title;

  const panelDescription = row?.updatedAt
    ? t(
        "Last updated on {0}",
        formatToUserTimezone(
          row.updatedAt as number,
          {
            timeFormat: user?.timeFormat || "24-hour",
          },
          user?.timezone,
        ),
      )
    : undefined;

  const rowId = row !== null && row !== undefined ? String(row.id) : "unable to retrieve ID";

  return (
    <Dialog.Root open={open} onOpenChange={handleDialogOpenChange}>
      <Dialog.Portal>
        <Dialog.Popup
          className={cn(
            "border-border bg-background fixed top-4 right-4 bottom-4 z-50 flex flex-col rounded-lg border outline-none",
            "data-open:animate-in data-open:slide-in-from-right",
            "data-closed:animate-out data-closed:slide-out-to-right",
            "duration-200",
          )}
          style={{ width: PANEL_SIZES[size] }}
        >
          <div className="border-border flex flex-col border-b px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex flex-row gap-1">
                <Dialog.Title className="text-2xl leading-none font-semibold">
                  {typeof panelTitle === "string" ? panelTitle : title}
                </Dialog.Title>
                <FormCopyButton rowId={rowId} />
              </div>
              <div className="flex items-center gap-1">
                {headerActions}
                <Dialog.Close
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-muted-foreground hover:text-foreground"
                    />
                  }
                >
                  <XCloseIcon className="size-4" />
                  <span className="sr-only">{t("Close panel")}</span>
                </Dialog.Close>
              </div>
            </div>
            {(panelDescription || descriptionExtra) && (
              <div className="mt-0.5 flex items-center justify-between">
                {panelDescription && (
                  <Dialog.Description className="text-muted-foreground text-xs">
                    {panelDescription}
                  </Dialog.Description>
                )}
                {descriptionExtra}
              </div>
            )}
          </div>

          {!row || isRecordLoading ? (
            <div className="flex-1 p-4">
              <ComponentLoader message={t("Loading {0}...", title)} />
            </div>
          ) : recordFailed ? (
            <div className="flex-1 p-4">
              <p className="text-destructive text-sm">
                {t(
                  "This {0} could not be loaded, so it cannot be edited safely. Close the panel and try again.",
                  title.toLowerCase(),
                )}
              </p>
            </div>
          ) : hasFormTabs ? (
            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as string)}
              className="flex flex-1 flex-col gap-0 overflow-hidden"
            >
              <div className="border-border border-b px-4">
                <OverflowTabsList
                  items={formTabs.map((tab) => ({
                    value: tab.value,
                    label: tab.label,
                    icon: tab.icon,
                  }))}
                  activeValue={activeTab}
                  onSelect={(value) => void setActiveTab(value)}
                />
              </div>

              <ScrollArea className="flex-1">
                <FormFieldsProvider registry={fieldRegistry}>
                  <FormProvider {...form}>
                    <PanelDirtyTracker form={form} guard={guard} />
                    <Form id="panel-edit-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
                      {formTabs.map((tab) => (
                        <TabsContent key={tab.value} value={tab.value} keepMounted className="p-4">
                          {tab.content}
                        </TabsContent>
                      ))}
                      {useDock && (
                        <FormSaveDock
                          closePrompt={dockClosePrompt}
                          changeFields={changeFields}
                          splitButton={splitButtonConfig}
                          formId="panel-edit-form"
                          position="right"
                          showReset={false}
                        />
                      )}
                    </Form>
                  </FormProvider>
                </FormFieldsProvider>
              </ScrollArea>
            </Tabs>
          ) : hasTabs ? (
            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as string)}
              className="flex flex-1 flex-col gap-0 overflow-hidden"
            >
              <div className="border-border border-b px-4">
                <OverflowTabsList
                  items={[
                    { value: "details", label: t("Details") },
                    ...tabs.map((tab) => ({
                      value: tab.value,
                      label: tab.label,
                      icon: tab.icon,
                    })),
                  ]}
                  activeValue={activeTab}
                  onSelect={(value) => void setActiveTab(value)}
                />
              </div>

              <ScrollArea className={cn("flex-1", activeTabManagesScroll && "hidden")}>
                <TabsContent value="details" className="p-4">
                  <FormFieldsProvider registry={fieldRegistry}>
                    <FormProvider {...form}>
                      <PanelDirtyTracker form={form} guard={guard} />
                      <Form id="panel-edit-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
                        {formComponent}
                        {useDock && (
                          <FormSaveDock
                            closePrompt={dockClosePrompt}
                            changeFields={changeFields}
                            splitButton={splitButtonConfig}
                            formId="panel-edit-form"
                            position="right"
                            showReset={false}
                          />
                        )}
                      </Form>
                    </FormProvider>
                  </FormFieldsProvider>
                </TabsContent>

                {tabs
                  .filter((tab) => !tab.manageScroll)
                  .map((tab) => (
                    <TabsContent key={tab.value} value={tab.value} className="p-4">
                      <Suspense fallback={<TabFallback />}>
                        <tab.content {...(tab.contentProps || {})} />
                      </Suspense>
                    </TabsContent>
                  ))}
              </ScrollArea>

              {tabs
                .filter((tab) => tab.manageScroll)
                .map((tab) => (
                  <TabsContent
                    key={tab.value}
                    value={tab.value}
                    className="flex min-h-0 flex-1 flex-col overflow-hidden"
                  >
                    <Suspense fallback={<TabFallback />}>
                      <tab.content {...(tab.contentProps || {})} />
                    </Suspense>
                  </TabsContent>
                ))}
            </Tabs>
          ) : (
            <ScrollArea className="flex-1">
              <div className="p-4">
                <FormFieldsProvider registry={fieldRegistry}>
                  <FormProvider {...form}>
                    <PanelDirtyTracker form={form} guard={guard} />
                    <Form id="panel-edit-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
                      {formComponent}
                      {useDock && (
                        <FormSaveDock
                          closePrompt={dockClosePrompt}
                          changeFields={changeFields}
                          splitButton={splitButtonConfig}
                          formId="panel-edit-form"
                          position="right"
                          showReset={false}
                        />
                      )}
                    </Form>
                  </FormProvider>
                </FormFieldsProvider>
              </div>
            </ScrollArea>
          )}

          <div
            className={cn(
              "border-border bg-muted/30 flex items-center gap-2 border-t px-4 py-3",
              (useDock || (activeTabHidesFooter && !guard.confirmingClose)) && "hidden",
            )}
          >
            <FormPanelFooter
              form={form}
              create={false}
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
                  disabled={saveBlocked}
                  loadingText={t("Saving...")}
                  formId="panel-edit-form"
                  shortcut={formatShortcut("S")}
                  tone="primary"
                />
              }
            />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
