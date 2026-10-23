import { PAGE_ACCESS_USE_LABELS } from "@/components/page-access/page-access-labels";
import { MAX_GUIDE_RESOURCES, type RoleResourceGuide } from "@/lib/graphql/page-access";
import { pageAccess } from "@/lib/queries/page-access";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueries } from "@tanstack/react-query";
import { Link } from "react-router";

/**
 * The role editor's sentences for a category's resources, asked of the
 * server a bounded batch at a time while the category is open. Each says
 * what granting and withholding the resource means for a role reaching
 * `maxSensitivity`, and which screens its grant changes.
 */
export function useRoleResourceGuides(
  resources: readonly string[],
  maxSensitivity: string,
  enabled: boolean,
) {
  const batches: string[][] = [];
  for (let start = 0; start < resources.length; start += MAX_GUIDE_RESOURCES) {
    batches.push(resources.slice(start, start + MAX_GUIDE_RESOURCES));
  }

  return useQueries({
    queries: batches.map((batch) => ({
      ...pageAccess.guides(batch, maxSensitivity),
      enabled,
      staleTime: 5 * 60_000,
    })),
    combine: (results) => {
      const guides = new Map<string, RoleResourceGuide>();
      for (const result of results) {
        for (const guide of result.data ?? []) {
          guides.set(guide.resource, guide);
        }
      }
      return { guides, isPending: results.some((result) => result.isPending) };
    },
  });
}

type ResourceGuideLineProps = {
  guide: RoleResourceGuide | undefined;
  /** Whether the role, as the editor holds it, reads the resource. */
  reads: boolean;
  isPending: boolean;
};

/**
 * What this row's grant means, in a sentence built from the registry and the
 * resource's reference view: what the role sees with its read, or without it
 * what other records still show of it. Then the screens the grant changes.
 */
export function ResourceGuideLine({ guide, reads, isPending }: ResourceGuideLineProps) {
  const t = useT();

  if (!guide) {
    return isPending ? (
      <Skeleton aria-label={t("Loading what this grant means")} className="mt-1 h-3 w-4/5" />
    ) : null;
  }

  return (
    <div className="mt-0.5 flex flex-col gap-0.5" data-slot="resource-guide">
      <p className="text-muted-foreground text-xs">{reads ? guide.granted : guide.withheld}</p>
      {!reads && guide.ceiling ? (
        <p className="text-warning-foreground text-xs">{guide.ceiling}</p>
      ) : null}
      {guide.screens.length > 0 ? <ResourceScreens guide={guide} /> : null}
    </div>
  );
}

function ResourceScreens({ guide }: { guide: RoleResourceGuide }) {
  const t = useT();

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="bare"
            size="bare"
            className="text-muted-foreground hover:text-foreground ui-focus-ring w-fit rounded-sm text-xs hover:underline"
          />
        }
      >
        {t(
          "{0, plural, one {# screen this affects} other {# screens this affects}}",
          guide.screens.length,
        )}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80 p-0">
        <div className="border-border border-b px-3 py-2">
          <p className="text-sm font-semibold">{t("Screens this affects")}</p>
          <p className="text-muted-foreground text-xs">
            {t("How each screen depends on {0}", guide.name)}
          </p>
        </div>
        <ul className="max-h-72 overflow-y-auto py-1" aria-label={t("Screens this affects")}>
          {guide.screens.map((screen) => {
            const name = screen.name ? translateLabel(screen.name) : screen.path;
            return (
              <li key={screen.path} className="flex items-center justify-between gap-2 px-3 py-1.5">
                <div className="min-w-0">
                  {screen.path.includes(":") ? (
                    <p className="truncate text-xs">{name}</p>
                  ) : (
                    <Link
                      to={screen.path}
                      className="ui-focus-ring block truncate rounded-sm text-xs hover:underline"
                    >
                      {name}
                    </Link>
                  )}
                  {screen.name ? (
                    <p className="text-muted-foreground truncate font-mono text-2xs">
                      {screen.path}
                    </p>
                  ) : null}
                </div>
                <Badge variant="neutral" appearance="outline">
                  {PAGE_ACCESS_USE_LABELS[screen.use]}
                </Badge>
              </li>
            );
          })}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
