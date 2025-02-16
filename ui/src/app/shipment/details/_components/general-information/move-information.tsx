import { MoveStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { MoveSchema } from "@/lib/schemas/move-schema";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { MoveStatus } from "@/types/move";
import { StopStatus, StopType } from "@/types/stop";
import {
    faEllipsisVertical,
    faRoute,
    faTrash,
    faUser,
} from "@fortawesome/pro-regular-svg-icons";
import { useCallback, useEffect, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";

export function MoveInformation() {
  const { control } = useFormContext<ShipmentSchema>();
  const { fields, append, prepend, remove, swap, insert } = useFieldArray({
    control,
    name: "moves",
  });

  const addMove = useCallback(() => {
    const newMove: MoveSchema = {
      status: MoveStatus.New,
      sequence: fields.length,
      loaded: true,
      stops: [
        {
          status: StopStatus.New,
          sequence: 0,
          locationId: "",
          type: StopType.Pickup,
          plannedDeparture: 0,
          plannedArrival: 0,
          addressLine: "",
        },
        {
          status: StopStatus.New,
          sequence: 1,
          locationId: "",
          type: StopType.Delivery,
          plannedDeparture: 0,
          plannedArrival: 0,
          addressLine: "",
        },
      ],
    };
    append(newMove);
  }, [append, fields.length]);

  // If there are no moves, add a new move
  useEffect(() => {
    if (fields.length === 0) {
      addMove();
    }
  }, [addMove, fields.length]);

  return (
    <div className="flex flex-col gap-2">
      {fields.map((field, index) => (
        <MoveInformationItem key={field.id} move={field} index={index} />
      ))}
      <Button onClick={() => addMove()}>Add Move</Button>
    </div>
  );
}

function MoveInformationItem({
  move,
  index,
}: {
  move: MoveSchema;
  index: number;
}) {
  const [isOpen, setIsOpen] = useState(false);

  const { control } = useFormContext<ShipmentSchema>();
  const { remove } = useFieldArray({
    control,
    name: "moves",
  });

  const removeMove = useCallback(() => {
    remove(index);
  }, [index, remove]);

  return (
    <div className="flex flex-col gap-2 w-full bg-muted/50 rounded-md p-2">
      <div className="flex items-center justify-between">
        <MoveStatusBadge status={move.status} />
        <MoveActions />
      </div>
      <div className="flex flex-col gap-2">
        <div className="bg-background rounded-md p-2">
          THIS IS WHERE THE STOP INFORMATION GOES
        </div>
      </div>
    </div>
  );
}

function MoveActions() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon">
          <Icon icon={faEllipsisVertical} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          startContent={<Icon icon={faRoute} />}
          description="Split the move into two or more moves"
          title="Split Move"
        />
        <DropdownMenuItem
          startContent={<Icon icon={faUser} />}
          description="Assign the move to a driver"
          title="Assign"
        />
        <DropdownMenuItem
          startContent={<Icon icon={faTrash} />}
          description="Remove the move from the shipment"
          title="Remove Move"
          color="danger"
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
