import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "group/badge inline-flex h-6 w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-md border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-all focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 [&>svg]:pointer-events-none [&>svg]:size-3!",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/80",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        active: "border-green-600/30 bg-green-600/20 text-green-700 dark:text-green-400",
        inactive: "border-red-600/30 bg-red-600/20 text-red-700 dark:text-red-400",
        info: "border-blue-600/30 bg-blue-600/20 text-blue-700 dark:text-blue-400",
        purple: "border-purple-600/30 bg-purple-600/20 text-purple-700 dark:text-purple-400",
        orange: "border-orange-600/30 bg-orange-600/20 text-orange-700 dark:text-orange-400",
        indigo: "border-indigo-600/30 bg-indigo-600/20 text-indigo-700 dark:text-indigo-400",
        pink: "border-pink-600/30 bg-pink-600/20 text-pink-700 dark:text-pink-400",
        teal: "border-teal-600/30 bg-teal-600/20 text-teal-700 dark:text-teal-400",
        warning: "border-yellow-600/30 bg-yellow-600/20 text-yellow-700 dark:text-yellow-400",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type BadgeProps = useRender.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & {
    icon?: React.ReactNode;
  };

function Badge({ className, variant = "default", render, ...props }: BadgeProps) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ className, variant })),
      },
      props,
    ),
    render,
    state: {
      slot: "badge",
      variant,
    },
  });
}

export type BadgeVariant = NonNullable<VariantProps<typeof badgeVariants>["variant"]>;

export { Badge, badgeVariants };
