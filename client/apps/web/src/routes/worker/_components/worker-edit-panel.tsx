import { ComponentLoader } from "@trenova/shared/components/component-loader";
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
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { checkSectionErrors } from "@/lib/form";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { TimeFormat } from "@trenova/shared/types/user";
import type { Worker } from "@trenova/shared/types/worker";
import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import {
  BriefcaseIcon,
  Clock4Icon,
  FileTextIcon,
  SmartphoneIcon,
  WalletIcon,
  ShieldCheckIcon,
  UserIcon,
  XIcon,
} from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { Suspense, lazy, useCallback, useEffect, useRef } from "react";
import { FormProvider, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { ComplianceTab, EmploymentTab, GeneralTab } from "./worker-form-tabs";

const GENERAL_FIELDS = [
  "status",
  "type",
  "firstName",
  "lastName",
  "gender",
  "driverType",
  "fleetCodeId",
  "addressLine1",
  "addressLine2",
  "city",
  "stateId",
  "postalCode",
  "email",
  "phoneNumber",
  "emergencyContactName",
  "emergencyContactPhone",
] as const;

const EMPLOYMENT_FIELDS = [
  "profile.dob",
  "profile.hireDate",
  "profile.terminationDate",
  "profile.licenseNumber",
  "profile.licenseStateId",
  "profile.licenseExpiry",
  "profile.cdlClass",
  "profile.cdlRestrictions",
  "profile.endorsement",
  "profile.hazmatExpiry",
  "profile.medicalCardExpiry",
  "profile.physicalDueDate",
  "profile.medicalExaminerName",
  "profile.medicalExaminerNpi",
] as const;

const COMPLIANCE_FIELDS = [
  "profile.complianceStatus",
  "profile.mvrDueDate",
  "profile.isQualified",
  "profile.disqualificationReason",
  "profile.twicCardNumber",
  "profile.twicExpiry",
  "profile.eldExempt",
  "profile.shortHaulExempt",
  "availableForDispatch",
  "canBeAssigned",
] as const;

const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));
const WorkerPayTab = lazy(() => import("./worker-pay-tab"));
const WorkerPortalTab = lazy(() => import("./worker-portal-tab"));
const WorkerHosTab = lazy(() => import("./worker-hos-tab"));

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

interface WorkerEditPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: Worker | null;
  form: UseFormReturn<Worker>;
}

export function WorkerEditPanel({ open, onOpenChange, row, form }: WorkerEditPanelProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const pendingActionRef = useRef<EditPanelSaveAction>(defaultAction);

  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("general"));

  const {
    setError,
    formState: { isSubmitting, errors },
    handleSubmit,
    reset,
  } = form;

  const hasGeneralErrors = checkSectionErrors(errors, [...GENERAL_FIELDS]) || !!errors.customFields;
  const hasEmploymentErrors = checkSectionErrors(errors, [...EMPLOYMENT_FIELDS]);
  const hasComplianceErrors = checkSectionErrors(errors, [...COMPLIANCE_FIELDS]);

  const handleClose = () => {
    onOpenChange(false);
    reset();
    void setActiveTab("general");
  };

  useEffect(() => {
    if (open && row) {
      reset(row as Worker, { keepDefaultValues: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  useEffect(() => {
    if (!open) {
      void setActiveTab("general");
    }
  }, [open, setActiveTab]);

  const { mutateAsync } = useApiMutation<Worker, Worker, unknown, Worker>({
    mutationFn: async (values: Worker) => {
      return await apiService.workerService.update(row?.id as string, values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: ["worker-list"] });
      const previousRecord = queryClient.getQueryData(["worker-list"]);
      queryClient.setQueryData(["worker-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: "Worker updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: ["worker-list"] });

      const action = pendingActionRef.current;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
        void setActiveTab("general");
      }
    },
    setFormError: setError,
    resourceName: "Worker",
  });

  const onSubmit = useCallback(
    async (values: Worker) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    pendingActionRef.current = action;
    setDefaultAction(action);
    void handleSubmit(onSubmit)();
  };

  const handleFormSubmit = (values: Worker) => {
    pendingActionRef.current = defaultAction;
    return onSubmit(values);
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

  const panelTitle = row?.wholeName || `${row?.firstName} ${row?.lastName}` || "Worker";
  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt as number, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : undefined;

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Popup
          className={cn(
            "fixed top-4 right-4 bottom-4 z-50 flex flex-col rounded-lg border border-border bg-background shadow-lg outline-none",
            "data-[open]:animate-in data-[open]:slide-in-from-right",
            "data-[closed]:animate-out data-[closed]:slide-out-to-right",
            "duration-200",
          )}
          style={{ width: 650 }}
        >
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <div className="flex flex-col gap-0.5">
              <Dialog.Title className="text-sm leading-none font-medium">{panelTitle}</Dialog.Title>
              {panelDescription && (
                <Dialog.Description className="text-xs text-muted-foreground">
                  {panelDescription}
                </Dialog.Description>
              )}
            </div>
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

          {!row ? (
            <div className="flex-1 p-4">
              <ComponentLoader message="Loading Worker..." />
            </div>
          ) : (
            <FormProvider {...form}>
              <Form
                id="worker-edit-form"
                onSubmit={() => void handleSubmit(handleFormSubmit)()}
                className="flex flex-1 flex-col overflow-hidden"
              >
                <Tabs
                  value={activeTab}
                  onValueChange={(value) => void setActiveTab(value as string)}
                  className="flex flex-1 flex-col overflow-hidden"
                >
                  <div className="border-b border-border px-4">
                    <OverflowTabsList
                      items={[
                        {
                          value: "general",
                          label: "General Information",
                          icon: UserIcon,
                          className: cn(hasGeneralErrors && "text-destructive"),
                        },
                        {
                          value: "employment",
                          label: "Employment Information",
                          icon: BriefcaseIcon,
                          className: cn(hasEmploymentErrors && "text-destructive"),
                        },
                        {
                          value: "compliance",
                          label: "Compliance Status",
                          icon: ShieldCheckIcon,
                          className: cn(hasComplianceErrors && "text-destructive"),
                        },
                        { value: "hos", label: "HOS", icon: Clock4Icon },
                        { value: "pay", label: "Pay", icon: WalletIcon },
                        { value: "documents", label: "Documents", icon: FileTextIcon },
                        { value: "portal", label: "Portal", icon: SmartphoneIcon },
                      ]}
                      activeValue={activeTab}
                      onSelect={(value) => void setActiveTab(value)}
                    />
                  </div>
                  <ScrollArea className="flex-1">
                    <TabsContent value="general" className="p-4">
                      <GeneralTab />
                    </TabsContent>
                    <TabsContent value="employment" className="p-4">
                      <EmploymentTab />
                    </TabsContent>
                    <TabsContent value="compliance" className="p-4">
                      <ComplianceTab />
                    </TabsContent>
                    <TabsContent value="hos" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message="Loading..." />
                          </div>
                        }
                      >
                        <WorkerHosTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="pay" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message="Loading..." />
                          </div>
                        }
                      >
                        <WorkerPayTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="documents" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message="Loading..." />
                          </div>
                        }
                      >
                        <DocumentsTab resourceType="worker" resourceId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="portal" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message="Loading..." />
                          </div>
                        }
                      >
                        <WorkerPortalTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                  </ScrollArea>
                </Tabs>
              </Form>
            </FormProvider>
          )}

          <div className="flex items-center justify-end gap-2 border-t border-border bg-muted/30 px-4 py-3">
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <SplitButton
              options={SAVE_OPTIONS}
              selectedOption={defaultAction}
              onOptionSelect={handleOptionSelect}
              isLoading={isSubmitting}
              loadingText="Saving..."
              formId="worker-edit-form"
            />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
