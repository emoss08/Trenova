import { FieldWrapper } from "@/components/fields/field-components";
import { InfoPopover } from "@/components/info-popover";
import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn, toTitleCase } from "@trenova/shared/lib/utils";
import type { IdentityProviderFormValues } from "@trenova/shared/types/iam";
import { AlertTriangleIcon, Key01Icon, SearchLgIcon } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";
import { Controller, type Control } from "react-hook-form";
import { riskVariant } from "./utils";

type SecuritySectionProps = {
  title: string;
  icon?: ReactNode;
  /** Shown only when above zero, so an empty section does not announce "0". */
  count?: number;
  /** A short muted note at the end of the heading: a window, where a value comes from. */
  hint?: ReactNode;
  /** A link or a button at the far end of the heading. */
  action?: ReactNode;
  /** A plain-words note behind an info icon: what the section counts, what it leaves out. */
  help?: ReactNode;
  className?: string;
  children: ReactNode;
};

/**
 * A titled section of the security workspace, drawn open rather than in a card: a heading line,
 * then the section's rows between hairlines, so the page reads like the other briefings (AI
 * control, accounting) instead of a stack of boxes.
 */
export function SecuritySection({
  title,
  icon,
  count,
  hint,
  action,
  help,
  className,
  children,
}: SecuritySectionProps) {
  return (
    <section aria-label={title} className={cn("flex min-w-0 flex-col", className)}>
      <header className="flex min-h-8 items-center gap-2 pb-1.5">
        {icon ? (
          <span className="text-foreground-subtle shrink-0 [&>svg]:size-3.5">{icon}</span>
        ) : null}
        <h3 className="truncate text-sm font-semibold">{title}</h3>
        {help ? <InfoPopover title={title}>{help}</InfoPopover> : null}
        {count !== undefined && count > 0 ? (
          <span className="text-foreground-subtle font-mono text-xs tabular-nums">{count}</span>
        ) : null}
        <span className="flex-1" />
        {hint ? <span className="text-foreground-subtle truncate text-xs">{hint}</span> : null}
        {action}
      </header>
      <div className="border-border-subtle border-y">{children}</div>
    </section>
  );
}

/** The search box in a section's toolbar, so the list it filters sits right under it. */
export function PanelSearch({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
}) {
  return (
    <div className="w-full min-w-0 sm:ml-auto sm:w-56">
      <Input
        value={value}
        placeholder={placeholder}
        aria-label={placeholder}
        onChange={(event) => onChange(event.target.value)}
        leftElement={<SearchLgIcon className="text-muted-foreground size-3" />}
        className="h-7 text-xs"
      />
    </div>
  );
}

/** The row under a section's heading for what narrows its list: a view switch, a search. */
export function PanelToolbar({ children }: { children: ReactNode }) {
  return (
    <div className="border-border-subtle flex flex-wrap items-center gap-2 border-b py-2">
      {children}
    </div>
  );
}

export function ActivityItem({
  title,
  detail,
  badge,
  when,
}: {
  title: string;
  detail: string;
  badge: string;
  when: string;
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-2.5">
      <div className="min-w-0">
        <div className="truncate text-sm">{title}</div>
        <div className="text-muted-foreground truncate text-xs">{detail}</div>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1">
        <Badge variant={riskVariant(badge)}>{toTitleCase(badge)}</Badge>
        <span className="text-muted-foreground text-xs">{when}</span>
      </div>
    </div>
  );
}

export function ProviderLogo({ name }: { name: string }) {
  const lowerName = name.toLowerCase();
  if (
    lowerName.includes("entra") ||
    lowerName.includes("microsoft") ||
    lowerName.includes("azure")
  ) {
    return <EntraLogo className="size-5" />;
  }
  if (lowerName.includes("okta")) {
    return <OktaLogo className="h-5 w-auto" />;
  }
  return <Key01Icon className="text-primary size-5" />;
}

export function ChipArrayField({
  control,
  name,
  label,
  placeholder,
  description,
  parseValue,
  formatValue,
  required,
}: {
  control: Control<IdentityProviderFormValues>;
  name: "allowedDomains" | "oidcScopes";
  label: string;
  placeholder: string;
  description: string;
  parseValue: (value: string) => string[];
  formatValue: (value: string[]) => string;
  required?: boolean;
}) {
  return (
    <Controller
      name={name}
      control={control}
      rules={required ? { required: true } : undefined}
      render={({ field, fieldState }) => {
        const chips = Array.isArray(field.value) ? field.value : [];

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={required}
            error={fieldState.error?.message}
          >
            <div className="space-y-2">
              <Input
                value={formatValue(chips)}
                placeholder={placeholder}
                aria-invalid={fieldState.invalid}
                onChange={(event) => field.onChange(parseValue(event.target.value))}
              />
              {chips.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {chips.map((chip) => (
                    <span key={chip} className="bg-muted rounded-badge border px-2 py-0.5 text-xs">
                      {chip}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}

export function MetaLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 truncate">
      <span className="text-foreground font-medium">{label}:</span> {value}
    </div>
  );
}

/**
 * What a section says when it has nothing to list: a quiet line in the section's own rhythm,
 * not a box of its own.
 */
export function EmptyState({
  icon,
  label,
  description,
}: {
  icon?: ReactNode;
  label: string;
  description?: string;
}) {
  return (
    <div className="flex items-center gap-3 py-4">
      {icon ? (
        <span className="bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-md [&_svg]:size-4">
          {icon}
        </span>
      ) : null}
      <div className="min-w-0">
        <p className="text-sm font-medium">{label}</p>
        {description ? (
          <p className="text-muted-foreground text-xs text-pretty">{description}</p>
        ) : null}
      </div>
    </div>
  );
}

export function ErrorState({ label }: { label: string }) {
  return (
    <Alert variant="destructive" size="sm" className="my-3">
      <AlertTriangleIcon />
      <AlertDescription>{label}</AlertDescription>
    </Alert>
  );
}

/** Rows of a list still loading, drawn in the section that will hold them. */
export function RowSkeleton({ rows, label }: { rows: number; label: string }) {
  return (
    <div aria-busy aria-label={label} className="divide-border-subtle divide-y">
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} aria-hidden className="flex items-center gap-3 py-3">
          <Skeleton className="size-8 shrink-0 rounded-md" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-1/3" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  );
}
