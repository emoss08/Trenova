import { useT } from "@trenova/shared/i18n/use-t";
import { CapabilityGate } from "@/components/capability-gate";
import { NumberField } from "@/components/fields/number-field";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { SelectField } from "@/components/fields/select-field";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { ScoringWeightForm } from "./scoring-weight-form";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  autoAssignmentStrategyChoices,
  complianceEnforcementLevelChoices,
  serviceIncidentTypeChoices,
  tenderExhaustedActionChoices,
} from "@/lib/choices";
import { dispatchControl } from "@/lib/queries/dispatch-control";
import { apiService } from "@/services/api";
import {
  dispatchControlSchema,
  type DispatchControl,
  serviceIncidentTypeSchema,
} from "@/types/dispatch-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";

export default function DispatchControlForm() {
  const t = useT();

  const { data } = useSuspenseQuery({
    ...dispatchControl.get(),
  });

  const form = useForm<DispatchControl>({
    resolver: zodResolver(dispatchControlSchema) as Resolver<DispatchControl>,
    defaultValues: data,
  });

  const { handleSubmit, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: dispatchControl.get._def,
    mutationFn: async (values: DispatchControl) => apiService.dispatchControlService.update(values),
    resourceName: "Dispatch Control",
    resetForm: reset,
    form,
    invalidateQueries: [dispatchControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: DispatchControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          {/* Auto-assignment, its scoring weights and every DOT check on this
              page act on employed drivers. An organization that brokers all of
              its freight keeps the page — service failure monitoring is about
              the load, not the driver — with the driver-side cards withheld.
              The values behind them are still submitted untouched, so nothing
              is reset by passing through here. */}
          <CapabilityGate capability={OrganizationCapability.AssetOperations}>
            <AutoAssignmentForm />
            <ScoringWeightForm />
          </CapabilityGate>
          <ServiceFailureForm />
          <CheckCallMonitoringForm />
          <EtaMonitoringForm />
          <ExceptionQueueForm />
          <CapabilityGate capability={OrganizationCapability.Brokerage}>
            <TenderNegotiationForm />
          </CapabilityGate>
          <CapabilityGate capability={OrganizationCapability.AssetOperations}>
            <ComplianceForm />
          </CapabilityGate>
          <FormSaveDock saveButtonContent={t("Save changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function AutoAssignmentForm() {
  const t = useT();

  const { control } = useFormContext<DispatchControl>();

  const enableAutoAssignment = useWatch({
    control,
    name: "enableAutoAssignment",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Automated resource assignment")}</CardTitle>
        <CardDescription>
          {t(
            "Configure how the system chooses workers and equipment for shipments. These controls influence assignment consistency, utilization, and dispatch throughput.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAutoAssignment"
              label={t("Enable automated assignment")}
              description={t(
                "When enabled, the system can automatically assign available resources to shipments.",
              )}
              position="left"
            />
          </FormControl>
          {enableAutoAssignment && (
            <>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <SelectField
                  control={control}
                  name="autoAssignmentStrategy"
                  label={t("Assignment optimization strategy")}
                  description={t(
                    "Select the primary strategy used when matching resources to shipments.",
                  )}
                  options={autoAssignmentStrategyChoices}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="autoAssignConfidenceThreshold"
                  label={t("Auto-execute confidence threshold")}
                  description={t(
                    "Minimum confidence (0 to 1) an automatic assignment must reach before it executes without dispatcher review.",
                  )}
                  placeholder="0.85"
                  decimalScale={2}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="autoAssignMaxDeadheadMiles"
                  label={t("Maximum deadhead miles")}
                  description={t(
                    "Candidates beyond this many empty miles from the pickup are never auto-assigned. Leave empty for no cap.",
                  )}
                  placeholder={t("No limit")}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="autoAssignPlanningHorizonHours"
                  label={t("Planning horizon (hours)")}
                  description={t("How far ahead the optimizer plans moves, from 1 to 336 hours.")}
                  placeholder="48"
                  sideText="hours"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="coverageRiskWindowHours"
                  label={t("Coverage risk window (hours)")}
                  description={t(
                    "A move this close to its start with nobody on it is raised to the watchtower and handed to the dispatch agent. Shorter than the planning horizon on purpose.",
                  )}
                  placeholder="12"
                  sideText="hours"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ServiceFailureForm() {
  const t = useT();

  const { control } = useFormContext<DispatchControl>();

  const recordServiceFailures = useWatch({
    control,
    name: "recordServiceFailures",
  });

  const showFailureFields = recordServiceFailures !== serviceIncidentTypeSchema.enum.Never;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Service failure monitoring")}</CardTitle>
        <CardDescription>
          {t(
            "Define which service failures to track and when they should be recorded. These settings drive operational reporting and exception visibility.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SelectField
              control={control}
              name="recordServiceFailures"
              label={t("Record Service failures")}
              description={t("Choose which incident types should be captured as service failures.")}
              options={serviceIncidentTypeChoices}
            />
          </FormControl>
          {showFailureFields && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="serviceFailureGracePeriod"
                  label={t("Service failure grace period")}
                  placeholder={t("Enter grace period in minutes")}
                  description={t(
                    "Defines the delay buffer before an eligible incident is recorded as a failure.",
                  )}
                  sideText="minutes"
                  min={1}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="serviceFailureTarget"
                  label={t("Service failure target")}
                  placeholder={t("Enter target percentage")}
                  description={t("Optional threshold for acceptable service failure rate.")}
                  sideText="%"
                  min={0}
                />
              </FormControl>
            </div>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function CheckCallMonitoringForm() {
  const t = useT();

  const { control } = useFormContext<DispatchControl>();

  const enableCheckCallMonitoring = useWatch({
    control,
    name: "enableCheckCallMonitoring",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Check call monitoring")}</CardTitle>
        <CardDescription>
          {t(
            "Watch brokered loads in transit for carriers that stop reporting. A load with no check call, tracking update or stop time inside the window is raised to the watchtower.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableCheckCallMonitoring"
              label={t("Monitor brokered loads for check calls")}
              description={t(
                "When enabled, a carrier-covered load that goes quiet raises a No check call item, and one silent for twice the window is critical.",
              )}
              position="left"
            />
          </FormControl>
          {enableCheckCallMonitoring && (
            <FormControl className="min-h-[3em] max-w-[400px] pl-10">
              <NumberField
                control={control}
                name="checkCallIntervalHours"
                label={t("Check call window")}
                description={t(
                  "How long a load may go without word from the carrier, from 1 to 72 hours. A customer can set its own window.",
                )}
                placeholder="4"
                sideText="hours"
                min={1}
                max={72}
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function EtaMonitoringForm() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Arrival projections")}</CardTitle>
        <CardDescription>
          {t(
            "Every open stop on an active shipment carries a projected arrival, worked out from the truck's position, the carrier's check calls or the last departure, with routed drive time and the driver's hours.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableEtaAlerts"
              label={t("Raise at-risk loads in the watchtower")}
              description={t(
                "A load projected to arrive with less slack than the threshold raises an ETA at risk item; one projected past its window is critical.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="etaRiskThresholdMinutes"
              label={t("At-risk threshold")}
              description={t(
                "Slack under this many minutes counts as at risk, from 0 to 1440. A customer can set its own threshold.",
              )}
              placeholder="60"
              sideText="minutes"
              min={0}
              max={1440}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="etaDwellMinutes"
              label={t("Stop dwell time")}
              description={t(
                "How long a stop takes once the truck is there, used to chain later stops, from 0 to 720 minutes.",
              )}
              placeholder="60"
              sideText="minutes"
              min={0}
              max={720}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ExceptionQueueForm() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Exception queue")}</CardTitle>
        <CardDescription>
          {t(
            "Every exception on the watchtower has an owner and a due time. An overdue one escalates to its owner and the escalation contact until it is dealt with or the escalation limit is reached.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionDueCriticalMinutes"
              label={t("Critical exceptions are due in")}
              placeholder="60"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionDueWarningMinutes"
              label={t("Warnings are due in")}
              placeholder="240"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionDueInfoMinutes"
              label={t("Information is due in")}
              description={t(
                "From 5 minutes to a week, counted from when the exception is raised.",
              )}
              placeholder="1440"
              sideText="minutes"
              min={5}
              max={10080}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionEscalationMinutes"
              label={t("Escalate an overdue exception every")}
              description={t(
                "Each escalation raises its severity, tells its owner and the escalation contact, and gives it this long again.",
              )}
              placeholder="60"
              sideText="minutes"
              min={5}
              max={1440}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionMaxEscalations"
              label={t("Escalate at most")}
              description={t("0 turns escalation off; at most 10.")}
              placeholder="3"
              sideText="times"
              min={0}
              max={10}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <UserAutocompleteField
              control={control}
              name="exceptionDefaultAssigneeId"
              label={t("Default owner")}
              description={t(
                "Owns an exception no rule assigns. A shipment's own exceptions go to its owner first.",
              )}
              placeholder={t("Nobody")}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <UserAutocompleteField
              control={control}
              name="exceptionEscalationContactId"
              label={t("Escalation contact")}
              description={t("Hears about every escalation, alongside the owner.")}
              placeholder={t("Nobody")}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionMissedStopGraceMinutes"
              label={t("A stop is missed after its window closes by")}
              description={t(
                "With no arrival recorded by then, a missed pickup or missed delivery is raised.",
              )}
              placeholder="30"
              sideText="minutes"
              min={0}
              max={1440}
            />
          </FormControl>
          <FormControl className="min-h-[3em] max-w-[400px]">
            <NumberField
              control={control}
              name="exceptionPodGraceHours"
              label={t("A POD is missing after delivery by")}
              description={t("With no proof of delivery on file by then, POD missing is raised.")}
              placeholder="24"
              sideText="hours"
              min={1}
              max={720}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function TenderNegotiationForm() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const marginsWithheld = !useProtectedFieldAccess("Margins");

  const counterOffersEnabled = useWatch({ control, name: "tenderCounterOffersEnabled" });
  const exhaustedAction = useWatch({ control, name: "tenderExhaustedAction" });
  const escalates = exhaustedAction === "Rebroadcast" || exhaustedAction === "Rewaterfall";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Carrier tenders")}</CardTitle>
        <CardDescription>
          {t(
            "How a tender handles a carrier asking for a different rate, and what happens when every carrier has passed.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="tenderCounterOffersEnabled"
              label={t("Take counter-offers")}
              description={t(
                "Carriers can answer an offer with a rate of their own instead of only accepting or declining.",
              )}
              position="left"
            />
          </FormControl>
          {counterOffersEnabled && (
            <>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="tenderCounterMarginFloorPercent"
                  valueType="string"
                  label={t("Margin floor")}
                  description={
                    marginsWithheld
                      ? t("Your role does not include margins.")
                      : t(
                          "A counter that leaves less margin than this on the shipment's charges needs someone who can approve tenders.",
                        )
                  }
                  placeholder={marginsWithheld ? t("Hidden") : "10"}
                  disabled={marginsWithheld}
                  sideText="%"
                  decimalScale={4}
                  allowNegative
                  min={-100}
                  max={100}
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="tenderAutoAcceptCounters"
                  label={t("Take counters at or above the floor automatically")}
                  description={t(
                    "The carrier is assigned at its rate without waiting for dispatch. Counters below the floor always wait for approval.",
                  )}
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="tenderCounterDecisionMinutes"
                  label={t("Dispatch has to answer a counter within")}
                  description={t(
                    "A counter nobody answers in time lapses and the tender moves on to the next carrier.",
                  )}
                  placeholder="60"
                  sideText="minutes"
                  min={5}
                  max={1440}
                />
              </FormControl>
            </>
          )}
          <FormControl className="min-h-[3em] max-w-[400px]">
            <SelectField
              control={control}
              name="tenderExhaustedAction"
              label={t("When every carrier passes")}
              description={t(
                "Dispatch is always told. An escalation offers the move again to the carriers on the tender, leaving out any that backed out.",
              )}
              options={tenderExhaustedActionChoices}
            />
          </FormControl>
          {escalates && (
            <>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="tenderEscalationRateIncreasePercent"
                  valueType="string"
                  label={t("Raise the rates by up to")}
                  description={t(
                    "Each carrier's rate goes up by this much, but never past the margin floor. With no charges on the shipment, rates stay as they were.",
                  )}
                  placeholder="0"
                  sideText="%"
                  decimalScale={4}
                  min={0}
                  max={100}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="tenderEscalationOfferTtlMinutes"
                  label={t("Escalated offers stay open for")}
                  placeholder="60"
                  sideText="minutes"
                  min={5}
                  max={10080}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <NumberField
                  control={control}
                  name="tenderMaxEscalations"
                  label={t("Escalate at most")}
                  description={t("Rounds after the first tender; at most 5.")}
                  placeholder="1"
                  sideText="times"
                  min={0}
                  max={5}
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ComplianceForm() {
  const t = useT();

  const { control, setValue } = useFormContext<DispatchControl>();

  const enforceHosCompliance = useWatch({
    control,
    name: "enforceHosCompliance",
  });

  useEffect(() => {
    if (!enforceHosCompliance) {
      setValue("enforceMedicalCertCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceDriverQualificationCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceHazmatCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
      setValue("enforceDrugAndAlcoholCompliance", false, {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
  }, [enforceHosCompliance, setValue]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("DOT compliance enforcement")}</CardTitle>
        <CardDescription>
          {t(
            "Configure dispatch-time compliance checks for worker qualification, medical certification, hazmat eligibility, and testing requirements.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceHosCompliance"
              label={t("Enable DOT compliance enforcement")}
              description={t(
                "When enabled, the system applies configured compliance checks before assignments proceed.",
              )}
              position="left"
            />
          </FormControl>
          {enforceHosCompliance && (
            <>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceMedicalCertCompliance"
                  label={t("Medical certification validation")}
                  description={t("Require current medical certification before assignment.")}
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceDriverQualificationCompliance"
                  label={t("Driver qualification verification")}
                  description={t(
                    "Require valid driver qualification and license state before assignment.",
                  )}
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceHazmatCompliance"
                  label={t("Hazardous materials compliance")}
                  description={t("Require hazmat-specific compliance checks for regulated loads.")}
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceDrugAndAlcoholCompliance"
                  label={t("Drug and alcohol testing compliance")}
                  description={t("Require testing compliance checks before assignment.")}
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <SelectField
                  control={control}
                  name="complianceEnforcementLevel"
                  label={t("Compliance enforcement level")}
                  description={t("Select whether violations should warn, block, or be audit-only.")}
                  options={complianceEnforcementLevelChoices}
                />
              </FormControl>
            </>
          )}
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerAssign"
              label={t("Require worker assignment")}
              description={t("Prevent dispatching without an assigned worker.")}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceTrailerContinuity"
              label={t("Require trailer continuity")}
              description={t("Enforce trailer continuity rules across movement chains.")}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerPtaRestrictions"
              label={t("Enforce worker PTA restrictions")}
              description={t(
                "Apply worker availability and paid-time-away restrictions during assignment.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerTractorFleetContinuity"
              label={t("Enforce worker tractor fleet continuity")}
              description={t("Require worker-to-tractor fleet continuity where configured.")}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
