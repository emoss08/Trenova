import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { formulaLibrary } from "@/lib/queries/formula-library";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";
import { useLibraryStudio } from "./library-studio-context";
import { libraryPath } from "./starter";
import { StatusLabel } from "./status-label";

export function LibraryUsagePaneSkeleton() {
  const t = useT();
  return (
    <div className="flex flex-col gap-2 p-3" aria-busy aria-label={t("Loading who imports it")}>
      {[0, 1, 2].map((row) => (
        <div key={row} className="flex items-center justify-between gap-3 py-1">
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-3.5 w-14" />
        </div>
      ))}
    </div>
  );
}

/**
 * Who imports the library: every template version that pins it, with the library version it
 * pinned. A template version keeps that version until a new version of the template pins
 * another, so this is also what an edit here will not change.
 */
export function LibraryUsagePane() {
  const t = useT();
  const { library } = useLibraryStudio();
  const libraryId = library?.id ?? "";
  const usages = useQuery({
    ...formulaLibrary.usages(libraryId),
    enabled: libraryId !== "",
    staleTime: 60_000,
  });

  if (!library) {
    return (
      <p className="text-foreground-muted p-4 text-sm">
        {t("Create the library, then templates can import it.")}
      </p>
    );
  }
  if (usages.isPending) return <LibraryUsagePaneSkeleton />;
  if (usages.isError) {
    return (
      <div className="flex flex-col items-start gap-2 p-4">
        <p className="text-foreground-muted text-sm">
          {t("The templates that import this library could not be loaded.")}
        </p>
        <Button type="button" variant="outline" size="xs" onClick={() => void usages.refetch()}>
          {t("Retry")}
        </Button>
      </div>
    );
  }
  if (usages.data.length === 0) {
    return (
      <div className="flex flex-col gap-1 p-4">
        <p className="text-sm font-medium">{t("No template imports it yet")}</p>
        <p className="text-foreground-muted text-xs">
          {t("A template imports it with use {0}.", libraryPath(library.name))}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <p className="text-foreground-muted border-b px-3 py-2 text-xs">
        {t(
          "{0, plural, one {# template version pins} other {# template versions pin}} a version of this library. Each keeps it until a new template version pins another.",
          usages.data.length,
        )}
      </p>
      <ul className="divide-y">
        {usages.data.map((usage) => (
          <li
            key={`${usage.templateId}:${usage.templateVersion}`}
            className="flex items-center gap-3 px-3 py-2"
          >
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <Link
                to={formulaTemplateRoutes.edit(usage.templateId)}
                className="ui-focus-ring truncate rounded-sm text-sm font-medium hover:underline"
              >
                {usage.templateName}
              </Link>
              <span className="text-foreground-muted flex items-center gap-2 text-xs">
                {t("v{0}", usage.templateVersion)}
                <StatusLabel status={usage.versionStatus} />
              </span>
            </div>
            <span
              className="text-foreground-muted shrink-0 font-mono text-xs"
              title={t("The library version this template version pinned")}
            >
              {t("pins v{0}", usage.libraryVersionNumber)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
