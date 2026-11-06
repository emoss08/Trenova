import { useT } from "@trenova/shared/i18n/use-t";
import { CapabilityGate } from "@/components/capability-gate";
import { usePermission } from "@/hooks/use-permission";
import {
  FlipBackwardIcon,
  HorizontalBarChart03Icon,
  Rows03Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { cn } from "@trenova/shared/lib/utils";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { preloadCenterView } from "./console-center-pane";
import type { DispatchActions } from "./use-dispatch-actions";
import { useDispatchView, type CenterMode } from "./url-state";

const CENTER_MODE_OPTIONS: readonly {
  id: CenterMode;
  label: string;
  Icon: typeof Rows03Icon;
}[] = [
  { id: "queue", label: "Queue", Icon: Rows03Icon },
  { id: "timeline", label: "Timeline", Icon: HorizontalBarChart03Icon },
];

/**
 * The page bar's actions: which view, Undo, and "Propose coverage". There is no ink button
 * in the page chrome; the decisions themselves are made on the queue.
 */
export function ConsolePageActions({
  actions,
  onPropose,
}: {
  actions: DispatchActions;
  onPropose: () => void;
}) {
  const t = useT();
  const { mode, setMode, timelineAvailable } = useDispatchView();
  const { allowed: mayAssign } = usePermission(Resource.ShipmentMove, Operation.Assign);

  return (
    <div className="flex items-center gap-2">
      {/* With the driver timeline withheld the queue is the only centre there is, so the
          switch goes rather than offering a choice of one. */}
      {timelineAvailable && (
        <div
          role="group"
          aria-label={t("View")}
          className="border-border inline-flex overflow-hidden rounded-md border"
        >
          {CENTER_MODE_OPTIONS.map((option, index) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setMode(option.id)}
              // Reaching for the switch is enough warning to fetch the view behind it.
              onPointerEnter={() => preloadCenterView(option.id)}
              onFocus={() => preloadCenterView(option.id)}
              aria-pressed={mode === option.id}
              className={cn(
                "ui-inset-focus-ring flex items-center gap-1 px-2 py-1 text-xs transition-colors",
                index > 0 && "border-border border-l",
                mode === option.id
                  ? "bg-muted text-foreground"
                  : "bg-background text-muted-foreground hover:text-foreground",
              )}
            >
              <option.Icon className="size-3" aria-hidden />
              {t(option.label)}
            </button>
          ))}
        </div>
      )}

      <span className="bg-border h-4 w-px" aria-hidden />

      <Button
        size="sm"
        variant="outline"
        disabled={!actions.canUndo || actions.isAssigning || actions.isDeciding}
        title={actions.undoLabel ?? t("Nothing to undo")}
        aria-label={actions.undoLabel ? t("Undo: {0}", actions.undoLabel) : t("Undo")}
        onClick={() => void actions.undo()}
        shortcut="U"
      >
        <FlipBackwardIcon className="size-3" aria-hidden />
        {t("Undo")}
      </Button>

      {/* Proposing coverage plans driver and tractor pairings, which the API refuses for
          an organization without asset operations or a person who may not assign. */}
      {mayAssign ? (
        <CapabilityGate capability={OrganizationCapability.AssetOperations}>
          <Button
            size="sm"
            variant="outline"
            className="group"
            disabled={actions.isPlanning}
            isLoading={actions.isPlanning}
            onClick={onPropose}
          >
            <NovaBadge size="sm" spin={actions.isPlanning ? "always" : "hover"} />
            {t("Propose coverage")}
          </Button>
        </CapabilityGate>
      ) : null}
    </div>
  );
}
