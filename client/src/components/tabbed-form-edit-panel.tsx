import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { FormSaveDock } from "./form-save-dock";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { API_ENDPOINTS } from "@/types/server";
import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { XIcon } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { Suspense, useCallback, useEffect, useRef, type LazyExoticComponent } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { ComponentLoader } from "./component-loader";

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

type TabbedFormEditPanelProps<T extends FieldValues, TData extends Record<string, unknown>> = Pick<
  DataTablePanelProps<TData>,
  "open" | "onOpenChange" | "row"
> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  fieldKey?: keyof TData;
  titleComponent?: (currentRecord: TData) => React.ReactNode;
  headerActions?: React.ReactNode;
  descriptionExtra?: React.ReactNode;
  tabs?: TabConfig[];
  size?: PanelSize;
  useDock?: boolean;
};

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

function TabFallback() {
  return (
    <div className="flex items-center justify-center py-12">
      <ComponentLoader message="Loading..." />
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
  size = "md",
  useDock = false,
}: TabbedFormEditPanelProps<T, TData>) {
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const pendingActionRef = useRef<EditPanelSaveAction>(defaultAction);

  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("details"));

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
    void setActiveTab("details");
  };

  useEffect(() => {
    if (open && row) {
      reset(row as unknown as T);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, row?.version, reset]);

  useEffect(() => {
    if (!open) {
      void setActiveTab("details");
    }
  }, [open, setActiveTab]);

  const { mutateAsync } = useApiMutation<T, T, unknown, T>({
    mutationFn: async (values: T) => {
      return api.put<T>(`${url}${row?.id as string}/`, values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: [queryKey] });
      const previousRecord = queryClient.getQueryData([queryKey]);
      queryClient.setQueryData([queryKey], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: `${title} updated successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [queryKey] });

      const action = pendingActionRef.current;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
        void setActiveTab("details");
      }
    },
    setFormError: setError,
    resourceName: title,
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    pendingActionRef.current = action;
    setDefaultAction(action);
    void handleSubmit(onSubmit)();
  };

  const handleFormSubmit = (values: T) => {
    pendingActionRef.current = defaultAction;
    return onSubmit(values);
  };

  const splitButtonConfig = {
    options: SAVE_OPTIONS,
    selectedOption: defaultAction,
    onOptionSelect: handleOptionSelect,
    loadingText: "Saving...",
  };

  const hasTabs = tabs.length > 0;
  const activeTabConfig = tabs.find((tab) => tab.value === activeTab);
  const activeTabManagesScroll = activeTabConfig?.manageScroll ?? false;
  const activeTabHidesFooter = activeTabConfig?.hideFooter ?? false;

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && !activeTabHidesFooter && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        pendingActionRef.current = defaultAction;
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, defaultAction, onSubmit, activeTabHidesFooter]);

  const panelTitle = titleComponent
    ? row
      ? titleComponent(row)
      : title
    : fieldKey && row
      ? String(row[fieldKey])
      : title;

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt as number, {
        timeFormat: user?.timeFormat || "24-hour",
      }, user?.timezone)}`
    : undefined;

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Popup
          className={cn(
            "fixed top-4 right-4 bottom-4 z-50 flex flex-col rounded-lg border border-border bg-background shadow-lg outline-none",
            "data-open:animate-in data-open:slide-in-from-right",
            "data-closed:animate-out data-closed:slide-out-to-right",
            "duration-200",
          )}
          style={{ width: PANEL_SIZES[size] }}
        >
          <div className="flex flex-col border-b border-border px-4 py-3">
            <div className="flex items-center justify-between">
              <Dialog.Title className="text-2xl leading-none font-semibold">
                {typeof panelTitle === "string" ? panelTitle : title}
              </Dialog.Title>
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
                  <XIcon className="size-4" />
                  <span className="sr-only">Close panel</span>
                </Dialog.Close>
              </div>
            </div>
            {(panelDescription || descriptionExtra) && (
              <div className="mt-0.5 flex items-center justify-between">
                {panelDescription && (
                  <Dialog.Description className="text-xs text-muted-foreground">
                    {panelDescription}
                  </Dialog.Description>
                )}
                {descriptionExtra}
              </div>
            )}
          </div>

          {!row ? (
            <div className="flex-1 p-4">
              <ComponentLoader message={`Loading ${title}...`} />
            </div>
          ) : hasTabs ? (
            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as string)}
              className="flex flex-1 flex-col overflow-hidden"
            >
              <div className="border-b border-border px-4 pt-2">
                <TabsList variant="underline">
                  <TabsTab value="details">Details</TabsTab>
                  {tabs.map((tab) => (
                    <TabsTab key={tab.value} value={tab.value} className="hover:text-foreground">
                      {tab.icon && <tab.icon className="mr-1 size-4" />}
                      {tab.label}
                    </TabsTab>
                  ))}
                </TabsList>
              </div>

              <ScrollArea className={cn("flex-1", activeTabManagesScroll && "hidden")}>
                <TabsContent value="details" className="p-4">
                  <FormProvider {...form}>
                    <Form id="panel-edit-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
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
                <FormProvider {...form}>
                  <Form id="panel-edit-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
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
              </div>
            </ScrollArea>
          )}

          <div
            className={cn(
              "flex items-center justify-between gap-2 border-t border-border bg-muted/30 px-4 py-3",
              (activeTabHidesFooter || useDock) && "hidden",
            )}
          >
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
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
