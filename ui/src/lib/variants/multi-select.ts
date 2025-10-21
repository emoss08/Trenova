/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cva } from "class-variance-authority";

/**
 * Variants for the multi-select component to handle different styles.
 * Uses class-variance-authority (cva) to define different styles based on "variant" prop.
 */
export const multiSelectVariants = cva(
  "flex items-center justify-center gap-0.5 px-1 py-1.5 max-h-5 h-auto rounded-md border [&_svg]:size-1 [&_svg]:cursor-pointer [&_svg]:mb-1 [&_svg]:text-muted-foreground",
  {
    variants: {
      variant: {
        default: "border-foreground/10 text-primary bg-background",
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
