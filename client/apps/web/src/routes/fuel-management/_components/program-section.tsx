import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  deleteFuelSurchargeProgram,
  type FuelProgramCurrentRate,
} from "@/lib/graphql/fuel-surcharge";
import { fuelSurchargeMethodChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Fuel, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { ProgramPanel } from "./program-panel";

function methodLabel(method: string) {
  return fuelSurchargeMethodChoices.find((choice) => choice.value === method)?.label ?? method;
}

function currentRateDisplay(entry: FuelProgramCurrentRate) {
  if (entry.ratePerMile != null) {
    return { value: `$${Number(entry.ratePerMile).toFixed(4)}`, unit: "/ mile" };
  }
  if (entry.percent != null) {
    return { value: `${Number(entry.percent).toFixed(2)}%`, unit: "of linehaul" };
  }
  if (entry.flatAmount != null) {
    return { value: `$${Number(entry.flatAmount).toFixed(2)}`, unit: "per shipment" };
  }
  return null;
}

export default function ProgramSection() {
  const { data: entries, isLoading } = useQuery(queries.fuelSurcharge.currentRates());
  const [panelOpen, setPanelOpen] = useState(false);
  const [editingProgramId, setEditingProgramId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<FuelProgramCurrentRate | null>(null);

  const openCreate = () => {
    setEditingProgramId(null);
    setPanelOpen(true);
  };

  const openEdit = (programId: string) => {
    setEditingProgramId(programId);
    setPanelOpen(true);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Programs apply automatically to shipments of customers assigned to them — this week&apos;s
          computed rate is shown on each card
        </p>
        <Button type="button" size="sm" onClick={openCreate} className="gap-1.5">
          <Plus className="size-3.5" />
          New Program
        </Button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <Skeleton key={index} className="h-40" />
          ))}
        </div>
      ) : !entries || entries.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
          <div className="flex size-12 items-center justify-center rounded-full bg-muted">
            <Fuel className="size-5 text-muted-foreground" />
          </div>
          <p className="mt-3 text-sm font-medium">No fuel surcharge programs</p>
          <p className="mt-1 max-w-md text-xs text-muted-foreground">
            Create a program with a peg price and increment, assign it to customers from their
            billing profile, and fuel surcharges apply to shipments automatically.
          </p>
          <Button type="button" size="sm" onClick={openCreate} className="mt-4 gap-1.5">
            <Plus className="size-3.5" />
            Create Program
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {entries.map((entry) => (
            <ProgramCard
              key={entry.program.id}
              entry={entry}
              onEdit={() => openEdit(entry.program.id)}
              onDelete={() => setDeleteTarget(entry)}
            />
          ))}
        </div>
      )}

      <ProgramPanel open={panelOpen} onOpenChange={setPanelOpen} programId={editingProgramId} />

      <DeleteProgramDialog target={deleteTarget} onOpenChange={() => setDeleteTarget(null)} />
    </div>
  );
}

function ProgramCard({
  entry,
  onEdit,
  onDelete,
}: {
  entry: FuelProgramCurrentRate;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const rate = currentRateDisplay(entry);
  const inactive = entry.program.status !== "Active";

  return (
    <div
      className={cn(
        "group relative flex cursor-pointer flex-col rounded-lg border bg-card p-4 transition-colors hover:bg-muted/40",
        inactive && "opacity-60",
      )}
      onClick={onEdit}
      role="button"
      tabIndex={0}
      onKeyDown={(event) => {
        if (event.key === "Enter") onEdit();
      }}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{entry.program.name}</p>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            {entry.program.code}
            {entry.program.fuelIndex ? ` · ${entry.program.fuelIndex.code}` : ""}
            {entry.program.fuelIndex?.region ? ` · ${entry.program.fuelIndex.region}` : ""}
          </p>
        </div>
        <div className="flex items-center gap-1.5">
          {inactive && <Badge variant="outline">Inactive</Badge>}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={(event) => {
              event.stopPropagation();
              onDelete();
            }}
            className="size-7 p-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>

      <div className="mt-4 flex items-baseline gap-1.5">
        {rate ? (
          <>
            <span className="text-2xl font-semibold tabular-nums">{rate.value}</span>
            <span className="text-xs text-muted-foreground">{rate.unit}</span>
          </>
        ) : (
          <span className="text-sm text-muted-foreground">No rate for this week yet</span>
        )}
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        <Badge variant="secondary" className="text-2xs">
          {methodLabel(entry.program.method)}
        </Badge>
        {entry.price && (
          <Badge variant="outline" className="text-2xs tabular-nums">
            DOE ${Number(entry.price.price).toFixed(3)}
          </Badge>
        )}
        {entry.usedFallback && (
          <Badge
            variant="outline"
            className="gap-1 border-amber-500/50 text-2xs text-amber-600 dark:text-amber-400"
          >
            <AlertTriangle className="size-3" />
            Prior week price
          </Badge>
        )}
      </div>
    </div>
  );
}

function DeleteProgramDialog({
  target,
  onOpenChange,
}: {
  target: FuelProgramCurrentRate | null;
  onOpenChange: () => void;
}) {
  const queryClient = useQueryClient();

  const { mutate: remove, isPending } = useMutation({
    mutationFn: (id: string) => deleteFuelSurchargeProgram(id),
    onSuccess: () => {
      toast.success("Fuel surcharge program deleted");
      void queryClient.invalidateQueries({
        queryKey: queries.fuelSurcharge.currentRates().queryKey,
      });
      onOpenChange();
    },
    onError: () => {
      toast.error("Could not delete the program", {
        description: "Programs assigned to customer billing profiles must be unassigned first",
      });
    },
  });

  return (
    <AlertDialog open={!!target} onOpenChange={(open) => !open && onOpenChange()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {target?.program.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            Customers assigned to this program will stop receiving fuel surcharges on new shipments.
            Already-billed surcharges and their audit snapshots are preserved.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={isPending}
            onClick={(event) => {
              event.preventDefault();
              if (target) remove(target.program.id);
            }}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {isPending ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
