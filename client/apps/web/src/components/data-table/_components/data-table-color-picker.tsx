import { ColorSwatchGrid } from "@/components/color-swatches";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";

/** A record's colour as the table draws it beside its name. */
export function ColorDot({ color, className }: { color: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn("size-2 shrink-0 rounded-full", className)}
      style={{ backgroundColor: color }}
    />
  );
}

/**
 * A record's colour, changed where it is drawn: the dot opens the swatches, and the one picked
 * is saved at once. A record with no colour shows where its dot would be while the row is
 * pointed at, so a colour can be given to it too.
 */
export function DataTableColorPicker({
  color,
  label,
  onChange,
}: {
  color: string | undefined;
  /** What the colour belongs to, for the button's name ("Code"). */
  label: string;
  onChange: (color: string | null) => Promise<void>;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  const pick = async (next: string | null) => {
    setOpen(false);
    if ((next ?? "").toLowerCase() === (color ?? "").toLowerCase()) return;
    setSaving(true);
    try {
      await onChange(next);
    } catch (error) {
      toast.error(t("Update failed"), {
        description:
          error instanceof Error ? error.message : translate("The change could not be saved."),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <button
            type="button"
            aria-label={t("Change the {0} color", label)}
            disabled={saving}
            onClick={(event) => event.stopPropagation()}
            onDoubleClick={(event) => event.stopPropagation()}
            className="group/dot ui-focus-ring -m-1 flex shrink-0 cursor-pointer items-center justify-center rounded-full p-1"
          />
        }
      >
        {saving ? (
          <Spinner className="size-2.5" />
        ) : color ? (
          <ColorDot color={color} />
        ) : (
          <span
            aria-hidden
            className="border-muted-foreground size-2 rounded-full border border-dashed opacity-0 group-hover/row:opacity-100 group-focus-visible/dot:opacity-100"
          />
        )}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-52 gap-2 p-2" onClick={(e) => e.stopPropagation()}>
        <ColorSwatchGrid value={color} onPick={(next) => void pick(next)} />
        {color ? (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="justify-start"
            onClick={() => void pick(null)}
          >
            {t("No color")}
          </Button>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
