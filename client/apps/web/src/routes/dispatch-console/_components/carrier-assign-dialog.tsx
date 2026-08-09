import {
  CarrierAssignmentFields,
  CarrierEligibilityAlerts,
  carrierEligibilityBlocksSubmit,
} from "@/components/carrier-assignment/carrier-assignment-fields";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { DispatchAssignMoveToCarrierInput } from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  carrierAssignmentPayloadSchema,
  emptyCarrierAssignmentPayload,
  type CarrierAssignmentPayload,
} from "@trenova/shared/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm, useWatch } from "react-hook-form";

export function toDispatchCarrierAssignmentInput(
  moveId: string,
  payload: CarrierAssignmentPayload,
  replace: boolean,
): DispatchAssignMoveToCarrierInput {
  return {
    moveId,
    carrierId: payload.carrierId,
    rateMethod: payload.rateMethod,
    baseRate: payload.baseRate.toString(),
    fuelSurcharge: payload.fuelSurcharge != null ? payload.fuelSurcharge.toString() : undefined,
    accessorials: payload.accessorials.map((accessorial) => ({
      accessorialChargeId: accessorial.accessorialChargeId ?? undefined,
      description: accessorial.description,
      amount: accessorial.amount.toString(),
    })),
    proNumber: payload.proNumber || undefined,
    externalDriverName: payload.externalDriverName || undefined,
    externalDriverPhone: payload.externalDriverPhone || undefined,
    externalTractorNumber: payload.externalTractorNumber || undefined,
    externalTrailerNumber: payload.externalTrailerNumber || undefined,
    replace,
    overrideInsuranceWarning: payload.overrideInsuranceWarning,
  };
}

/**
 * Brokered coverage entry point for the console. The eligibility preview runs the moment
 * a carrier is picked so the dispatcher sees compliance blockers and insurance warnings
 * before committing a rate, not as a failed mutation afterwards.
 */
export function CarrierAssignDialog({
  move,
  isAssigning,
  onCancel,
  onConfirm,
}: {
  move: DispatchBoardMove;
  isAssigning: boolean;
  onCancel: () => void;
  onConfirm: (input: DispatchAssignMoveToCarrierInput) => Promise<unknown>;
}) {
  const replace = move.coverageType === "carrier" && !!move.carrierAssignmentId;

  const form = useForm<CarrierAssignmentPayload>({
    resolver: zodResolver(carrierAssignmentPayloadSchema),
    defaultValues: { ...emptyCarrierAssignmentPayload },
  });
  const { control, handleSubmit } = form;

  const carrierId = useWatch({ control, name: "carrierId" });
  const overrideInsuranceWarning = useWatch({ control, name: "overrideInsuranceWarning" });

  const { data: eligibility, isLoading: isPreviewLoading } = useQuery({
    ...dispatchConsoleQueries.carrierAssignmentPreview({ carrierId }),
    enabled: !!carrierId,
  });

  const onSubmit = useCallback(
    async (values: CarrierAssignmentPayload) => {
      // Failures surface as toasts from the mutation; the dialog stays open with the
      // dispatcher's input intact so they can correct and retry.
      try {
        await onConfirm(toDispatchCarrierAssignmentInput(move.moveId, values, replace));
      } catch {
        // handled by the mutation's onError
      }
    },
    [move.moveId, onConfirm, replace],
  );

  const submitBlocked =
    !!carrierId &&
    (isPreviewLoading || carrierEligibilityBlocksSubmit(eligibility, overrideInsuranceWarning));

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>
            {replace ? "Replace Carrier Assignment" : "Assign Move to Carrier"}
          </DialogTitle>
          <DialogDescription>
            {move.proNumber} · {move.originCity}, {move.originState} → {move.destinationCity},{" "}
            {move.destinationState}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit(onSubmit)(event);
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
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={submitBlocked}
              isLoading={isAssigning}
              loadingText="Assigning..."
            >
              {replace ? "Replace carrier" : "Assign to carrier"}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
