import { Button } from "@trenova/shared/components/ui/button";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, ReactNode } from "react";

type NovaButtonProps = Omit<ComponentProps<typeof Button>, "variant" | "children"> & {
  children: ReactNode;
  /** Nova is working on this button's request: the ring turns until it answers. */
  busy?: boolean;
};

/**
 * A request Nova carries out: Ask Nova, Fix with Nova, Describe, Suggest
 * edge cases, Rewrite. Triage's hand-off, set as an outline button: the ring
 * mark, and on hover Nova's spectrum lights the edge with a faint wash while
 * the ring turns once.
 */
export function NovaButton({
  children,
  busy = false,
  className,
  size = "xs",
  ...props
}: NovaButtonProps) {
  return (
    <Button
      type="button"
      variant="outline"
      size={size}
      className={cn("fs-nova-btn gap-1.5", className)}
      aria-busy={busy || undefined}
      {...props}
    >
      <NovaMark spin={busy ? "always" : "none"} className="fs-nova-ring size-3" />
      {children}
    </Button>
  );
}
