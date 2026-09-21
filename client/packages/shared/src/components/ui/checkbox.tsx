import { Checkbox as CheckboxPrimitive } from "@base-ui/react/checkbox";

import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, MinusIcon } from "lucide-react";

function Checkbox({ className, ...props }: CheckboxPrimitive.Root.Props) {
  const { indeterminate, checked, ...rest } = props;
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      className={cn(
        "ui-focus-ring peer relative flex size-4 shrink-0 items-center justify-center rounded-[calc(var(--radius-control)-2px)] border border-input bg-field transition-colors outline-none hover:border-border-strong group-has-disabled/field:opacity-50 after:absolute after:-inset-x-3 after:-inset-y-2 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-danger aria-invalid:[--ring:var(--ring-danger)] data-checked:border-brand data-checked:bg-brand data-checked:text-brand-foreground data-checked:hover:border-brand-hover data-checked:hover:bg-brand-hover data-indeterminate:border-brand data-indeterminate:bg-brand data-indeterminate:text-brand-foreground",
        className,
      )}
      indeterminate={indeterminate}
      checked={checked}
      {...rest}
    >
      <CheckboxPrimitive.Indicator
        data-slot="checkbox-indicator"
        className="grid animate-confirm place-content-center text-current transition-none [&>svg]:size-3.5 [&>svg]:stroke-[3]"
        render={indeterminate ? <MinusIcon /> : checked ? <CheckIcon /> : undefined}
      />
    </CheckboxPrimitive.Root>
  );
}

export { Checkbox };
