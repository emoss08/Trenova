import {
  ACTION_DOCK_SECONDARY_BUTTON,
  ActionDock,
  ActionDockIndicator,
} from "@/components/action-dock";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { LayoutGridIcon, PlusIcon } from "lucide-react";
import { m } from "motion/react";

export type HomeEditDockProps = {
  /** True once the draft differs from what the server holds. */
  dirty: boolean;
  saving: boolean;
  saveLabel?: string;
  /** Set when the canvas is full, so the add button says why it is disabled. */
  addDisabled: boolean;
  maxWidgets: number;
  onAdd: () => void;
  onDiscard: () => void;
  onSave: () => void;
};

/**
 * The floating dock that carries every edit-mode action for the home canvas.
 * It floats rather than sitting above the grid so the actions stay in reach no
 * matter how tall the layout being arranged is.
 */
export function HomeEditDock({
  dirty,
  saving,
  saveLabel = "Save",
  addDisabled,
  maxWidgets,
  onAdd,
  onDiscard,
  onSave,
}: HomeEditDockProps) {
  return (
    <m.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
    >
      <ActionDock
        indicator={
          dirty ? (
            <ActionDockIndicator
              title="Unsaved layout"
              description="You have unsaved changes."
            />
          ) : (
            <ActionDockIndicator
              icon={
                <span className="flex size-6 items-center justify-center rounded-full bg-background/15">
                  <LayoutGridIcon className="size-3.5 text-background" />
                </span>
              }
              title="Editing home screen"
              description="Drag to rearrange, resize from a widget's menu."
            />
          )
        }
      >
        <Button
          variant="outline"
          onClick={onAdd}
          disabled={addDisabled || saving}
          title={addDisabled ? `This home screen has reached the ${maxWidgets}-widget limit.` : undefined}
          className={ACTION_DOCK_SECONDARY_BUTTON}
        >
          <PlusIcon className="size-3.5" />
          Add widget
        </Button>
        <Button
          variant="outline"
          onClick={onDiscard}
          disabled={saving}
          className={ACTION_DOCK_SECONDARY_BUTTON}
        >
          Discard
        </Button>
        <Button variant="default" onClick={onSave} disabled={saving} className="min-w-16">
          {saving ? <Spinner /> : saveLabel}
        </Button>
      </ActionDock>
    </m.div>
  );
}
