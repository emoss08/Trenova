import { cn } from "@trenova/shared/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { useFormContext } from "react-hook-form";

export type GridCols = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;
export type ColSpan = GridCols | "full" | "auto";

const formGroupVariants = cva("grid gap-2", {
  variants: {
    cols: {
      1: "grid-cols-1",
      2: "grid-cols-1 md:grid-cols-2",
      3: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
      4: "grid-cols-1 md:grid-cols-2 lg:grid-cols-4",
    },
    dense: {
      true: "gap-2",
      false: "gap-x-4 gap-y-2",
    },
  },
  defaultVariants: {
    cols: 1,
    dense: false,
  },
});

interface FormProps extends React.FormHTMLAttributes<HTMLFormElement> {
  onSubmit?: (e: React.FormEvent<HTMLFormElement>) => void;
}

interface FormGroupProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof formGroupVariants> {
  children: React.ReactNode;
}

interface FormControlProps extends React.HTMLAttributes<HTMLDivElement> {
  cols?: ColSpan;
  error?: boolean;
  disabled?: boolean;
  children: React.ReactNode;
}

const colSpanClasses: Record<ColSpan, string> = {
  1: "col-span-1",
  2: "col-span-2",
  3: "col-span-3",
  4: "col-span-4",
  5: "col-span-5",
  6: "col-span-6",
  7: "col-span-7",
  8: "col-span-8",
  9: "col-span-9",
  10: "col-span-10",
  11: "col-span-11",
  12: "col-span-12",
  full: "col-span-full",
  auto: "col-auto",
} as const;

// FormRootError surfaces errors the server reported against something the form has no
// input for — a whole-object rejection, or a field this form does not render. Without
// somewhere to show them the user gets a save that failed for no stated reason.
//
// Rendered by Form automatically. Export exists for layouts that need it placed
// somewhere other than above the fields.
export const FormRootError = React.memo(({ className }: { className?: string }) => {
  const form = useFormContext();
  const message = form?.formState.errors.root?.message;

  if (!message) {
    return null;
  }

  return (
    <div
      role="alert"
      className={cn(
        "rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive",
        className,
      )}
    >
      {message}
    </div>
  );
});

FormRootError.displayName = "FormRootError";

export const Form = React.memo(({ className, onSubmit, children, ...props }: FormProps) => {
  const form = useFormContext();

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    // react-hook-form clears root inside handleSubmit, but only for submits that go
    // through it. Clearing here covers the ones that do not — a dialog wired straight to
    // mutate(), for instance — so a stale server complaint never outlives the attempt
    // that produced it.
    form?.clearErrors("root");
    onSubmit?.(e);
  };

  return (
    <form onSubmit={handleSubmit} className={className} {...props}>
      <FormRootError className="mb-4" />
      {children}
    </form>
  );
});

Form.displayName = "Form";

export const FormGroup = React.memo(
  ({ className, cols, dense, children, ...props }: FormGroupProps) => (
    <div className={cn(formGroupVariants({ cols, dense }), className)} role="group" {...props}>
      {children}
    </div>
  ),
);

FormGroup.displayName = "FormGroup";

export const FormControl = React.memo(
  ({ className, cols = 1, error, disabled, children, ...props }: FormControlProps) => {
    const colSpanClass = colSpanClasses[cols];

    return (
      <div
        data-slot="form-control"
        className={cn(
          "relative min-h-[4em]",
          colSpanClass,
          error && "has-error",
          disabled && "opacity-60",
          className,
        )}
        {...props}
      >
        {children}
      </div>
    );
  },
);

FormControl.displayName = "FormControl";

interface FormSectionProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string;
  titleCount?: number;
  description?: string;
  action?: React.ReactNode;
}

export const FormSection = React.memo(
  ({ className, title, titleCount, description, action, children, ...props }: FormSectionProps) => (
    <div
      className={cn("flex flex-col gap-2", className)}
      role="group"
      aria-labelledby={title ? `section-${title}` : undefined}
      {...props}
    >
      {(title || description || action) && (
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            {title && (
              <span className="flex items-center gap-1">
                <h3
                  id={`section-${title}`}
                  className="font-table text-base leading-none tracking-tight"
                >
                  {title}
                </h3>
                {titleCount && titleCount > 0 ? (
                  <span className="text-xs text-muted-foreground">({titleCount})</span>
                ) : null}
              </span>
            )}
            {description && <p className="text-xs text-muted-foreground">{description}</p>}
          </div>
          {action && <div className="shrink-0">{action}</div>}
        </div>
      )}
      {children}
    </div>
  ),
);

FormSection.displayName = "FormSection";
