import { XCloseIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { fieldClearOverUnitClass, fieldClearRevealClass } from "./field-clear-reveal";

/**
 * The control that empties a field, at its end while it holds a value and shown while the
 * field is hovered or focused. Every field that is not required has one, so a value
 * entered by mistake can always be taken back. It
 * is left out of the tab order: a keyboard already empties a typed field by deleting, and a
 * form's tab stops stay its fields.
 */
export function FieldClearButton({
  onClear,
  overUnit = false,
  className,
}: {
  onClear: () => void;
  /** The control shares its place with the field's unit, which gives way to it. */
  overUnit?: boolean;
  className?: string;
}) {
  const t = useT();

  return (
    <button
      type="button"
      tabIndex={-1}
      aria-label={t("Clear")}
      title={t("Clear")}
      onClick={onClear}
      className={cn(
        "flex size-5 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground hover:bg-surface-hover hover:text-foreground",
        overUnit ? fieldClearOverUnitClass : fieldClearRevealClass,
        className,
      )}
    >
      <XCloseIcon className="size-3" />
    </button>
  );
}
