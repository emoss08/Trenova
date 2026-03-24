import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  autoAssignmentStrategyChoices,
  complianceEnforcementLevelChoices,
  serviceIncidentTypeChoices,
} from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  dispatchControlSchema,
  type DispatchControl,
  serviceIncidentTypeSchema,
} from "@/types/dispatch-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";

export default function DispatchControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.dispatchControl.get(),
  });

  const form = useForm<DispatchControl>({
    resolver: zodResolver(dispatchControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.dispatchControl.get._def,
    mutationFn: async (values: DispatchControl) =>
      apiService.dispatchControlService.update(values),
    resourceName: "Dispatch Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.dispatchControl.get._def],
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
          <AutoAssignmentForm />
          <ServiceFailureForm />
          <ComplianceForm />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function AutoAssignmentForm() {
  const { control } = useFormContext<DispatchControl>();

  const enableAutoAssignment = useWatch({
    control,
    name: "enableAutoAssignment",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Automated Resource Assignment</CardTitle>
        <CardDescription>
          Configure how the system chooses workers and equipment for shipments.
          These controls influence assignment consistency, utilization, and
          dispatch throughput.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAutoAssignment"
              label="Enable Automated Assignment"
              description="When enabled, the system can automatically assign available resources to shipments."
              position="left"
            />
          </FormControl>
          {enableAutoAssignment && (
            <FormControl className="min-h-[3em] max-w-[400px] pl-10">
              <SelectField
                control={control}
                name="autoAssignmentStrategy"
                label="Assignment Optimization Strategy"
                description="Select the primary strategy used when matching resources to shipments."
                options={autoAssignmentStrategyChoices}
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ServiceFailureForm() {
  const { control } = useFormContext<DispatchControl>();

  const recordServiceFailures = useWatch({
    control,
    name: "recordServiceFailures",
  });

  const showFailureFields =
    recordServiceFailures !== serviceIncidentTypeSchema.enum.Never;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Service Failure Monitoring</CardTitle>
        <CardDescription>
          Define which service failures to track and when they should be
          recorded. These settings drive operational reporting and exception
          visibility.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SelectField
              control={control}
              name="recordServiceFailures"
              label="Record Service Failures"
              description="Choose which incident types should be captured as service failures."
              options={serviceIncidentTypeChoices}
            />
          </FormControl>
          {showFailureFields && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="serviceFailureGracePeriod"
                  label="Service Failure Grace Period"
                  placeholder="Enter grace period in minutes"
                  description="Defines the delay buffer before an eligible incident is recorded as a failure."
                  sideText="minutes"
                  min={1}
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="serviceFailureTarget"
                  label="Service Failure Target"
                  placeholder="Enter target percentage"
                  description="Optional threshold for acceptable service failure rate."
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

function ComplianceForm() {
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
        <CardTitle>DOT Compliance Enforcement</CardTitle>
        <CardDescription>
          Configure dispatch-time compliance checks for worker qualification,
          medical certification, hazmat eligibility, and testing requirements.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceHosCompliance"
              label="Enable DOT Compliance Enforcement"
              description="When enabled, the system applies configured compliance checks before assignments proceed."
              position="left"
            />
          </FormControl>
          {enforceHosCompliance && (
            <>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceMedicalCertCompliance"
                  label="Medical Certification Validation"
                  description="Require current medical certification before assignment."
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceDriverQualificationCompliance"
                  label="Driver Qualification Verification"
                  description="Require valid driver qualification and license state before assignment."
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceHazmatCompliance"
                  label="Hazardous Materials Compliance"
                  description="Require hazmat-specific compliance checks for regulated loads."
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] pl-10">
                <SwitchField
                  control={control}
                  name="enforceDrugAndAlcoholCompliance"
                  label="Drug and Alcohol Testing Compliance"
                  description="Require testing compliance checks before assignment."
                  position="left"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px] pl-10">
                <SelectField
                  control={control}
                  name="complianceEnforcementLevel"
                  label="Compliance Enforcement Level"
                  description="Select whether violations should warn, block, or be audit-only."
                  options={complianceEnforcementLevelChoices}
                />
              </FormControl>
            </>
          )}
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerAssign"
              label="Require Worker Assignment"
              description="Prevent dispatching without an assigned worker."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceTrailerContinuity"
              label="Require Trailer Continuity"
              description="Enforce trailer continuity rules across movement chains."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerPtaRestrictions"
              label="Enforce Worker PTA Restrictions"
              description="Apply worker availability and paid-time-away restrictions during assignment."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceWorkerTractorFleetContinuity"
              label="Enforce Worker Tractor Fleet Continuity"
              description="Require worker-to-tractor fleet continuity where configured."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
