import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Placeholders for the field components, drawn at each one's own size. They live apart from the
 * fields so a page can import them statically for its loading state without pulling in
 * react-hook-form, the controls and their popovers, which arrive with the form's own chunk.
 */

/** An `InputField`, `NumberField`, `SelectField` or autocomplete: label, control, description. */
export function InputFieldSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      <Skeleton className="h-4 w-37.5" />
      <Skeleton className="h-7 max-w-prose" />
      <Skeleton className="h-4 max-w-md" />
    </div>
  );
}

/** A `TextareaField`: label, a control three lines tall, description. */
export function TextareaFieldSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      <Skeleton className="h-4 w-37.5" />
      <Skeleton className="h-14 max-w-prose" />
      <Skeleton className="h-4 max-w-md" />
    </div>
  );
}

type TogglePosition = "left" | "right";

/** A `SwitchField`; `position` is the side the switch sits on, as the field's own prop. */
export function SwitchFieldSkeleton({ position = "left" }: { position?: TogglePosition }) {
  return (
    <div className="flex w-full items-start gap-2 rounded-md border border-transparent p-2.5">
      <Skeleton
        className={cn("h-4.5 w-8 shrink-0 rounded-full", position === "right" && "order-1")}
      />
      <div className="grid grow gap-1">
        <Skeleton className="h-4 w-37.5" />
        <Skeleton className="h-4 w-50" />
      </div>
    </div>
  );
}

/** A `CheckboxField`, or a bordered checkbox option when `outlined`. */
export function CheckboxFieldSkeleton({ outlined = false }: { outlined?: boolean }) {
  return (
    <div className={cn("flex w-full items-start gap-3 rounded-md p-2.5", outlined && "border p-3")}>
      <Skeleton className="mt-0.5 size-4 shrink-0 rounded-sm" />
      <div className="grid grow gap-1">
        <Skeleton className="h-4 w-28" />
        <Skeleton className="h-3.5 w-60" />
      </div>
    </div>
  );
}

/** An image upload: the square preview and the upload button under it. */
export function ImageFieldSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <Skeleton className="size-24" />
      <Skeleton className="h-8 w-28" />
    </div>
  );
}
