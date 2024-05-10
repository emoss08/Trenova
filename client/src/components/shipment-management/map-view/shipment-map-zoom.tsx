import { Button } from "@/components/ui/button";
import { GoogleMap } from "@google";
import { MinusIcon, PlusIcon } from "lucide-react";

export function ShipmentMapZoom({ map }: { map: GoogleMap }) {
  if (!map) return null;

  return (
    <div className="flex flex-col space-y-2">
      <Button size="icon" onClick={() => map.setZoom(map.getZoom() + 1)}>
        <PlusIcon />
      </Button>
      <Button size="icon" onClick={() => map.setZoom(map.getZoom() - 1)}>
        <MinusIcon />
      </Button>
    </div>
  );
}
