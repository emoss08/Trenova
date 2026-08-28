import {
  ACTION_DOCK_SECONDARY_BUTTON,
  ActionDock,
  ActionDockIndicator,
} from "@/components/action-dock";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { LayoutGridIcon, PlusIcon } from "lucide-react";

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
    <ActionDock
      animated
      indicator={
        dirty ? (
          <ActionDockIndicator title="Unsaved layout" description="You have unsaved changes." />
        ) : (
          <ActionDockIndicator
            icon={
              <span className="bg-background/15 flex size-6 items-center justify-center rounded-full">
                <LayoutGridIcon className="text-background size-3.5" />
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
        title={
          addDisabled ? `This home screen has reached the ${maxWidgets}-widget limit.` : undefined
        }
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
  );
}
