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
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  ShipmentControlSchema,
  shipmentControlSchema,
} from "@/lib/schemas/shipmentcontrol-schema";
import { api } from "@/services/api";
import type { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
  type Path,
} from "react-hook-form";
import { toast } from "sonner";

export default function ShipmentControlForm() {
  const queryClient = useQueryClient();
  const shipmentControl = useSuspenseQuery({
    ...queries.organization.getShipmentControl(),
  });

  const form = useForm({
    resolver: zodResolver(shipmentControlSchema),
    defaultValues: shipmentControl.data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: ShipmentControlSchema) => {
      return await api.shipmentControl.update(values);
    },
    onMutate: async (newValues) => {
      // * Cancel any outgoing refetches so they don't overwrite our optimistic update
      await queryClient.cancelQueries({
        queryKey: queries.organization.getShipmentControl._def,
      });

      // * Snapshot the previous value
      const previousShipmentControl = queryClient.getQueryData([
        queries.organization.getShipmentControl._def,
      ]);

      // * Optimistically update to the new value
      queryClient.setQueryData(
        [queries.organization.getShipmentControl._def],
        newValues,
      );

      return { previousShipmentControl, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Shipment control updated successfully");

      broadcastQueryInvalidation({
        queryKey: queries.organization.getShipmentControl
          ._def as unknown as string[],
        options: {
          correlationId: `update-shipment-control-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // * Reset the form to the new values
      reset(newValues);
    },
    onError: (error: APIError, _, context) => {
      // * Rollback the optimistic update
      queryClient.setQueryData(
        [queries.organization.getShipmentControl._def],
        context?.previousShipmentControl,
      );

      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<ShipmentControlSchema>, {
            message: fieldError.reason,
          });
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }

      // * Regardless of the error, reset the form to the previous state
      reset();
    },
    onSettled: () => {
      // * Invalidate the query to refresh the data
      queryClient.invalidateQueries({
        queryKey: queries.organization.getShipmentControl._def,
      });
    },
  });

  const onSubmit = useCallback(
    async (values: ShipmentControlSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <PerformanceMetricsForm />
          <AutoAssignmentForm />
          <ShipmentEntryForm />
          <ServiceFailureForm />
          <ComplianceForm />
          <DelayShipmentForm />
          <DetentionForm />
          <FormSaveDock />
        </div>
      </Form>
    </FormProvider>
  );
}

function ServiceFailureForm() {
  const { control } = useFormContext<ShipmentControlSchema>();
  const [showGracePeriod, setShowGracePeriod] = useState<boolean>(false);

  const recordServiceFailure = useWatch({
    control,
    name: "recordServiceFailures",
  });

  useEffect(() => {
    if (recordServiceFailure) {
      setShowGracePeriod(true);
    } else {
      setShowGracePeriod(false);
    }
  }, [recordServiceFailure]);

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
            <SwitchField
              control={control}
              name="recordServiceFailures"
              label="Record Service Failures"
              description="When enabled, the system will automatically track and document instances where service commitments aren't met, including late pickups, deliveries, or other operational failures that impact service quality."
              position="left"
            />
          </FormControl>
          {showGracePeriod && (
            <FormControl className="pl-10 min-h-[3em]">
              <InputField
                rules={{ required: true, min: 0 }}
                control={control}
                name="serviceFailureGracePeriod"
                label="Service Failure Grace Period"
                placeholder="Enter grace period in minutes"
                description="Defines the buffer time (in minutes) before a missed appointment is formally recorded as a service failure."
                sideText="minutes"
                className="max-w-[300px]"
              />
            </FormControl>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function DetentionForm() {
  const { control } = useFormContext<ShipmentControlSchema>();
  const [showDetentionOptions, setShowDetentionOptions] =
    useState<boolean>(false);

  const trackDetentionTime = useWatch({
    control,
    name: "trackDetentionTime",
  });

  useEffect(() => {
    if (trackDetentionTime) {
      setShowDetentionOptions(true);
    } else {
      setShowDetentionOptions(false);
    }
  }, [trackDetentionTime]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Detention Management</CardTitle>
        <CardDescription>
          Configure how the system monitors, calculates, and bills for detention
          time when drivers are delayed at shipping or receiving facilities
          beyond allowable timeframes. Proper detention tracking helps recover
          revenue, improve asset utilization, and provide documentation for
          customer negotiations.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="trackDetentionTime"
              label="Track Detention Time"
              description="When enabled, the system will automatically calculate and record detention time at pickup and delivery locations based on geofence entry/exit times or driver status updates."
              position="left"
            />
          </FormControl>
          {showDetentionOptions && (
            <>
              <FormControl className="pl-10 min-h-[3em]">
                <SwitchField
                  control={control}
                  name="autoGenerateDetentionCharges"
                  label="Auto Generate Detention Charges"
                  description="Automatically creates detention charge line items on invoices when detention exceeds the configured threshold."
                  position="left"
                  size="sm"
                />
              </FormControl>
              <FormControl className="pl-10 min-h-[3em]">
                <NumberField
                  control={control}
                  name="detentionThreshold"
                  label="Detention Threshold"
                  placeholder="Enter threshold in minutes"
                  description="Defines the standard free time allowance (in minutes) at facilities before detention charges begin accruing."
                  sideText="minutes"
                  className="max-w-[300px]"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ShipmentEntryForm() {
  const { control } = useFormContext<ShipmentControlSchema>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Shipment Processing Configuration</CardTitle>
        <CardDescription>
          Define core operational rules for shipment creation, validation, and
          management throughout the shipment lifecycle. These settings establish
          system-wide behaviors that ensure data integrity, prevent
          duplications, and determine permissible operations for users across
          all departments.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="checkForDuplicateBols"
              label="Check for Duplicate Bills of Lading"
              description="When enabled, the system will verify that each BOL number is unique during shipment creation. This prevents accidental duplications that could lead to operational confusion, billing errors, and customer service issues. Recommended for most operations to maintain data integrity."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowMoveRemovals"
              label="Allow Move Removals"
              description="When enabled, users can completely remove moves from shipments rather than canceling them. This affects shipment integrity, billing, and audit trails. Enable with caution as it allows permanent removal of shipment segments, which may impact financial reconciliation and historical reporting."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="checkHazmatSegregation"
              label="Check Hazmat Segregation"
              description="When enabled, the system will verify that hazmat shipments are properly segregated and labeled according to DOT regulations. This ensures that hazardous materials are transported safely and in compliance with all applicable laws and regulations."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ComplianceForm() {
  const { control, setValue } = useFormContext<ShipmentControlSchema>();
  const [showComplianceOptions, setShowComplianceOptions] =
    useState<boolean>(false);

  const enforceHOSCompliance = useWatch({
    control,
    name: "enforceHosCompliance",
  });

  useEffect(() => {
    if (enforceHOSCompliance) {
      setShowComplianceOptions(true);
    } else {
      setShowComplianceOptions(false);
      // * If the user disables the HOS compliance, we need to disable the other compliance options
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

function PerformanceMetricsForm() {
  const { control } = useFormContext<ShipmentControlSchema>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Performance Metrics Configuration</CardTitle>
        <CardDescription>
          Establish key performance indicators (KPIs) and operational targets
          that drive your transportation business. These metrics serve as
          benchmarks for evaluating carrier performance, influence
          performance-based compensation models, and help identify operational
          improvement opportunities. The targets set here will be used across
          dashboards, reports, and exception alerts.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="onTimeDeliveryTarget"
              label="On-Time Delivery Target"
              placeholder="Enter target in percentage"
              description="Sets the organizational benchmark for on-time delivery performance, typically 95-98% for premium service carriers."
              sideText="%"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="serviceFailureTarget"
              label="Service Failure Target"
              placeholder="Enter target in percentage"
              description="Defines the maximum acceptable percentage of service failures, establishing your company's service failure tolerance threshold."
              sideText="%"
            />
          </FormControl>
          <FormControl className="min-h-[3em]" cols="full">
            <SwitchField
              control={control}
              name="trackCustomerRejections"
              label="Track Customer Rejections"
              description="When enabled, the system will monitor and document instances where customers refuse shipments."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AutoAssignmentForm() {
  const { control } = useFormContext<ShipmentControlSchema>();

  const [showAutoAssignmentOptions, setShowAutoAssignmentOptions] =
    useState<boolean>(false);

  const enableAutoAssignment = useWatch({
    control,
    name: "enableAutoAssignment",
  });

  useEffect(() => {
    if (enableAutoAssignment) {
      setShowAutoAssignmentOptions(true);
    } else {
      setShowAutoAssignmentOptions(false);
    }
  }, [enableAutoAssignment]);

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

function DelayShipmentForm() {
  const { control } = useFormContext<ShipmentControlSchema>();
  const [showDelayOptions, setShowDelayOptions] = useState<boolean>(false);

  const autoDelayShipments = useWatch({
    control,
    name: "autoDelayShipments",
  });

  useEffect(() => {
    if (autoDelayShipments) {
      setShowDelayOptions(true);
    } else {
      setShowDelayOptions(false);
    }
  }, [autoDelayShipments]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Shipment Delay Management</CardTitle>
        <CardDescription>
          Configure how the system identifies, records, and responds to shipment
          delays throughout the transportation lifecycle. Automated delay
          detection and status updates improve operational visibility, enable
          proactive customer communication, and provide key data for service
          failure analysis. These settings determine when a shipment&apos;s
          status is automatically changed to &quot;Delayed&quot; and what
          threshold triggers escalation protocols.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="autoDelayShipments"
              label="Automatic Delay Status Updates"
              description="When enabled, the system will automatically change a shipment's status to 'Delayed' when it exceeds the configured threshold from the scheduled delivery time. This ensures consistent status reporting, eliminates manual status updates, and triggers appropriate notifications to internal staff and external stakeholders."
              position="left"
            />
          </FormControl>
          {showDelayOptions && (
            <>
              <FormControl className="pl-10 min-h-[3em]">
                <NumberField
                  control={control}
                  name="autoDelayShipmentsThreshold"
                  label="Delay Status Threshold"
                  placeholder="Enter threshold in minutes"
                  description="Defines the time variance (in minutes) from scheduled delivery or transit milestones before a shipment is flagged as 'Delayed'."
                  sideText="minutes"
                  className="max-w-[300px]"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
