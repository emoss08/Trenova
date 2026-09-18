import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { cva, type VariantProps } from "class-variance-authority";

import type {
  BadgeAccent,
  BadgeAppearance,
  BadgeTone,
  BadgeVariant,
} from "@trenova/shared/types/badge";
import { cn } from "@trenova/shared/lib/utils";

/* Variant selects a palette by pointing four local custom properties at a token
   ladder; appearance decides which of them get spent. Two independent lists of
   14 and 3 rather than 42 compound variants, and adding a tone stays one line.

   Variants were named by colour before — `purple`, `orange`, `teal`. Once a
   variant is called `purple` there is no correct answer to "what colour is a
   tender?", so every caller answered differently. Names state meaning now, and
   the categorical set is prefixed `accent-` to make it obvious at the call site
   that you are choosing an identity, not a severity. */
const badgeVariants = cva(
  "group/badge inline-flex h-5 w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-md border px-1.5 py-0 text-xs font-medium whitespace-nowrap transition-all ui-focus-ring has-data-[icon=inline-end]:pr-1 has-data-[icon=inline-start]:pl-1 aria-invalid:border-danger [&>svg]:pointer-events-none [&>svg]:size-3!",
  {
    variants: {
      variant: {
        neutral:
          "[--badge:var(--neutral)] [--badge-border:var(--neutral-border)] [--badge-on-subtle:var(--neutral-subtle-foreground)] [--badge-subtle:var(--neutral-subtle)]",
        brand:
          "[--badge:var(--brand)] [--badge-border:var(--brand-border)] [--badge-on-subtle:var(--brand-subtle-foreground)] [--badge-subtle:var(--brand-subtle)]",
        info: "[--badge:var(--info)] [--badge-border:var(--info-border)] [--badge-on-subtle:var(--info-subtle-foreground)] [--badge-subtle:var(--info-subtle)]",
        success:
          "[--badge:var(--success)] [--badge-border:var(--success-border)] [--badge-on-subtle:var(--success-subtle-foreground)] [--badge-subtle:var(--success-subtle)]",
        warning:
          "[--badge:var(--warning)] [--badge-border:var(--warning-border)] [--badge-on-subtle:var(--warning-subtle-foreground)] [--badge-subtle:var(--warning-subtle)]",
        danger:
          "[--badge:var(--danger)] [--badge-border:var(--danger-border)] [--badge-on-subtle:var(--danger-subtle-foreground)] [--badge-subtle:var(--danger-subtle)]",

        "accent-indigo":
          "[--badge:var(--accent-indigo)] [--badge-border:var(--accent-indigo-border)] [--badge-on-subtle:var(--accent-indigo-on-subtle)] [--badge-subtle:var(--accent-indigo-subtle)]",
        "accent-teal":
          "[--badge:var(--accent-teal)] [--badge-border:var(--accent-teal-border)] [--badge-on-subtle:var(--accent-teal-on-subtle)] [--badge-subtle:var(--accent-teal-subtle)]",
        "accent-amber":
          "[--badge:var(--accent-amber)] [--badge-border:var(--accent-amber-border)] [--badge-on-subtle:var(--accent-amber-on-subtle)] [--badge-subtle:var(--accent-amber-subtle)]",
        "accent-rose":
          "[--badge:var(--accent-rose)] [--badge-border:var(--accent-rose-border)] [--badge-on-subtle:var(--accent-rose-on-subtle)] [--badge-subtle:var(--accent-rose-subtle)]",
        "accent-emerald":
          "[--badge:var(--accent-emerald)] [--badge-border:var(--accent-emerald-border)] [--badge-on-subtle:var(--accent-emerald-on-subtle)] [--badge-subtle:var(--accent-emerald-subtle)]",
        "accent-sky":
          "[--badge:var(--accent-sky)] [--badge-border:var(--accent-sky-border)] [--badge-on-subtle:var(--accent-sky-on-subtle)] [--badge-subtle:var(--accent-sky-subtle)]",
        "accent-violet":
          "[--badge:var(--accent-violet)] [--badge-border:var(--accent-violet-border)] [--badge-on-subtle:var(--accent-violet-on-subtle)] [--badge-subtle:var(--accent-violet-subtle)]",
        "accent-slate":
          "[--badge:var(--accent-slate)] [--badge-border:var(--accent-slate-border)] [--badge-on-subtle:var(--accent-slate-on-subtle)] [--badge-subtle:var(--accent-slate-subtle)]",
      } satisfies Record<BadgeVariant, string>,

      appearance: {
        subtle: "border-(--badge-border) bg-(--badge-subtle) text-(--badge-on-subtle)",
        solid: "border-transparent bg-(--badge) text-foreground-on-solid",
        outline: "border-(--badge-border) bg-transparent text-(--badge-on-subtle)",
      } satisfies Record<BadgeAppearance, string>,
    },
    defaultVariants: {
      variant: "neutral",
      appearance: "subtle",
    },
  },
);

type BadgeProps = useRender.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & {
    icon?: React.ReactNode;
  };

function Badge({
  className,
  variant = "neutral",
  appearance = "subtle",
  render,
  ...props
}: BadgeProps) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ appearance, className, variant })),
      },
      props,
    ),
    render,
    state: {
      slot: "badge",
      appearance,
      variant,
    },
  });
}

export type { BadgeAccent, BadgeAppearance, BadgeTone, BadgeVariant };
export { Badge, badgeVariants };
