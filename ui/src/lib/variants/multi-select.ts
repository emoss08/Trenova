import { cva } from "class-variance-authority";

/**
 * Variants for the multi-select component to handle different styles.
 * Uses class-variance-authority (cva) to define different styles based on "variant" prop.
 */
export const multiSelectVariants = cva("m-0.5", {
  variants: {
    variant: {
      default:
        "border-foreground/10 text-foreground bg-card hover:bg-foreground/5",
      secondary:
        "border-foreground/10 bg-secondary text-secondary-foreground hover:bg-secondary/80",
      destructive:
        "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
      inverted: "inverted",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});
