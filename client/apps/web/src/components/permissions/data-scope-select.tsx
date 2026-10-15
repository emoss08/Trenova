import { dataScopeChoices } from "@/lib/choices";
import { scopeChoices } from "@/lib/data-scopes";
import type { ResourceDefinition } from "@/lib/role-api";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DataScope } from "@trenova/shared/types/role";
import { useMemo } from "react";

const scopeLabels = new Map<DataScope, string>(
  dataScopeChoices.map((choice) => [choice.value, choice.label]),
);

export function DataScopeSelect({
  resource,
  value,
  onValueChange,
  className,
}: {
  resource: ResourceDefinition;
  value: DataScope;
  onValueChange: (scope: DataScope) => void;
  className?: string;
}) {
  const t = useT();
  const choices = useMemo(
    () => scopeChoices(resource.enforceableScopes, value),
    [resource.enforceableScopes, value],
  );
  const items = useMemo(
    () =>
      choices.map((choice) => ({
        value: choice.value,
        label: scopeLabels.get(choice.value) ?? choice.value,
      })),
    [choices],
  );
  const current = choices.find((choice) => choice.value === value);
  const unenforced = current !== undefined && !current.enforced;

  return (
    <div className="flex items-center gap-1">
      <Select
        value={value}
        onValueChange={(next) => onValueChange(next as DataScope)}
        items={items}
      >
        <SelectTrigger className={cn("h-7 w-28 text-xs", className)}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {choices.map((choice) => (
            <SelectItem key={choice.value} value={choice.value} disabled={!choice.enforced}>
              {t(scopeLabels.get(choice.value) ?? choice.value)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {unenforced ? (
        <Tooltip>
          <TooltipTrigger
            render={
              <span
                className="text-warning flex size-5 items-center justify-center"
                aria-label={t("This scope is not enforced")}
              >
                <AlertTriangleIcon className="size-3.5" />
              </span>
            }
          />
          <TooltipContent className="max-w-64 text-xs">
            {t(
              "{0} is not enforced for {1} in this deployment, so this grant reaches the whole organization. Choose Organization to say so.",
              t(scopeLabels.get(value) ?? value),
              resource.displayName,
            )}
          </TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  );
}
