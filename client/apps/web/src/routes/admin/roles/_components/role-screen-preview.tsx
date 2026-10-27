import { PageAccessReportSkeleton } from "@/components/page-access/page-access-panel-skeleton";
import { PageAccessReport } from "@/components/page-access/page-access-report";
import type { RoleDraft } from "@/lib/graphql/page-access";
import { pageAccess } from "@/lib/queries/page-access";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

/** How long the editor waits after a change before asking again. */
const DRAFT_SETTLE_MS = 400;

/**
 * Previews a screen as the role being edited: what a person holding only
 * this role (and the roles it inherits) would need there, and which linked
 * fields would come back as reference view only. It is worked out from the
 * form, before anything is saved, and reads no record.
 */
export function RoleScreenPreview({ draft }: { draft: RoleDraft }) {
  const t = useT();
  const [path, setPath] = useState<string | null>(null);
  const settled = useDebounce(draft, DRAFT_SETTLE_MS);

  const screens = useQuery({ ...pageAccess.screens(), staleTime: Infinity });
  const preview = useQuery({
    ...pageAccess.preview(path ?? "", { draft: settled }),
    enabled: path !== null,
    placeholderData: (previous) => previous,
  });

  const items = (screens.data ?? []).map((screen) => ({
    value: screen.path,
    label: screen.name ? translateLabel(screen.name) : screen.path,
  }));

  return (
    <div className="flex flex-col gap-4" data-slot="role-screen-preview">
      <div className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">{t("Screen")}</span>
        <Select
          value={path}
          onValueChange={(next) => setPath(typeof next === "string" ? next : null)}
          items={items}
          disabled={screens.isPending}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder={t("Choose a screen to preview")} />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                <span className="flex min-w-0 items-center gap-2">
                  <span className="truncate">{item.label}</span>
                  {item.label !== item.value ? (
                    <span className="text-muted-foreground truncate font-mono text-2xs">
                      {item.value}
                    </span>
                  ) : null}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {path === null ? (
        <p className="text-muted-foreground text-xs">
          {t("Choose a screen to see what this role would need and see there.")}
        </p>
      ) : preview.isPending ? (
        <PageAccessReportSkeleton />
      ) : preview.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{t("The preview could not be worked out.")}</AlertDescription>
        </Alert>
      ) : preview.data ? (
        <PageAccessReport access={preview.data} />
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("This location is not a screen the page access manifest describes.")}
        </p>
      )}
    </div>
  );
}
