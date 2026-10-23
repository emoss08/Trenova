import { staleTablePins, type FormulaTemplateTable } from "@/lib/graphql/formula-template-tables";
import { formulaTemplateTables } from "@/lib/queries/formula-template-tables";
import { recordPath } from "@/config/record-links";
import { useQuery } from "@tanstack/react-query";
import { TableIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TariffCheckResult, TariffContext } from "@trenova/shared/types/tariff";
import { Link } from "react-router";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

/** One rate table the source reads: what the check reads now and what the template pins. */
export type StudioTableRow = {
  code: string;
  name: string;
  /** The published version the live check reads: the newest one. Null when none is served. */
  reads: number | null;
  /** What the saved template pins, when it has been saved reading this table. */
  pin: FormulaTemplateTable | null;
};

/**
 * Every table the source reads or the saved template pins, by code: the live check's read with
 * the newest published version, beside the pin the last save or approval recorded.
 */
export function studioTableRows(
  check: Pick<TariffCheckResult, "tables">,
  context: Pick<TariffContext, "tables"> | null,
  pins: readonly FormulaTemplateTable[],
): StudioTableRow[] {
  const codes = new Set<string>([
    ...check.tables.map((table) => table.code),
    ...pins.map((pin) => pin.code),
  ]);
  return [...codes]
    .sort((a, b) => a.localeCompare(b))
    .map((code) => {
      const read = check.tables.find((table) => table.code === code);
      const shape = context?.tables.find((table) => table.code === code);
      return {
        code,
        name: shape?.name ?? "",
        reads: read?.version ?? shape?.version ?? null,
        pin: pins.find((pin) => pin.code === code) ?? null,
      };
    });
}

/** What the saved template pins for one table, and whether a newer version is out. */
export function TablePinState({ row }: { row: StudioTableRow }) {
  const t = useT();
  const { pin } = row;
  if (!pin) {
    return (
      <span className="text-foreground-muted text-xs">
        {t("Saving pins the version the check reads.")}
      </span>
    );
  }
  if (pin.archived) {
    return (
      <span className="flex items-center gap-1.5 text-xs">
        <Badge variant="danger" className="text-2xs px-1 py-0">
          {t("Archived")}
        </Badge>
        {t(
          "Pinned v{0}. Contracts keep reading it, but no new template version can read this table.",
          pin.versionNumber,
        )}
      </span>
    );
  }
  if (pin.latestVersionNumber > pin.versionNumber) {
    return (
      <span className="flex items-center gap-1.5 text-xs">
        <Badge variant="warning" className="text-2xs px-1 py-0">
          {t("Newer version")}
        </Badge>
        {t(
          "Pinned v{0}; v{1} is published. Contracts keep pricing with v{0} until a new template version is approved.",
          pin.versionNumber,
          pin.latestVersionNumber,
        )}
      </span>
    );
  }
  return (
    <span className="text-foreground-muted text-xs">
      {t("Pinned v{0}, the newest published version.", pin.versionNumber)}
    </span>
  );
}

/**
 * The rate tables the tariff reads. Tests and the preview read each table's newest published
 * version; a template version pins the version it was approved with, so publishing a matrix
 * version never reprices a contract until a new template version is approved.
 */
export function TablesSection() {
  const t = useT();
  const { check, context, template, mode } = useStudio();
  const templateId = template?.id ?? "";
  const pins = useQuery({
    ...formulaTemplateTables.get(templateId),
    enabled: mode === "edit" && templateId !== "",
  });
  const rows = studioTableRows(check, context, pins.data ?? []);
  const stale = staleTablePins(pins.data ?? []).length;
  if (rows.length === 0) return null;

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="tables"
        icon={TableIcon}
        title={t("Rate tables")}
        description={t(
          "Tests and the preview read each table's newest published version. Approval pins it for contracts.",
        )}
      >
        {stale > 0 && (
          <Badge variant="warning">
            {t(
              "{0, plural, one {# newer table version available} other {# newer table versions available}}",
              stale,
            )}
          </Badge>
        )}
      </SectionHead>
      <ul className="bg-card divide-y rounded-lg border">
        {rows.map((row) => (
          <li key={row.code} className="flex flex-col gap-1 px-3 py-2">
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-baseline gap-2">
                <span className="font-mono text-sm">{`tables.${row.code}`}</span>
                {row.name && (
                  <span className="text-foreground-muted truncate text-xs">{row.name}</span>
                )}
              </div>
              <span className="text-foreground-muted shrink-0 font-mono text-xs">
                {row.reads === null ? t("not published") : t("reads v{0}", row.reads)}
              </span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <TablePinState row={row} />
              {row.pin && (
                <Link
                  to={recordPath("rate_matrix", row.pin.rateMatrixId)}
                  className="ui-focus-ring text-foreground-muted shrink-0 rounded-sm text-xs hover:underline"
                >
                  {t("Open matrix")}
                </Link>
              )}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
