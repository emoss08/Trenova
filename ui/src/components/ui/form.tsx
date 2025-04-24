import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

// Types
export type GridCols = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;
export type ColSpan = GridCols | "full" | "auto";

// Variants for FormGroup layout
const formGroupVariants = cva(
  "grid gap-2", // base styles
  {
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
  },
);

// Form Interfaces
interface FormProps extends React.FormHTMLAttributes<HTMLFormElement> {
  onSubmit?: (e: React.FormEvent<HTMLFormElement>) => void;
}

interface FormGroupProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof formGroupVariants> {
  children: React.ReactNode;
}

interface FormControlProps extends React.HTMLAttributes<HTMLDivElement> {
  cols?: ColSpan;
  error?: boolean;
  disabled?: boolean;
}

// Column span mapping
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

// Form Components
export const Form = React.forwardRef<HTMLFormElement, FormProps>(
  ({ className, onSubmit, children, ...props }, ref) => {
    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      onSubmit?.(e);
    };

    return (
      <form ref={ref} onSubmit={handleSubmit} className={className} {...props}>
        {children}
      </form>
    );
  },
);

Form.displayName = "Form";

export const FormGroup = React.forwardRef<HTMLDivElement, FormGroupProps>(
  ({ className, cols, dense, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(formGroupVariants({ cols, dense }), className)}
      role="group"
      {...props}
    >
      {children}
    </div>
  ),
);

FormGroup.displayName = "FormGroup";

export const FormControl = React.forwardRef<HTMLDivElement, FormControlProps>(
  ({ className, cols = 1, error, disabled, children, ...props }, ref) => {
    const colSpanClass = colSpanClasses[cols];

    return (
      <div
        ref={ref}
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

// Helper Components
interface FormSectionProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string;
  description?: string;
}

export const FormSection = React.forwardRef<HTMLDivElement, FormSectionProps>(
  ({ className, title, description, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("mt-2 space-y-4", className)}
      role="group"
      aria-labelledby={title ? `section-${title}` : undefined}
      {...props}
    >
      {(title || description) && (
        <div className="space-y-1">
          {title && (
            <h3
              id={`section-${title}`}
              className="text-base font-semibold leading-none tracking-tight"
            >
              {title}
            </h3>
          )}
          {description && (
            <p className="text-xs text-muted-foreground">{description}</p>
          )}
        </div>
      )}
      {children}
    </div>
  ),
);

FormSection.displayName = "FormSection";
