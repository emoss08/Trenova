import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  createFieldRegistry,
  FormFieldsProvider,
  useFormFieldRegistry,
} from "@trenova/shared/lib/form-field-registry";
import { refusalMessages } from "@trenova/shared/lib/form-refusal";
import { cn } from "@trenova/shared/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { useFormContext, useFormState, type Control } from "react-hook-form";

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

/**
 * What the server said when it refused the save: every message, whether it landed on a field
 * or on the form as a whole, in one alert at the head of the form, brought into view when it
 * appears. A long settings page is saved from a dock at the bottom of the window, so a refusal
 * shown only beside the fields it names, or only in a toast, leaves the person looking at a
 * form that did not save for no reason they can see.
 *
 * It follows the form's errors through `useFormState`. Reading `formState` off the form
 * object does not: that object is the same from one render to the next, so a memoized
 * component holding it is never drawn again when an error is set, and the alert stayed
 * empty.
 *
 * Rendered by Form automatically. Export exists for layouts that need it placed somewhere
 * other than above the fields.
 */
export function FormRefusal({ className }: { className?: string }) {
  const form = useFormContext();

  return form ? <FormRefusalAlert control={form.control} className={className} /> : null;
}

function FormRefusalAlert({ control, className }: { control: Control; className?: string }) {
  const t = useT();
  const { errors } = useFormState({ control });
  const messages = refusalMessages(errors);
  const shown = messages.join("\n");
  const alertRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (shown !== "") {
      alertRef.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }
  }, [shown]);

  if (messages.length === 0) {
    return null;
  }

  return (
    <Alert ref={alertRef} role="alert" variant="destructive" size="sm" className={className}>
      <AlertCircleIcon />
      <AlertTitle>{t("This change could not be saved")}</AlertTitle>
      <AlertDescription>
        {messages.length === 1 ? (
          messages[0]
        ) : (
          <ul className="list-disc pl-4">
            {messages.map((message) => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        )}
      </AlertDescription>
    </Alert>
  );
}

export const Form = React.memo(({ className, onSubmit, children, ...props }: FormProps) => {
  const form = useFormContext();
  // Every form keeps a registry of the fields it draws, so a change review or a save
  // dock can name them as the form does. A form panel provides one above its form,
  // to share with its footer; a form without one gets its own.
  const enclosing = useFormFieldRegistry();
  const [own] = React.useState(() => (enclosing ? null : createFieldRegistry()));
  const registry = enclosing ?? own;

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
    // Validation is react-hook-form's and zod's job, and native constraint validation
    // only gets in its way: a control that fails a browser constraint blocks submission
    // before React sees the event. The controls that trip it are usually invisible —
    // Base UI's NumberField, for one, mirrors its value into an aria-hidden input
    // carrying min/max/step — so the browser refuses a submit it cannot even focus a
    // control to explain ("An invalid form control with name='' is not focusable").
    <form onSubmit={handleSubmit} className={className} noValidate {...props}>
      <FormRefusal className="mb-4" />
      {registry && registry !== enclosing ? (
        <FormFieldsProvider registry={registry}>{children}</FormFieldsProvider>
      ) : (
        children
      )}
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
  description?: React.ReactNode;
  action?: React.ReactNode;
  /** Drawn just before the title, such as a mark that the section holds unsaved changes. */
  indicator?: React.ReactNode;
  /** Drawn just after the title, such as a control that explains the section. */
  titleAside?: React.ReactNode;
}

export const FormSection = React.memo(
  ({
    className,
    title,
    titleCount,
    description,
    action,
    indicator,
    titleAside,
    children,
    ...props
  }: FormSectionProps) => {
    // aria-labelledby is a list of ids split on spaces, so a title can't be the id: one with a
    // space, like "Billing & rating", would leave the group unnamed.
    const titleId = React.useId();

    return (
      <div
        className={cn("flex flex-col gap-2", className)}
        role="group"
        aria-labelledby={title ? titleId : undefined}
        {...props}
      >
        {(title || description || action) && (
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-1">
              {title && (
                <span className="flex items-center gap-1">
                  {indicator}
                  <h3 id={titleId} className="text-base leading-none font-semibold">
                    {title}
                  </h3>
                  {titleAside}
                  {titleCount && titleCount > 0 ? (
                    <span className="text-xs text-muted-foreground">({titleCount})</span>
                  ) : null}
                </span>
              )}
              {description && (
                <p className="text-foreground-muted text-sm text-pretty">{description}</p>
              )}
            </div>
            {action && <div className="shrink-0">{action}</div>}
          </div>
        )}
        {children}
      </div>
    );
  },
);

FormSection.displayName = "FormSection";
