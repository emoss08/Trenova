import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { ExternalLink } from "@/components/ui/link";
import { NumberField } from "@/components/ui/number-input";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { serviceIncidentTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import {
  dispatchControlSchema,
  DispatchControlSchema,
  ServiceIncidentType,
} from "@/lib/schemas/dispatchcontrol-schema";
import { api } from "@/services/api";
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
  const dispatchControl = useSuspenseQuery({
    ...queries.organization.getDispatchControl(),
  });

  const form = useForm({
    resolver: zodResolver(dispatchControlSchema),
    defaultValues: dispatchControl.data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.organization.getDispatchControl._def,
    mutationFn: async (values: DispatchControlSchema) =>
      api.dispatchControl.update(values),
    successMessage: "Dispatch control updated successfully",
    resourceName: "Dispatch Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.organization.getDispatchControl._def],
  });

  const onSubmit = useCallback(
    async (values: DispatchControlSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <DispatchControlOuter>
          <AutoAssignmentForm />
          <ServiceFailureForm />
          <ComplianceForm />
          <FormSaveDock />
        </DispatchControlOuter>
      </Form>
    </FormProvider>
  );
}

function DispatchControlOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4 pb-14">{children}</div>;
}

function ServiceFailureForm() {
  const { control } = useFormContext<DispatchControlSchema>();

  const recordServiceFailures = useWatch({
    control,
    name: "recordServiceFailures",
  });

  const showGracePeriod =
    recordServiceFailures !== ServiceIncidentType.enum.Never;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Service Failure Monitoring</CardTitle>
        <CardDescription>
          Configure how the system tracks, records, and manages service failures
          to maintain customer satisfaction and meet contractual service level
          agreements (SLAs). These settings affect performance metrics,
          reporting, and alerting mechanisms.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SelectField
              control={control}
              name="recordServiceFailures"
              label="Record Service Failures"
              description="When enabled, the system will automatically track and document instances where service commitments aren't met, including late pickups, deliveries, or other operational failures that impact service quality."
              options={serviceIncidentTypeChoices}
              placeholder="Select service failure type"
            />
          </FormControl>
          {showGracePeriod && (
            <div className="flex flex-col pl-10">
              <FormControl className="min-h-[3em] max-w-[400px]">
                <InputField
                  rules={{ required: true, min: 0 }}
                  control={control}
                  name="serviceFailureGracePeriod"
                  label="Service Failure Grace Period"
                  placeholder="Enter grace period in minutes"
                  description="Defines the buffer time (in minutes) before a missed appointment is formally recorded as a service failure."
                  sideText="minutes"
                />
              </FormControl>
              <FormControl className="min-h-[3em] max-w-[400px]">
                <NumberField
                  control={control}
                  name="serviceFailureTarget"
                  label="Service Failure Target"
                  placeholder="Enter target in percentage"
                  description="Defines the maximum acceptable percentage of service failures, establishing your company's service failure tolerance threshold."
                  sideText="%"
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
  const { control, setValue } = useFormContext<DispatchControlSchema>();

  const enforceHOSCompliance = useWatch({
    control,
    name: "enforceHosCompliance",
  });

  const showComplianceOptions = enforceHOSCompliance;

  useEffect(() => {
    if (!enforceHOSCompliance) {
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
  }, [enforceHOSCompliance, setValue]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>DOT Compliance Enforcement (US Only)</CardTitle>
        <CardDescription>
          Configure comprehensive Department of Transportation (DOT) compliance
          validation across your entire operation. These critical settings
          enforce Federal Motor Carrier Safety Administration (FMCSA)
          regulations and maintain your company&apos;s safety rating. Our
          multi-layered compliance system validates driver qualifications,
          documentation, endorsements, and operational parameters to prevent
          violations before they occur. For detailed implementation guidance,
          review our{" "}
          <ExternalLink href="https://docs.trenova.io/compliance/enforcement-framework">
            Compliance Enforcement Framework
          </ExternalLink>
          .
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enforceHosCompliance"
              label="Enable DOT Compliance Enforcement"
              description="When enabled, the system performs comprehensive validation across all major DOT compliance domains to prevent violations before they occur. This protection helps maintain your safety rating, reduce violation risk during roadside inspections, and minimize potential liability exposure from non-compliant operations."
              position="left"
            />
          </FormControl>
          {showComplianceOptions && (
            <>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  name="enforceMedicalCertCompliance"
                  label="Medical Certification Validation"
                  description="Verifies that drivers maintain current medical examiner's certificates (49 CFR §391.45) and prevents assignments when certifications are expired or approaching expiration. This ensures drivers meet physical qualification standards and helps prevent citations for operating with expired medical cards."
                  position="left"
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  name="enforceDriverQualificationCompliance"
                  label="Driver Qualification Verification"
                  description="Validates driver qualification files against DOT requirements (49 CFR §391.11), including age verification, license validity, endorsement verification, and required records. Prevents assignment of unqualified drivers and maintains regulatory compliance with driver qualification requirements."
                  position="left"
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  name="enforceHazmatCompliance"
                  label="Hazardous Materials Compliance"
                  description="Ensures that only properly endorsed and trained drivers (49 CFR §383.93) are assigned to hazardous materials shipments. Validates hazmat endorsement expiration dates and prevents assignment of drivers without current hazmat credentials to regulated loads."
                  position="left"
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  name="enforceDrugAndAlcoholCompliance"
                  label="Drug and Alcohol Testing Compliance"
                  description="Validates compliance with drug and alcohol testing requirements (49 CFR §382.301), including pre-employment, random, post-accident, and reasonable suspicion testing. Prevents assignment of drivers who aren't compliant with testing requirements."
                  position="left"
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <SelectField
                  className="max-w-[300px]"
                  control={control}
                  name="complianceEnforcementLevel"
                  label="Compliance Enforcement Level"
                  description="Determines system behavior when compliance violations are detected."
                  options={[
                    {
                      label: "Warning",
                      value: "Warning",
                      description:
                        "Notifies users but allows operations to continue",
                      color: "#f59e0b",
                    },
                    {
                      label: "Block",
                      value: "Block",
                      description: "Prevents non-compliant operations entirely",
                      color: "#b91c1c",
                    },
                    {
                      label: "Audit",
                      value: "Audit",
                      description:
                        "Logs violations for review without interfering with operations",
                      color: "#7e22ce",
                    },
                  ]}
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AutoAssignmentForm() {
  const { control } = useFormContext<DispatchControlSchema>();

  const enableAutoAssignment = useWatch({
    control,
    name: "enableAutoAssignment",
  });

  const showAutoAssignmentOptions = enableAutoAssignment;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Automated Resource Assignment</CardTitle>
        <CardDescription>
          Configure intelligent dispatching algorithms that optimize driver-load
          matching based on your operational priorities. Automated assignment
          reduces dispatcher workload, minimizes empty miles, improves asset
          utilization, and helps maintain hours of service compliance. These
          settings determine how the system allocates drivers, tractors, and
          trailers to shipments throughout your network.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="enableAutoAssignment"
              label="Enable Automated Assignment"
              description="When enabled, the system will automatically match available drivers and equipment to shipments based on your selected optimization strategy."
              position="left"
            />
          </FormControl>
          {showAutoAssignmentOptions && (
            <FormControl className="pl-10 min-h-[3em]">
              <SelectField
                control={control}
                name="autoAssignmentStrategy"
                label="Assignment Optimization Strategy"
                description="Determines the primary optimization goal when matching drivers to loads."
                options={[
                  {
                    label: "Proximity",
                    value: "Proximity",
                    description:
                      "Prioritizes drivers closest to the pickup location to minimize deadhead miles, reduce fuel consumption, and improve on-time pickup performance.",
                    color: "#0369a1",
                  },
                  {
                    label: "Availability",
                    value: "Availability",
                    description:
                      "Prioritizes drivers with the most available Hours of Service and fewest upcoming commitments.",
                    color: "#15803d",
                  },
                  {
                    label: "Load Balancing",
                    value: "LoadBalancing",
                    description:
                      "Distributes work evenly across your driver fleet to ensure equitable assignment of miles, stops, and revenue opportunities.",
                    color: "#ec4899",
                  },
                ]}
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
