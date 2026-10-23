import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { GraphQLSelectOptionsConfig, SelectOption } from "@/lib/graphql/select-options";
import { pageAccess } from "@/lib/queries/page-access";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { SheetDescription, SheetHeader, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { PageAccessReportSkeleton } from "./page-access-panel-skeleton";
import { PageAccessReport } from "./page-access-report";

const roleOptions = { resource: "ROLE" } satisfies GraphQLSelectOptionsConfig;

type PageAccessPanelProps = {
  /** The location the panel describes: the page it was opened on. */
  location: string;
  /** Whether the reader may preview the screen as a role (role read). */
  canPreview: boolean;
};

/**
 * The page access panel: what the screen at a location needs, which kinds of
 * record the reader holds read on, and which linked fields come back as
 * reference view only. An administrator may view the screen as any role.
 */
export function PageAccessPanel({ location, canPreview }: PageAccessPanelProps) {
  const t = useT();
  const [roleId, setRoleId] = useState<string | null>(null);

  const callerQuery = useQuery({
    ...pageAccess.forCaller(location),
    enabled: roleId === null,
    staleTime: 30_000,
  });
  const previewQuery = useQuery({
    ...pageAccess.preview(location, { roleId: roleId ?? "" }),
    enabled: roleId !== null,
    staleTime: 30_000,
  });
  const query = roleId === null ? callerQuery : previewQuery;
  const access = query.data;

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-slot="page-access-panel">
      <SheetHeader className="border-border border-b pr-12">
        <SheetTitle>{t("Page access")}</SheetTitle>
        <SheetDescription className="truncate font-mono text-xs">
          {access?.path ?? location}
        </SheetDescription>
        {access?.name ? (
          <p className="text-muted-foreground truncate text-xs">{translateLabel(access.name)}</p>
        ) : null}
      </SheetHeader>
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
        {canPreview ? (
          <div className="flex flex-col gap-1.5">
            <span className="text-xs font-medium">{t("View as")}</span>
            <Autocomplete<SelectOption, Record<string, never>>
              graphql={roleOptions}
              value={roleId}
              onChange={(next: string | null) => setRoleId(next ?? null)}
              getOptionValue={(option) => option.id || ""}
              getDisplayValue={(option) => option.label || ""}
              renderOption={(option) => option.label || ""}
              placeholder={t("You, with your own roles")}
              clearable
            />
          </div>
        ) : null}
        {query.isPending ? (
          <PageAccessReportSkeleton />
        ) : query.isError ? (
          <Alert variant="destructive" size="sm">
            <AlertDescription>{t("Page access could not be loaded.")}</AlertDescription>
          </Alert>
        ) : access ? (
          <PageAccessReport access={access} />
        ) : (
          <p className="text-muted-foreground text-sm">
            {t("This location is not a screen the page access manifest describes.")}
          </p>
        )}
      </div>
    </div>
  );
}
