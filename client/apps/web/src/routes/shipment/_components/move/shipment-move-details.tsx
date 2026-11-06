import { EmptyState } from "@/components/empty-state";
import { shipment } from "@/lib/queries/shipment";
import { useQuery } from "@tanstack/react-query";
import { CapabilityExplainer } from "@trenova/shared/components/capability-explainer";
import {
  MarkerPin01Icon,
  PlusIcon,
  RouteIcon,
  Truck01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getProfile, isMoveRemovalAllowed } from "@trenova/shared/lib/capability";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useEffect, useRef, useState } from "react";
import { useFieldArray, useFormContext, useFormState, useWatch } from "react-hook-form";
import { MoveEditDialog, type MoveDialogState } from "../shipment-move-edit-dialog";
import { MoveCard } from "./move-card";

/**
 * The shipment's moves, each a compact card of its route and coverage, with adding, editing and
 * removing a move. Removal follows the server's rule: always for an unsaved shipment, and past
 * that as the mode profile (or, without one, the organisation's setting) allows.
 *
 * A stop's fields are drawn only in the move editor, so when a save fails on one, the form has
 * no field of its own to focus. When the moves hold the only problems, the first one takes focus
 * here instead, bringing its move and stop into view.
 */
export default function ShipmentMoveDetails() {
  const t = useT();

  const { control } = useFormContext<Shipment>();
  const shipmentID = useWatch({ control, name: "id" });
  const {
    fields: moveFields,
    append: appendMove,
    remove: removeMove,
  } = useFieldArray({
    control,
    name: "moves",
    keyName: "fieldId",
  });

  const [moveDialog, setMoveDialog] = useState<MoveDialogState>({ open: false });
  const { errors, submitCount, isSubmitting } = useFormState({ control });
  const movesRef = useRef<HTMLDivElement>(null);
  const handledSubmit = useRef(submitCount);

  useEffect(() => {
    if (isSubmitting || submitCount === handledSubmit.current) return;
    handledSubmit.current = submitCount;
    // A problem on a field drawn above the moves is focused by the form itself.
    const elsewhere = Object.keys(errors).some((key) => key !== "moves" && key !== "root");
    if (elsewhere || !errors.moves) return;
    const card = movesRef.current?.querySelector("[data-move-card][data-invalid]");
    const target =
      card?.querySelector<HTMLElement>("[data-stop-problem]") ??
      card?.querySelector<HTMLElement>("[data-move-problem]");
    target?.focus();
  }, [errors, isSubmitting, submitCount]);
  const { data: shipmentUIPolicy } = useQuery({ ...shipment.uiPolicy() });
  const profile = getProfile(shipmentUIPolicy);
  // An unsaved shipment has no persisted moves to detach history from, so removal
  // is always allowed there. Past that, the decision has to match the server's:
  // the profile rule wins, the org flag is only a fallback.
  const allowPersistedMoveRemoval =
    !shipmentID || isMoveRemovalAllowed(profile, shipmentUIPolicy?.allowMoveRemovals);

  function handleDialogClose() {
    setMoveDialog({ open: false });
  }

  function handleDialogCancel() {
    if (moveDialog.open && moveDialog.isNew) {
      removeMove(moveDialog.moveIndex);
    }
    setMoveDialog({ open: false });
  }

  function handleAddMove() {
    const newIndex = moveFields.length;
    appendMove({
      status: "New",
      loaded: true,
      sequence: newIndex,
      distance: 0,
      stops: [
        {
          status: "New",
          type: "Pickup",
          scheduleType: "Open",
          handling: "Live",
          locationId: "",
          sequence: 0,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
        {
          status: "New",
          type: "Delivery",
          scheduleType: "Open",
          handling: "Live",
          locationId: "",
          sequence: 1,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
      ],
    });
    setMoveDialog({ open: true, moveIndex: newIndex, isNew: true });
  }

  return (
    <>
      <FormSection
        title={t("Move details")}
        description={t("Each leg's route and who covers it")}
        action={
          <div className="flex items-center gap-1.5">
            <CapabilityExplainer profile={profile} field="moves" />
            {moveFields.length > 0 && (
              <Button type="button" variant="outline" size="xxs" onClick={handleAddMove}>
                <PlusIcon className="size-3" />
                {t("Add move")}
              </Button>
            )}
          </div>
        }
      >
        <div ref={movesRef} className="contents">
          {moveFields.map((field, moveIndex) => (
            <MoveCard
              key={field.fieldId}
              moveIndex={moveIndex}
              allowPersistedMoveRemoval={allowPersistedMoveRemoval}
              onEdit={(focus) => setMoveDialog({ open: true, moveIndex, isNew: false, focus })}
              onRemove={() => {
                if (field.id && !allowPersistedMoveRemoval) {
                  return;
                }
                removeMove(moveIndex);
              }}
            />
          ))}
        </div>
        {moveFields.length === 0 && (
          <EmptyState
            className="rounded-lg border p-4"
            title={t("No moves yet")}
            description={t("Add a move to define the route for this shipment")}
            icons={[MarkerPin01Icon, RouteIcon, Truck01Icon]}
            action={{ label: t("Add first move"), onClick: handleAddMove, icon: PlusIcon }}
          />
        )}
      </FormSection>
      <MoveEditDialog
        state={moveDialog}
        onClose={handleDialogClose}
        onCancel={handleDialogCancel}
      />
    </>
  );
}
