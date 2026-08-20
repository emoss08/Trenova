import { FieldWrapper } from "@/components/fields/field-components";
import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { cn, toTitleCase } from "@trenova/shared/lib/utils";
import type { IdentityProviderFormValues } from "@trenova/shared/types/iam";
import { AlertTriangleIcon, KeyRoundIcon, SearchIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Controller, type Control } from "react-hook-form";
import { riskVariant } from "./utils";

export function StatusTile({
  icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: "active" | "warning" | "info" | "muted";
}) {
  return (
    <div className="bg-background rounded-lg border p-3 shadow-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="text-muted-foreground text-xs font-medium uppercase">{label}</div>
          <div className="truncate text-lg font-semibold tracking-tight">{value}</div>
          <div className="text-muted-foreground truncate text-xs">{detail}</div>
        </div>
        <span
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-md border [&_svg]:size-4",
            tone === "active" && "border-green-600/30 bg-green-600/10 text-green-700",
            tone === "warning" && "border-yellow-600/30 bg-yellow-600/10 text-yellow-700",
            tone === "info" && "border-blue-600/30 bg-blue-600/10 text-blue-700",
            tone === "muted" && "bg-muted text-muted-foreground",
          )}
        >
          {icon}
        </span>
      </div>
    </div>
  );
}

export function ConsoleToolbar({
  title,
  description,
  search,
  onSearchChange,
  searchPlaceholder,
  action,
}: {
  title: string;
  description: string;
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder: string;
  action?: ReactNode;
}) {
  return (
    <div className="bg-sidebar rounded-lg border p-3">
      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h3 className="text-base font-semibold tracking-tight">{title}</h3>
          <p className="text-muted-foreground text-sm">{description}</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <div className="relative min-w-0 sm:w-80">
            <Input
              value={search}
              placeholder={searchPlaceholder}
              onChange={(event) => onSearchChange(event.target.value)}
              leftElement={<SearchIcon className="text-muted-foreground size-3" />}
            />
          </div>
          {action}
        </div>
      </div>
    </div>
  );
}

export function PanelHeader({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 border-b p-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-2">
        <span className="bg-muted/40 text-muted-foreground flex size-8 items-center justify-center rounded-md border [&_svg]:size-4">
          {icon}
        </span>
        <div>
          <div className="text-sm font-medium">{title}</div>
          <div className="text-muted-foreground text-xs">{description}</div>
        </div>
      </div>
      {action}
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
    <div className="flex items-start justify-between gap-3 p-3">
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{title}</div>
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
  return <KeyRoundIcon className="text-primary size-5" />;
}

export function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <label className="bg-background flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm">
      <span className="min-w-0">
        <span className="block font-medium">{label}</span>
        {description && <span className="text-muted-foreground block text-xs">{description}</span>}
      </span>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </label>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="font-medium">{label}</span>
      {children}
    </label>
  );
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
                    <span key={chip} className="bg-muted rounded-full border px-2 py-0.5 text-xs">
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

export function EmptyState({
  icon,
  label,
  description,
  compact,
}: {
  icon?: ReactNode;
  label: string;
  description?: string;
  compact?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-lg border border-dashed text-center",
        compact ? "m-3 p-4" : "p-8",
      )}
    >
      {icon && (
        <span className="bg-muted text-muted-foreground mb-2 flex size-9 items-center justify-center rounded-md [&_svg]:size-4">
          {icon}
        </span>
      )}
      <div className="text-sm font-medium">{label}</div>
      {description && (
        <div className="text-muted-foreground mt-1 max-w-md text-sm">{description}</div>
      )}
    </div>
  );
}

export function ErrorState({ label, compact }: { label: string; compact?: boolean }) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-lg border border-red-600/30 bg-red-600/10 text-sm text-red-700 dark:text-red-400",
        compact ? "m-3 p-3" : "p-4",
      )}
    >
      <AlertTriangleIcon className="size-4" />
      {label}
    </div>
  );
}

export function RowSkeleton({ rows }: { rows: number }) {
  return (
    <div className="bg-background space-y-2 rounded-lg border p-3">
      {Array.from({ length: rows }).map((_, index) => (
        <Skeleton key={index} className="h-16 w-full rounded-md" />
      ))}
    </div>
  );
}

export function OverviewSkeleton() {
  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-24 rounded-lg" />
        ))}
      </div>
      <Skeleton className="h-32 rounded-lg" />
    </div>
  );
}
