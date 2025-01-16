import { cn } from "@/lib/utils";
import { badgeVariants } from "@/lib/variants/badge";
import { type VariantProps } from "class-variance-authority";
import * as React from "react";

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  withDot?: boolean;
}

function Badge({ className, variant, withDot = true, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props}>
      {withDot && (
        <svg
          className="size-1.5 fill-current"
          viewBox="0 0 6 6"
          aria-hidden="true"
        >
          <circle cx={3} cy={3} r={3} />
        </svg>
      )}
      {props.children}
    </div>
  );
}

export { Badge };
