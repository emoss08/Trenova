import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { OverflowTabsList } from "@trenova/shared/components/ui/overflow-tabs-list";
import { Tabs, TabsContent } from "@trenova/shared/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useCreatePanelActionPreference,
  type CreatePanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@trenova/shared/lib/api";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { API_ENDPOINTS } from "@trenova/shared/types/server";
import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { XIcon } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, type FieldValues, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { FormSaveDock } from "./form-save-dock";
import type { FormTabConfig } from "./tabbed-form-edit-panel";

const PANEL_SIZES = {
  sm: 400,
  md: 500,
  lg: 650,
  xl: 800,
} as const;

type PanelSize = keyof typeof PANEL_SIZES;

type TabbedFormCreatePanelProps<T extends FieldValues, TData> = Pick<
  DataTablePanelProps<TData>,
  "open" | "onOpenChange"
> & {
  url?: API_ENDPOINTS;
  title: string;
  queryKey: string;
  description?: string;
  formComponent?: React.ReactNode;
  form: UseFormReturn<T>;
  headerActions?: React.ReactNode;
  notice?: React.ReactNode;
  formTabs?: FormTabConfig[];
  size?: PanelSize;
  useDock?: boolean;
  mutationFn?: (values: T) => Promise<T>;
};

const SAVE_OPTIONS: SplitButtonOption<CreatePanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
  { id: "save-add-another", label: "Save & Add Another" },
];

export function TabbedFormCreatePanel<T extends FieldValues, TData>({
  open,
  onOpenChange,
  url,
  title,
  queryKey,
  description,
  formComponent,
  form,
  headerActions,
  notice,
  formTabs = [],
  size = "md",
  useDock = false,
  mutationFn,
}: TabbedFormCreatePanelProps<T, TData>) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useCreatePanelActionPreference();
  const pendingActionRef = useRef<CreatePanelSaveAction>(defaultAction);
  const hasFormTabs = formTabs.length > 0;
  const defaultTab = hasFormTabs ? formTabs[0].value : "details";
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault(defaultTab));
  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
    void setActiveTab(defaultTab);
  };

  useEffect(() => {
    if (open) {
      reset();
      void setActiveTab(defaultTab);
    }
  }, [open, defaultTab, reset, setActiveTab]);

  const { mutateAsync } = useApiMutation<T, T, unknown, T>({
    mutationFn: async (values: T) => {
      if (mutationFn) {
        return mutationFn(values);
      }

      if (!url) {
        throw new Error(`No URL configured for ${title}`);
      }

      return api.post<T>(url, values);
    },
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: `${title} created successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [queryKey] });

      const action = pendingActionRef.current;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
        void setActiveTab(defaultTab);
      } else if (action === "save-add-another") {
        reset();
        void setActiveTab(defaultTab);
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

  const handleOptionSelect = (action: CreatePanelSaveAction) => {
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

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        pendingActionRef.current = defaultAction;
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, defaultAction, onSubmit]);

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
                {`Add New ${title}`}
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
            <Dialog.Description className="mt-0.5 text-xs text-muted-foreground">
              {description ?? `Fill out the form below to create a new ${title}.`}
            </Dialog.Description>
          </div>

          {notice}

          {hasFormTabs ? (
            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as string)}
              className="flex flex-1 flex-col overflow-hidden"
            >
              <div className="border-b border-border px-4 pt-2">
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
                <FormProvider {...form}>
                  <Form id="panel-create-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
                    {formTabs.map((tab) => (
                      <TabsContent key={tab.value} value={tab.value} keepMounted className="p-4">
                        {tab.content}
                      </TabsContent>
                    ))}
                    {useDock && (
                      <FormSaveDock
                        splitButton={splitButtonConfig}
                        formId="panel-create-form"
                        position="right"
                        showReset={false}
                      />
                    )}
                  </Form>
                </FormProvider>
              </ScrollArea>
            </Tabs>
          ) : (
            <ScrollArea className="flex-1">
              <div className="p-4">
                <FormProvider {...form}>
                  <Form id="panel-create-form" onSubmit={() => handleSubmit(handleFormSubmit)()}>
                    {formComponent}
                    {useDock && (
                      <FormSaveDock
                        splitButton={splitButtonConfig}
                        formId="panel-create-form"
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
              useDock && "hidden",
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
              formId="panel-create-form"
            />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
