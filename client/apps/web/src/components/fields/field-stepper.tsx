import { ChevronDownIcon, ChevronUpIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The pair of buttons at the end of a number or money field that move its value up and down
 * a step. They stay out of the tab order's way no more than the field's own arrow keys do:
 * each is a real button, named for what it does.
 */
export function FieldStepper({
  disabled,
  onIncrement,
  onDecrement,
}: {
  disabled?: boolean;
  onIncrement: () => void;
  onDecrement: () => void;
}) {
  const t = useT();

  return (
    <div className="flex h-full flex-col items-stretch rounded-r-md border-l border-muted-foreground/20 bg-transparent">
      <button
        type="button"
        aria-label={t("Increment")}
        className="inline-flex h-7 w-6 flex-1 items-center justify-center border-b border-muted-foreground/20 text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
        disabled={disabled}
        onClick={onIncrement}
      >
        <ChevronUpIcon className="h-3 w-3" />
      </button>
      <button
        type="button"
        aria-label={t("Decrement")}
        className="inline-flex h-7 w-6 flex-1 items-center justify-center text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
        disabled={disabled}
        onClick={onDecrement}
      >
        <ChevronDownIcon className="h-3 w-3" />
      </button>
    </div>
  );
}
