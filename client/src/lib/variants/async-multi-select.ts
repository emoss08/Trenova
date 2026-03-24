import { cva } from "class-variance-authority";

export const multiSelectVariants = cva(
  "flex h-auto max-h-5 items-center justify-center gap-0.5 rounded-md border px-1 py-1.5 [&_svg]:mb-1 [&_svg]:size-1 [&_svg]:cursor-pointer [&_svg]:text-muted-foreground",
  {
    variants: {
      variant: {
        default: "border-foreground/10 bg-background text-primary",
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
  },
);
