import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { usePermissions } from "@/hooks/use-permissions";
import { holdTypeChoices } from "@/lib/choices";
import {
  ReleaseShipmentHoldRequestSchema,
  ShipmentHoldSchema,
} from "@/lib/schemas/shipment-hold-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { Resource } from "@/types/audit-entry";
import { Action } from "@/types/roles-permissions";
import {
  CheckIcon,
  Cross2Icon,
  DotsHorizontalIcon,
  EyeOpenIcon,
} from "@radix-ui/react-icons";
import { useMutation } from "@tanstack/react-query";
import { formatDistanceToNow, fromUnixTime } from "date-fns";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { UserHoverCard } from "../comment/user-hover-card";

export function HoldList({ holds }: { holds: ShipmentSchema["holds"] }) {
  if (holds?.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 p-2">
        <TooltipProvider>
          {holds?.map((hold) => <HoldRow key={hold.id} hold={hold} />)}
        </TooltipProvider>
      </div>
    </div>
  );
}

function HoldRow({ hold }: { hold: ShipmentHoldSchema }) {
  const { can } = usePermissions();
  const holdType = useMemo(() => {
    return holdTypeChoices.find((choice) => choice.value === hold.type);
  }, [hold.type]);
  const holdCounter = fromUnixTime(hold.startedAt ?? 0);

  const { mutateAsync } = useMutation({
    mutationFn: (values: ReleaseShipmentHoldRequestSchema) =>
      api.shipments.releaseHold(values),
    onSuccess: () => {
      toast.success("Shipment hold released successfully", {
        description: `The shipment hold has been released`,
      });

      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment"],
        options: {
          correlationId: `release-shipment-hold-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
  });

  const onReleaseHold = useCallback(async () => {
    await mutateAsync({
      holdId: hold.id || "",
      orgId: hold.organizationId,
      buId: hold.businessUnitId,
      userId: hold.createdBy?.id,
    });
  }, [
    mutateAsync,
    hold.id,
    hold.organizationId,
    hold.businessUnitId,
    hold.createdBy?.id,
  ]);

  const shipmentReleased = useMemo(
    () => hold.releasedAt !== null,
    [hold.releasedAt],
  );

  return (
    <div className="flex flex-col gap-2 bg-sidebar rounded-md p-2 border border-border">
      <div className="flex justify-between">
        <div className="flex flex-row gap-1 items-center">
          {hold.visibleToCustomer && (
            <Tooltip>
              <TooltipTrigger>
                <EyeOpenIcon className="size-4" />
              </TooltipTrigger>
              <TooltipContent>
                <p>Visible to customer</p>
              </TooltipContent>
            </Tooltip>
          )}
          <div className="text-sm font-medium">{holdType?.label}</div>
          <div className="text-sm text-muted-foreground">({hold.severity})</div>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="xs">
              <DotsHorizontalIcon />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              title="Release Hold"
              description={
                shipmentReleased
                  ? "Shipment hold is already released"
                  : "Release Hold to release blockages on this shipment"
              }
              onClick={onReleaseHold}
              disabled={
                !can(Resource.ShipmentHold, Action.Release) || shipmentReleased
              }
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="text-sm text-muted-foreground">{hold.notes}</div>
      <div className="grid grid-cols-3 gap-2 max-w-[300px]">
        <HoldRowGatingRow blocks={hold.blocksBilling} title="Billing" />
        <HoldRowGatingRow blocks={hold.blocksDelivery} title="Delivery" />
        <HoldRowGatingRow blocks={hold.blocksDispatch} title="Dispatch" />
      </div>
      <div className="flex flex-row gap-2 border-t border-border items-center justify-between p-0.5 shrink-0">
        <div className="flex flex-row gap-1 items-center text-sm text-muted-foreground">
          {!hold.releasedAt ? (
            <div className="flex flex-row gap-0.5 items-center">
              Started {formatDistanceToNow(holdCounter, { addSuffix: true })} by
              <UserHoverCard
                userId={hold.createdBy?.id}
                username={hold.createdBy?.username || ""}
              />
            </div>
          ) : (
            <div className="flex flex-row gap-0.5 items-center">
              Released by {hold.releasedBy?.name}
              <UserHoverCard
                userId={hold.releasedBy?.id}
                username={hold.releasedBy?.username || ""}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function HoldRowGatingRow({
  blocks,
  title,
}: {
  blocks: boolean;
  title: string;
}) {
  return (
    <div
      className={cn(
        "flex px-1 py-0.5 size-full text-nowrap rounded-md border text-green-700 bg-green-600/20 border-green-600/30 dark:text-green-400 text-xs font-medium shrink-0",
        blocks &&
          "text-red-700 bg-red-600/20 border-red-600/30 dark:text-red-400",
      )}
    >
      <div className="flex flex-row gap-x-0.5 items-center">
        {blocks ? <Cross2Icon /> : <CheckIcon />}
        <span>{title}</span>
      </div>
    </div>
  );
}
