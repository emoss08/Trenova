import { Switch as SwitchPrimitive } from "@base-ui/react/switch";

import { cn } from "@trenova/shared/lib/utils";

function Switch({
  className,
  size = "default",
  ...props
}: SwitchPrimitive.Root.Props & {
  size?: "sm" | "default";
}) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      data-size={size}
      className={cn(
        "ui-focus-ring data-checked:bg-brand data-unchecked:bg-border-strong data-unchecked:hover:bg-foreground-subtle/60 data-checked:hover:bg-brand-hover",
        "aria-invalid:border-danger aria-invalid:[--ring:var(--ring-danger)]",
        "shrink-0 rounded-full border border-transparent",
        "data-[size=default]:h-[18.4px] data-[size=default]:w-[32px]",
        "peer group/switch relative inline-flex items-center transition-colors outline-none after:absolute after:-inset-x-3 after:-inset-y-2 data-[size=sm]:h-[14px] data-[size=sm]:w-[24px] data-disabled:cursor-not-allowed data-disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="pointer-events-none block rounded-full bg-card shadow-raised ring-0 transition-transform duration-200 ease-spring group-active/switch:scale-x-110 group-data-[size=default]/switch:size-4 group-data-[size=sm]/switch:size-3 group-data-[size=default]/switch:data-checked:translate-x-[calc(100%-2px)] group-data-[size=sm]/switch:data-checked:translate-x-[calc(100%-2px)] dark:data-checked:bg-ink group-data-[size=default]/switch:data-unchecked:translate-x-0 group-data-[size=sm]/switch:data-unchecked:translate-x-0 dark:data-unchecked:bg-foreground"
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
