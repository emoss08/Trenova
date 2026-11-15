import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, ReactNode } from "react";
import { useStudioNova } from "./use-studio-nova";

type NovaButtonProps = Omit<ComponentProps<typeof Button>, "variant" | "children"> & {
  children: ReactNode;
  /** Nova is working on this button's request: the ring turns until it answers. */
  busy?: boolean;
};

/**
 * A request Nova carries out: Ask Nova, Fix with Nova, Describe, Suggest
 * edge cases, Rewrite. Triage's hand-off, set as an outline button: the ring
 * mark, and on hover Nova's spectrum lights the edge with a faint wash while
 * the ring turns once. Without Trenova AI on the plan there is no Nova to ask, and the
 * button is not drawn.
 */
export function NovaButton({
  children,
  busy = false,
  className,
  size = "xs",
  ...props
}: NovaButtonProps) {
  const nova = useStudioNova();
  if (!nova) return null;

  return (
    <Button
      type="button"
      variant="outline"
      size={size}
      className={cn("fs-nova-btn gap-1.5", className)}
      aria-busy={busy || undefined}
      {...props}
    >
      <span aria-hidden className="fs-nova-wash ui-spin-ring" />
      <span aria-hidden className="fs-nova-edge ui-spin-ring" />
      <NovaBadge size="xs" spin={busy ? "always" : "none"} className="fs-nova-badge" />
      {children}
    </Button>
  );
}
