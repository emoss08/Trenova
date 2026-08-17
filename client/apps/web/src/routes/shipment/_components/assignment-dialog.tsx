import {
  TractorAutocompleteField,
  TrailerAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import {
  CarrierAssignmentFields,
  CarrierEligibilityAlerts,
  carrierEligibilityBlocksSubmit,
} from "@/components/carrier-assignment/carrier-assignment-fields";
import { handleMutationError } from "@/hooks/use-api-mutation";
import type { SelectOption } from "@/lib/graphql/select-options";
import { LocateTrailerDialog } from "@/routes/trailer/_components/locate-trailer-dialog";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { ApiRequestError } from "@trenova/shared/lib/api";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import type {
  Assignment,
  AssignmentPayload,
  CarrierAssignment,
  CarrierAssignmentPayload,
  CarrierAssignmentPayloadInput,
} from "@trenova/shared/types/shipment";
import {
  assignmentPayloadSchema,
  carrierAssignmentPayloadSchema,
  emptyCarrierAssignmentPayload,
  isActiveCarrierAssignment,
} from "@trenova/shared/types/shipment";
import { Building2Icon, TriangleAlertIcon, UserIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { AssignmentHosFeasibility } from "./assignment-hos-feasibility";

type CoverageMode = "driver" | "carrier";

type AssignmentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moveId: string;
  shipmentId?: string | null;
  existingAssignment?: Assignment | null;
  existingCarrierAssignment?: CarrierAssignment | null;
  /**
   * Field values that take precedence over the existing assignment when the
   * dialog opens — e.g. the target driver after a drag-to-reassign on the
   * dispatch timeline. Must be referentially stable while the dialog is open.
   */
  prefill?: Partial<AssignmentPayload> | null;
  onAssigned?: (assignment: Assignment) => void;
  onCarrierAssigned?: (carrierAssignment: CarrierAssignment) => void;
};

export function AssignmentDialog({
  open,
  onOpenChange,
  moveId,
  shipmentId,
  existingAssignment,
  existingCarrierAssignment,
  prefill,
  onAssigned,
  onCarrierAssigned,
}: AssignmentDialogProps) {
  const queryClient = useQueryClient();
  const isEditing = !!existingAssignment?.id;
  const hasCarrierCoverage = isActiveCarrierAssignment(existingCarrierAssignment);
  // Driver coverage is refused by the API for an organization without asset
  // operations, so the mode is not offered rather than offered and rejected.
  const capabilities = useOrgCapabilities();
  const canAssignDrivers = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );
  const [mode, setMode] = useState<CoverageMode>(
    hasCarrierCoverage || !canAssignDrivers ? "carrier" : "driver",
  );
  // A move that already carries a driver at an organization that cannot assign
  // one: neither mode can act on it, so the dialog explains rather than offers.
  const driverAssignmentBlocked = !canAssignDrivers && isEditing;

  const [continuityError, setContinuityError] = useState<{
    message: string;
    trailerId: string;
    pickupLocationId: string;
  } | null>(null);
  const [complianceViolations, setComplianceViolations] = useState<string[]>([]);
  const [locateDialogOpen, setLocateDialogOpen] = useState(false);

  const form = useForm({
    resolver: zodResolver(assignmentPayloadSchema),
    defaultValues: {
      tractorId: "",
      trailerId: null,
      primaryWorkerId: "",
      secondaryWorkerId: null,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setValue,
    getValues,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      tractorId: existingAssignment?.tractorId ?? "",
      trailerId: existingAssignment?.trailerId ?? null,
      primaryWorkerId: existingAssignment?.primaryWorkerId ?? "",
      secondaryWorkerId: existingAssignment?.secondaryWorkerId ?? null,
      ...prefill,
    });
  }, [open, existingAssignment, prefill, reset]);

  useEffect(() => {
    if (!open) return;
    setMode(hasCarrierCoverage || !canAssignDrivers ? "carrier" : "driver");
  }, [open, hasCarrierCoverage, canAssignDrivers]);

  const watchedTrailerId = useWatch({ control, name: "trailerId" });
  useEffect(() => {
    setContinuityError(null);
  }, [watchedTrailerId]);

  const watchedPrimaryWorker = useWatch({ control, name: "primaryWorkerId" });
  const watchedSecondaryWorker = useWatch({ control, name: "secondaryWorkerId" });
  useEffect(() => {
    setComplianceViolations([]);
  }, [watchedPrimaryWorker, watchedSecondaryWorker]);

  const { mutateAsync } = useMutation({
    mutationFn: (payload: AssignmentPayload) =>
      isEditing
        ? apiService.assignmentService.reassign(moveId, payload)
        : apiService.assignmentService.assignToMove(moveId, payload),
    onSuccess: (data: Assignment) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      onAssigned?.(data);
      toast.success(isEditing ? "Reassigned successfully" : "Assigned successfully");
    },
    onError: (error: ApiRequestError) => {
      if (error.isBusinessError()) {
        const params = error.getParams();
        if (params.trailerId && params.pickupLocationId) {
          setContinuityError({
            message: error.data.detail || error.data.title,
            trailerId: params.trailerId,
            pickupLocationId: params.pickupLocationId,
          });
          return;
        }
      }
      handleMutationError({ error, form, resourceName: "Assignment" });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
    setContinuityError(null);
    setComplianceViolations([]);
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: AssignmentPayload) => {
      setComplianceViolations([]);
      try {
        await apiService.assignmentService.checkWorkerCompliance(moveId, {
          primaryWorkerId: values.primaryWorkerId,
          secondaryWorkerId: values.secondaryWorkerId,
        });
      } catch (err) {
        if (err instanceof ApiRequestError && err.isValidationError()) {
          const errors = err.getFieldErrors();
          setComplianceViolations(errors.map((e) => e.message));
          return;
        }
      }
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose, moveId],
  );

  const handleTractorChange = useCallback(
    (tractor: SelectOption | null) => {
      if (tractor?.meta) {
        const currentPrimary = getValues("primaryWorkerId");
        const currentSecondary = getValues("secondaryWorkerId");
        const primaryWorkerId = tractor.meta.primaryWorkerId;
        const secondaryWorkerId = tractor.meta.secondaryWorkerId;

        if (!currentPrimary && typeof primaryWorkerId === "string") {
          setValue("primaryWorkerId", primaryWorkerId);
        }
        if (!currentSecondary && typeof secondaryWorkerId === "string") {
          setValue("secondaryWorkerId", secondaryWorkerId);
        }
      }
    },
    [setValue, getValues],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-150">
        <DialogHeader>
          <DialogTitle>
            {driverAssignmentBlocked
              ? "Move Coverage"
              : mode === "carrier"
                ? hasCarrierCoverage
                  ? "Replace Carrier Assignment"
                  : "Assign Move to Carrier"
                : isEditing
                  ? "Reassign Move"
                  : "Assign Move"}
          </DialogTitle>
          <DialogDescription>
            {driverAssignmentBlocked
              ? "This move is covered by a driver, which this organization cannot reassign."
              : mode === "carrier"
                ? "Broker this move to an external carrier with its rate and reference details."
                : isEditing
                  ? "Update the tractor, trailer, and worker assignments for this move."
                  : "Assign a tractor, trailer, and workers to this move."}
          </DialogDescription>
        </DialogHeader>
        {canAssignDrivers && (
          <Tabs value={mode} onValueChange={(value) => setMode(value as CoverageMode)}>
            <TabsList
              variant="underline"
              className="w-full border-b border-border"
              aria-label="Coverage type"
            >
              <TabsTab value="driver">
                <UserIcon className="size-4" />
                Driver
              </TabsTab>
              <TabsTab value="carrier">
                <Building2Icon className="size-4" />
                Carrier
              </TabsTab>
            </TabsList>
          </Tabs>
        )}
        {driverAssignmentBlocked ? (
          <>
            <Alert variant="default">
              <TriangleAlertIcon />
              <AlertTitle>Driver assignment is not enabled</AlertTitle>
              <AlertDescription>
                This organization does not run its own drivers, so an existing driver assignment
                cannot be changed here. Unassign the driver to release the move, then cover it with
                a carrier.
              </AlertDescription>
            </Alert>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Close
              </Button>
            </DialogFooter>
          </>
        ) : mode === "carrier" ? (
          isEditing ? (
            <>
              <Alert variant="default">
                <TriangleAlertIcon />
                <AlertTitle>Move is covered by a driver</AlertTitle>
                <AlertDescription>
                  This move already has a driver assignment. Unassign the driver before brokering
                  the move to an external carrier.
                </AlertDescription>
              </Alert>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={handleClose}>
                  Close
                </Button>
              </DialogFooter>
            </>
          ) : (
            <CarrierAssignmentTab
              moveId={moveId}
              existingCarrierAssignment={hasCarrierCoverage ? existingCarrierAssignment : null}
              onCarrierAssigned={onCarrierAssigned}
              onClose={handleClose}
            />
          )
        ) : hasCarrierCoverage ? (
          <>
            <Alert variant="default">
              <TriangleAlertIcon />
              <AlertTitle>Move is covered by a carrier</AlertTitle>
              <AlertDescription>
                This move is brokered to
                {existingCarrierAssignment?.carrier?.name
                  ? ` ${existingCarrierAssignment.carrier.name}`
                  : " an external carrier"}
                . Cancel the carrier assignment before assigning a driver.
              </AlertDescription>
            </Alert>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Close
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            {complianceViolations.length > 0 && (
              <Alert variant="destructive">
                <TriangleAlertIcon />
                <AlertTitle>Compliance Violations</AlertTitle>
                <AlertDescription>
                  <ul className="list-disc pl-4">
                    {complianceViolations.map((msg, idx) => (
                      <li key={idx}>{msg}</li>
                    ))}
                  </ul>
                </AlertDescription>
              </Alert>
            )}
            {continuityError && (
              <Alert variant="default">
                <TriangleAlertIcon />
                <AlertTitle>Trailer Location Mismatch</AlertTitle>
                <AlertDescription>{continuityError.message}</AlertDescription>
                <AlertAction>
                  <Button
                    type="button"
                    size="xs"
                    variant="outline"
                    onClick={() => setLocateDialogOpen(true)}
                  >
                    Locate Trailer
                  </Button>
                </AlertAction>
              </Alert>
            )}
            <Form
              onSubmit={(e) => {
                e.stopPropagation();
                void handleSubmit(onSubmit)(e);
              }}
            >
              <FormGroup cols={2} className="pb-4">
                <FormControl>
                  <TractorAutocompleteField
                    control={control}
                    name="tractorId"
                    label="Tractor"
                    placeholder="Select tractor"
                    rules={{ required: true }}
                    onOptionChange={handleTractorChange}
                  />
                </FormControl>
                <FormControl>
                  <TrailerAutocompleteField
                    control={control}
                    name="trailerId"
                    label="Trailer"
                    placeholder="Select trailer"
                    clearable
                  />
                </FormControl>
                <FormControl>
                  <WorkerAutocompleteField
                    control={control}
                    name="primaryWorkerId"
                    label="Primary Worker"
                    placeholder="Select primary worker"
                    rules={{ required: true }}
                    clearable
                  />
                </FormControl>
                <FormControl>
                  <WorkerAutocompleteField
                    control={control}
                    name="secondaryWorkerId"
                    label="Secondary Worker"
                    placeholder="Select secondary worker"
                    clearable
                  />
                </FormControl>
              </FormGroup>
              <AssignmentHosFeasibility
                open={open}
                shipmentId={shipmentId}
                selectedWorkerId={watchedPrimaryWorker}
                onSelectWorker={(workerId) =>
                  setValue("primaryWorkerId", workerId, { shouldDirty: true, shouldValidate: true })
                }
              />
              <DialogFooter>
                <Button type="button" variant="outline" onClick={handleClose}>
                  Cancel
                </Button>
                <Button type="submit" isLoading={isSubmitting} loadingText="Saving...">
                  {isEditing ? "Reassign" : "Assign"}
                </Button>
              </DialogFooter>
            </Form>
          </>
        )}
      </DialogContent>
      {continuityError && (
        <LocateTrailerDialog
          open={locateDialogOpen}
          onOpenChange={setLocateDialogOpen}
          trailerId={continuityError.trailerId}
          targetLocationId={continuityError.pickupLocationId}
          onLocated={() => setContinuityError(null)}
        />
      )}
    </Dialog>
  );
}

function toCarrierAssignmentDefaults(
  existing: CarrierAssignment | null | undefined,
): CarrierAssignmentPayload {
  if (!existing) return { ...emptyCarrierAssignmentPayload };
  return {
    carrierId: existing.carrierId ?? "",
    rateMethod: existing.rateMethod,
    baseRate: Number(existing.baseRate ?? 0),
    fuelSurcharge: existing.fuelSurcharge != null ? Number(existing.fuelSurcharge) : null,
    accessorials: (existing.accessorials ?? []).map((accessorial) => ({
      accessorialChargeId: accessorial.accessorialChargeId ?? null,
      description: accessorial.description,
      amount: Number(accessorial.amount ?? 0),
    })),
    proNumber: existing.proNumber ?? "",
    externalDriverName: existing.externalDriverName ?? "",
    externalDriverPhone: existing.externalDriverPhone ?? "",
    externalTractorNumber: existing.externalTractorNumber ?? "",
    externalTrailerNumber: existing.externalTrailerNumber ?? "",
    overrideInsuranceWarning: false,
  };
}

function CarrierAssignmentTab({
  moveId,
  existingCarrierAssignment,
  onCarrierAssigned,
  onClose,
}: {
  moveId: string;
  existingCarrierAssignment?: CarrierAssignment | null;
  onCarrierAssigned?: (carrierAssignment: CarrierAssignment) => void;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isReplacing = isActiveCarrierAssignment(existingCarrierAssignment);

  const form = useForm<CarrierAssignmentPayloadInput, unknown, CarrierAssignmentPayload>({
    resolver: zodResolver(carrierAssignmentPayloadSchema),
    defaultValues: toCarrierAssignmentDefaults(existingCarrierAssignment),
  });
  const {
    control,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const carrierId = useWatch({ control, name: "carrierId" });
  const overrideInsuranceWarning = useWatch({ control, name: "overrideInsuranceWarning" });

  const { data: eligibility, isLoading: isPreviewLoading } = useQuery({
    queryKey: ["carrier-assignment-preview", moveId, carrierId],
    queryFn: () => apiService.carrierAssignmentService.preview(moveId, carrierId),
    enabled: !!carrierId,
    staleTime: 30_000,
  });

  const { mutateAsync } = useMutation({
    mutationFn: (payload: CarrierAssignmentPayload) =>
      apiService.carrierAssignmentService.assign(moveId, payload, { replace: isReplacing }),
    onSuccess: (carrierAssignment: CarrierAssignment) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      onCarrierAssigned?.(carrierAssignment);
      toast.success(isReplacing ? "Carrier assignment replaced" : "Move assigned to carrier");
    },
    onError: (error: ApiRequestError) => {
      handleMutationError({ error, form, resourceName: "Carrier Assignment" });
    },
  });

  const onSubmit = useCallback(
    async (values: CarrierAssignmentPayload) => {
      // Failures are surfaced by handleMutationError; keep the dialog open with the
      // entered rate intact so the dispatcher can correct and retry.
      try {
        await mutateAsync(values);
        onClose();
      } catch {
        // handled by the mutation's onError
      }
    },
    [mutateAsync, onClose],
  );

  const submitBlocked =
    !!carrierId &&
    (isPreviewLoading || carrierEligibilityBlocksSubmit(eligibility, overrideInsuranceWarning));

  return (
    <Form
      onSubmit={(e) => {
        e.stopPropagation();
        void handleSubmit(onSubmit)(e);
      }}
    >
      <ScrollArea className="max-h-[60vh] pr-2">
        <div className="flex flex-col gap-3 pb-4">
          <CarrierEligibilityAlerts
            control={control}
            eligibility={carrierId ? eligibility : undefined}
            isLoading={!!carrierId && isPreviewLoading}
          />
          <CarrierAssignmentFields form={form} />
        </div>
      </ScrollArea>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button
          type="submit"
          disabled={submitBlocked}
          isLoading={isSubmitting}
          loadingText="Saving..."
        >
          {isReplacing ? "Replace carrier" : "Assign to carrier"}
        </Button>
      </DialogFooter>
    </Form>
  );
}
