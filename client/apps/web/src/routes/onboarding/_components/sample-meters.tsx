import { SAMPLE_DATA_SET } from "@/lib/onboarding-form";
import { formatPlanMeterValue, planMeterLabel } from "@/lib/plan-meters";
import { useT } from "@trenova/shared/i18n/use-t";
import type { CSSProperties } from "react";

/** What "Load sample data" spends of each free demo limit, as meters. */
export function SampleMeters({ limits }: { limits: Readonly<Record<string, number>> }) {
  const t = useT();

  return (
    <>
      <span className="nv-meters">
        {SAMPLE_DATA_SET.map((row) => {
          const limit = limits[row.meter];
          const share = limit ? Math.min(100, (row.quantity / limit) * 100) : 0;
          return (
            <span key={row.meter} className="nv-mt">
              <span>
                {t(planMeterLabel(row.meter))}
                <span>
                  {limit
                    ? t(
                        "{0} of {1}",
                        formatPlanMeterValue(row.meter, row.quantity),
                        formatPlanMeterValue(row.meter, limit),
                      )
                    : formatPlanMeterValue(row.meter, row.quantity)}
                </span>
              </span>
              <i aria-hidden="true">
                <u style={{ "--nv-w": `${share}%` } as CSSProperties} />
              </i>
            </span>
          );
        })}
      </span>
      <span className="nv-lim">
        {t(
          "Sample records count toward the free demo's limits, the same as records you create. Deleting one frees its slot.",
        )}
      </span>
    </>
  );
}

/** The feature chips under an operation type: the parts of Trenova it leads with. */
export function ModuleChips({ modules }: { modules: readonly string[] }) {
  const t = useT();
  return (
    <span className="nv-mods">
      {modules.map((module) => (
        <span key={module}>{t(module)}</span>
      ))}
    </span>
  );
}
