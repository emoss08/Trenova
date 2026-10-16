import { cn } from "@trenova/shared/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";

const novaBadgeVariants = cva("inline-block shrink-0 nova-badge", {
  variants: {
    size: {
      xs: "size-3",
      sm: "size-3.5",
      md: "size-4",
      lg: "size-5",
      xl: "size-6",
    },
    spin: {
      none: "",
      hover: "group-hover:animate-nova-spin group-focus-visible:animate-nova-spin",
      always: "animate-nova-spin",
    },
  },
  defaultVariants: {
    size: "md",
    spin: "none",
  },
});

export type NovaBadgeSize = NonNullable<VariantProps<typeof novaBadgeVariants>["size"]>;

/**
 * Nova's badge: a disc with a hollow centre, swept with Nova's three colours
 * (`nova-badge` in tokens.css), so it reads as Nova's own and never as a tone.
 * Every place that names Nova, or shows what Nova wrote, read or is doing,
 * wears it: AI control, the setup guide, Triage, the Formula Studio, the
 * command palette and the briefs. Machine suggestions that are not Nova's keep
 * `AssistMark`.
 *
 * `size` runs from `xs` (12px, beside a line of small print) to `xl` (24px,
 * heading a pane); `lg` (20px) heads a sentence Nova speaks. `spin` runs the
 * colours round the badge, which itself stays still: "hover" while the pointer
 * or focus is on the nearest `group` ancestor, "always" while Nova is working.
 * Reduced motion keeps it still.
 */
export function NovaBadge({
  size,
  spin,
  className,
  label,
}: VariantProps<typeof novaBadgeVariants> & {
  className?: string;
  /** Names the badge for assistive technology; omitted, the badge is decoration. */
  label?: string;
}) {
  return (
    <span
      role={label === undefined ? undefined : "img"}
      aria-label={label}
      aria-hidden={label === undefined ? true : undefined}
      data-slot="nova-badge"
      className={cn(novaBadgeVariants({ size, spin }), className)}
    />
  );
}
