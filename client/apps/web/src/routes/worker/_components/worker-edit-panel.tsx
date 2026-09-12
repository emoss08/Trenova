import { useT } from "@trenova/shared/i18n/use-t";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { OverflowTabsList } from "@trenova/shared/components/ui/overflow-tabs-list";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
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
import type { WorkerRow } from "@/lib/graphql/worker-table";
import type { Worker } from "@trenova/shared/types/worker";
import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import {
  BriefcaseIcon,
  CalendarClockIcon,
  CalendarRangeIcon,
  Clock4Icon,
  FileTextIcon,
  SmartphoneIcon,
  WalletIcon,
  ShieldCheckIcon,
  UserIcon,
  XIcon,
  IdCardIcon,
  HistoryIcon,
  ClipboardListIcon,
  GraduationCapIcon,
  FlaskConicalIcon,
  FolderCheckIcon,
  ShieldAlertIcon,
  ClipboardCheckIcon,
  GaugeIcon,
  HeartPulseIcon,
} from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { Suspense, lazy, useCallback, useEffect, useRef, useState } from "react";
import { FormProvider, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { ComplianceTab, EmploymentTab, GeneralTab } from "./worker-form-tabs";
import { resolveWorkerPanelTab, type EmploymentView } from "./worker-panel-tabs";

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
const WorkerPTOTab = lazy(() => import("./worker-pto-tab"));
const WorkerScheduleTab = lazy(() => import("./worker-schedule-tab"));
const WorkerCredentialsTab = lazy(() => import("./worker-credentials-tab"));
const WorkerTimelineTab = lazy(() => import("./worker-timeline-tab"));
const WorkerChecklistTab = lazy(() => import("./worker-checklist-tab"));
const WorkerTrainingTab = lazy(() => import("./worker-training-tab"));
const WorkerSafetyTab = lazy(() => import("./worker-safety-tab"));
const WorkerTestingTab = lazy(() => import("./testing/worker-testing-tab"));
const WorkerDQFTab = lazy(() => import("./dqf/worker-dqf-tab"));
const WorkerLeaveTab = lazy(() => import("./leave/worker-leave-tab"));
const WorkerOverviewTab = lazy(() => import("./worker-overview-tab"));
const WorkerReviewsTab = lazy(() => import("./worker-reviews-tab"));

const EMPLOYMENT_VIEWS = [
  { value: "details", label: "Details" },
  { value: "history", label: "History", icon: HistoryIcon },
] satisfies { value: EmploymentView; label: string; icon?: typeof HistoryIcon }[];

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

interface WorkerEditPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: WorkerRow | null;
  form: UseFormReturn<Worker>;
}

export function WorkerEditPanel({ open, onOpenChange, row, form }: WorkerEditPanelProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const pendingActionRef = useRef<EditPanelSaveAction>(defaultAction);

  // The panel opens on the overview because most visits are to read a worker,
  // not to edit one. Deep links that name a tab are unaffected.
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("overview"));

  // The employment history used to be its own tab. Links and concerns still
  // name "timeline", so that value resolves to the Employment tab with the
  // history view showing rather than to nothing.
  const [employmentViewChoice, setEmploymentViewChoice] = useState<EmploymentView>("details");
  const { tab: resolvedTab, employmentView } = resolveWorkerPanelTab(
    activeTab,
    employmentViewChoice,
  );
  const showEmploymentView = (view: EmploymentView) => {
    setEmploymentViewChoice(view);
    if (activeTab === "timeline") void setActiveTab("employment");
  };

  const {
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
      toast.success(t("Changes have been saved"), {
        description: t("Worker updated successfully"),
      });
      void queryClient.invalidateQueries({ queryKey: ["worker-list"] });

      const action = pendingActionRef.current;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
        void setActiveTab("general");
      }
    },
    form,
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
            "border-border bg-background fixed top-4 right-4 bottom-4 z-50 flex flex-col rounded-lg border shadow-lg outline-none",
            "data-[open]:animate-in data-[open]:slide-in-from-right",
            "data-[closed]:animate-out data-[closed]:slide-out-to-right",
            "duration-200",
          )}
          style={{ width: 650 }}
        >
          <div className="border-border flex items-center justify-between border-b px-4 py-3">
            <div className="flex flex-col gap-0.5">
              <Dialog.Title className="text-sm leading-none font-medium">{panelTitle}</Dialog.Title>
              {panelDescription && (
                <Dialog.Description className="text-muted-foreground text-xs">
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
              <span className="sr-only">{t("Close panel")}</span>
            </Dialog.Close>
          </div>

          {!row ? (
            <div className="flex-1 p-4">
              <ComponentLoader message={t("Loading Worker...")} />
            </div>
          ) : (
            <FormProvider {...form}>
              <Form
                id="worker-edit-form"
                onSubmit={() => void handleSubmit(handleFormSubmit)()}
                className="flex flex-1 flex-col overflow-hidden"
              >
                <Tabs
                  value={resolvedTab}
                  onValueChange={(value) => void setActiveTab(value as string)}
                  className="flex flex-1 flex-col overflow-hidden"
                >
                  <div className="border-border border-b px-4">
                    <OverflowTabsList
                      items={[
                        {
                          value: "overview",
                          label: t("Overview"),
                          icon: GaugeIcon,
                        },
                        {
                          value: "general",
                          label: t("General Information"),
                          icon: UserIcon,
                          className: cn(hasGeneralErrors && "text-destructive"),
                        },
                        {
                          value: "employment",
                          label: t("Employment Information"),
                          icon: BriefcaseIcon,
                          className: cn(hasEmploymentErrors && "text-destructive"),
                        },
                        {
                          value: "compliance",
                          label: t("Compliance Status"),
                          icon: ShieldCheckIcon,
                          className: cn(hasComplianceErrors && "text-destructive"),
                        },
                        { value: "credentials", label: t("Credentials"), icon: IdCardIcon },
                        { value: "checklist", label: t("Checklist"), icon: ClipboardListIcon },
                        { value: "training", label: t("Training"), icon: GraduationCapIcon },
                        { value: "safety", label: t("Safety"), icon: ShieldAlertIcon },
                        { value: "testing", label: t("Testing"), icon: FlaskConicalIcon },
                        { value: "dqf", label: t("DQ File"), icon: FolderCheckIcon },
                        { value: "reviews", label: t("Reviews"), icon: ClipboardCheckIcon },
                        { value: "hos", label: t("HOS"), icon: Clock4Icon },
                        { value: "pay", label: t("Pay"), icon: WalletIcon },
                        { value: "pto", label: t("Time Off"), icon: CalendarRangeIcon },
                        { value: "schedule", label: t("Schedule"), icon: CalendarClockIcon },
                        { value: "leave", label: t("Leave"), icon: HeartPulseIcon },
                        { value: "documents", label: t("Documents"), icon: FileTextIcon },
                        { value: "portal", label: t("Portal"), icon: SmartphoneIcon },
                      ]}
                      activeValue={resolvedTab}
                      onSelect={(value) => void setActiveTab(value)}
                    />
                  </div>
                  <ScrollArea className="flex-1">
                    <TabsContent value="overview" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerOverviewTab
                          workerId={row?.id as string}
                          onOpenTab={(tab) => void setActiveTab(tab)}
                        />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="general" className="p-4">
                      <GeneralTab />
                    </TabsContent>
                    <TabsContent value="employment" className="p-4">
                      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                        <div>
                          <h3 className="text-sm font-semibold">{t("Employment")}</h3>
                          <p className="text-muted-foreground text-xs">
                            {employmentView === "details"
                              ? t("Dates, licence and medical details on the record.")
                              : t(
                                  "Every hire, transfer, leave and termination, with who recorded it.",
                                )}
                          </p>
                        </div>
                        <SegmentedControl<EmploymentView>
                          items={EMPLOYMENT_VIEWS}
                          value={employmentView}
                          onValueChange={showEmploymentView}
                          aria-label={t("Employment view")}
                        />
                      </div>
                      {employmentView === "details" ? (
                        <EmploymentTab />
                      ) : (
                        <Suspense
                          fallback={
                            <div className="flex items-center justify-center py-12">
                              <ComponentLoader message={t("Loading...")} />
                            </div>
                          }
                        >
                          <WorkerTimelineTab
                            workerId={row?.id as string}
                            worker={{
                              fleetCodeId: row?.fleetCodeId ?? null,
                              driverType: row?.driverType ?? "",
                              type: row?.type ?? "",
                              status: row?.status ?? "",
                            }}
                          />
                        </Suspense>
                      )}
                    </TabsContent>
                    <TabsContent value="compliance" className="p-4">
                      <ComplianceTab />
                    </TabsContent>
                    <TabsContent value="credentials" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerCredentialsTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="checklist" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerChecklistTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="training" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerTrainingTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="safety" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerSafetyTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="testing" className="p-0">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerTestingTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="dqf" className="p-0">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerDQFTab
                          workerId={row?.id as string}
                          onOpenTab={(tab) => void setActiveTab(tab)}
                        />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="reviews" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerReviewsTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="hos" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
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
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerPayTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="pto" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerPTOTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="schedule" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerScheduleTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="leave" className="p-0">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
                          </div>
                        }
                      >
                        <WorkerLeaveTab workerId={row?.id as string} />
                      </Suspense>
                    </TabsContent>
                    <TabsContent value="documents" className="p-4">
                      <Suspense
                        fallback={
                          <div className="flex items-center justify-center py-12">
                            <ComponentLoader message={t("Loading...")} />
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
                            <ComponentLoader message={t("Loading...")} />
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

          <div className="border-border bg-muted/30 flex items-center justify-end gap-2 border-t px-4 py-3">
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Cancel")}
            </Button>
            <SplitButton
              options={SAVE_OPTIONS}
              selectedOption={defaultAction}
              onOptionSelect={handleOptionSelect}
              isLoading={isSubmitting}
              loadingText={t("Saving...")}
              formId="worker-edit-form"
            />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
