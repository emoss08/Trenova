import type { PageAccess, PageAccessKind, PageAccessRequirement } from "@/lib/graphql/page-access";
import {
  PAGE_ACCESS_DECISION_LABELS,
  PAGE_ACCESS_USE_HELP,
  PAGE_ACCESS_USE_LABELS,
} from "./page-access-labels";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";

type Need = PageAccessRequirement & { source: string };

/**
 * Every permission a screen checks, once each: its route's first, then each
 * REST endpoint's, with the first endpoint that asks for it.
 */
export function screenNeeds(access: PageAccess): Need[] {
  const needs = new Map<string, Need>();
  for (const requirement of access.requires) {
    needs.set(`${requirement.resource}:${requirement.operation}`, {
      ...requirement,
      source: "",
    });
  }
  for (const endpoint of access.endpoints) {
    for (const requirement of endpoint.requires) {
      const key = `${requirement.resource}:${requirement.operation}`;
      if (!needs.has(key)) {
        needs.set(key, { ...requirement, source: `${endpoint.method} ${endpoint.path}` });
      }
    }
  }
  return [...needs.values()];
}

/**
 * What a screen needs and what its reader sees there: the permissions it
 * checks, each kind of record it shows and the linked fields that come back
 * as reference view only.
 */
export function PageAccessReport({ access }: { access: PageAccess }) {
  const t = useT();
  const needs = screenNeeds(access);
  const refused = needs.filter((need) => !need.allowed);

  return (
    <div className="flex flex-col gap-4" data-slot="page-access-report">
      {access.blocked ? (
        <Alert variant="warning" size="sm">
          <AlertTriangleIcon />
          <AlertDescription>
            {t(
              "This screen would not open, or would open empty: its route or the records it lists need a permission this reader does not hold.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-wrap gap-2">
        <Badge variant="neutral" appearance="outline">
          {t("{0, plural, one {# kind of record} other {# kinds of record}}", access.kinds.length)}
        </Badge>
        <Badge variant={access.withheld > 0 ? "warning" : "success"}>
          {t(
            "{0, plural, one {# linked field withheld} other {# linked fields withheld}}",
            access.withheld,
          )}
        </Badge>
        {refused.length > 0 ? (
          <Badge variant="danger">
            {t(
              "{0, plural, one {# permission missing} other {# permissions missing}}",
              refused.length,
            )}
          </Badge>
        ) : null}
      </div>

      <section aria-label={t("What this screen needs")} className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{t("What this screen needs")}</h3>
        {needs.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t("Its route and endpoints check no permission of their own.")}
          </p>
        ) : (
          <ul className="divide-border flex flex-col divide-y rounded-md border">
            {needs.map((need) => (
              <li
                key={`${need.resource}:${need.operation}`}
                className="flex items-center justify-between gap-3 px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm">
                    {t("{0} on {1}", need.operationName, need.resourceName)}
                  </p>
                  {need.source ? (
                    <p className="text-muted-foreground truncate font-mono text-2xs">
                      {need.source}
                    </p>
                  ) : null}
                </div>
                <Badge variant={need.allowed ? "success" : "danger"}>
                  {need.allowed ? t("Held") : t("Missing")}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-label={t("Records on this screen")} className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{t("Records on this screen")}</h3>
        {access.kinds.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t("The screen shows no kind of record the permission registry knows.")}
          </p>
        ) : (
          <ul className="divide-border flex flex-col divide-y rounded-md border">
            {access.kinds.map((kind) => (
              <PageAccessKindRow key={kind.resource} kind={kind} />
            ))}
          </ul>
        )}
      </section>

      {access.operations.length > 0 ? (
        <details className="group text-xs">
          <summary className="text-muted-foreground hover:text-foreground ui-focus-ring w-fit cursor-pointer rounded-sm">
            {t(
              "{0, plural, one {# GraphQL operation} other {# GraphQL operations}}",
              access.operations.length,
            )}
          </summary>
          <ul className="text-muted-foreground mt-2 flex flex-col gap-0.5 font-mono text-2xs">
            {access.operations.map((operation) => (
              <li key={operation}>{operation}</li>
            ))}
          </ul>
        </details>
      ) : null}
    </div>
  );
}

function PageAccessKindRow({ kind }: { kind: PageAccessKind }) {
  const t = useT();
  const withheld = kind.fields.filter((field) => field.decision !== "VISIBLE");

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2" data-kind={kind.resource}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm">{kind.resourceName}</span>
          <Badge variant="neutral" appearance="outline" title={PAGE_ACCESS_USE_HELP[kind.use]}>
            {PAGE_ACCESS_USE_LABELS[kind.use]}
          </Badge>
        </div>
        <Badge variant={kind.canRead ? "success" : kind.use === "READ" ? "danger" : "neutral"}>
          {kind.canRead ? t("Can read") : t("No read")}
        </Badge>
      </div>
      {withheld.length > 0 ? (
        <ul className="flex flex-wrap gap-1" aria-label={t("Withheld fields")}>
          {withheld.map((field) => (
            <li
              key={field.name}
              className="bg-warning-subtle text-warning-subtle-foreground border-warning-border rounded-badge border px-1.5 py-0.5 font-mono text-2xs"
              title={PAGE_ACCESS_DECISION_LABELS[field.decision]}
            >
              {field.name}
              <span className="sr-only"> {PAGE_ACCESS_DECISION_LABELS[field.decision]}</span>
            </li>
          ))}
        </ul>
      ) : kind.fields.length > 0 ? (
        <p className="text-muted-foreground text-xs">
          {t(
            "{0, plural, one {# linked field shown} other {# linked fields shown}}",
            kind.fields.length,
          )}
        </p>
      ) : null}
    </li>
  );
}
