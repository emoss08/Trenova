import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ShipmentView } from "@/hooks/use-shipment-view";
import { faList, faMap } from "@fortawesome/pro-solid-svg-icons";

interface ViewToggleProps {
  currentView: ShipmentView;
  onViewChange: (view: ShipmentView) => void;
  isTransitioning: boolean;
}

export function ViewToggle({
  currentView,
  onViewChange,
  isTransitioning,
}: ViewToggleProps) {
  return (
    <div className="flex items-center gap-2 mb-4">
      <Button
        variant={currentView === "list" ? "default" : "outline"}
        size="sm"
        onClick={() => onViewChange("list")}
        disabled={isTransitioning}
      >
        <Icon icon={faList} className="h-4 w-4 mr-2" />
        List View
      </Button>
      <Button
        variant={currentView === "map" ? "default" : "outline"}
        size="sm"
        onClick={() => onViewChange("map")}
        disabled={isTransitioning}
      >
        <Icon icon={faMap} className="h-4 w-4 mr-2" />
        Map View
      </Button>
    </div>
  );
}
