import { cva, type VariantProps } from "class-variance-authority";

export const holdButtonVariants = cva(
  "relative touch-none overflow-hidden transition-[border-color,scale] select-none data-holding:scale-[0.985]",
  {
    variants: {
      tone: {
        warning:
          "not-data-done:hover:border-warning data-done:border-warning-border data-done:text-warning-foreground",
        success:
          "not-data-done:hover:border-success data-done:border-success-border data-done:text-success",
      },
    },
    defaultVariants: {
      tone: "warning",
    },
  },
);

export const holdFillVariants = cva(
  "pointer-events-none absolute inset-0 origin-left transition-transform ease-linear",
  {
    variants: {
      tone: {
        warning: "bg-warning/28",
        success: "bg-success/25",
      },
    },
    defaultVariants: {
      tone: "warning",
    },
  },
);

export type HoldTone = NonNullable<VariantProps<typeof holdButtonVariants>["tone"]>;
