/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { memo } from "react";

const AddMoveButton = memo(function AddMoveButton({
  onClick,
}: {
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="outline" size="xs" onClick={onClick}>
      <Icon icon={faPlus} className="size-4" />
      Add Move
    </Button>
  );
});

function MoveListHeaderDetails({ moves }: { moves: ShipmentSchema["moves"] }) {
  return (
    <div className="flex items-center gap-1">
      <h3 className="text-sm font-medium">Moves</h3>
      <span className="text-2xs text-muted-foreground">
        ({moves?.length ?? 0})
      </span>
    </div>
  );
}

function MoveListHeaderInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between">{children}</div>;
}

export function MoveListHeader({
  moves,
  handleAddMove,
}: {
  moves: ShipmentSchema["moves"];
  handleAddMove: () => void;
}) {
  return (
    <MoveListHeaderInner>
      <MoveListHeaderDetails moves={moves} />
      <AddMoveButton onClick={handleAddMove} />
    </MoveListHeaderInner>
  );
}
