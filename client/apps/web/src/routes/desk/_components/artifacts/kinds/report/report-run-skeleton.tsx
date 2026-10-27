import { useT } from "@trenova/shared/i18n/use-t";

const ROW_WIDTHS = [92, 80, 71, 63, 55, 47];

/** The sheet while its rows are formatted: the header, the rows and the totals in shimmer. */
export function ReportSheetSkeleton() {
  const t = useT();
  return (
    <div className="dk-rr-sheet" aria-busy="true" aria-label={t("Loading the result…")}>
      <div className="dk-rr-sheet-h">
        <span className="ui-shimmer dk-rr-shim-line w-40" />
        <span className="ui-shimmer dk-rr-shim-line w-28" />
      </div>
      <div className="dk-rr-pic">
        <div className="dk-rr-bars">
          {ROW_WIDTHS.map((width) => (
            <div key={width} className="dk-rr-bar">
              <span className="ui-shimmer dk-rr-shim-line" />
              <span className="dk-rr-tr">
                <i className="ui-shimmer" style={{ width: `${width}%` }} />
              </span>
              <span className="ui-shimmer dk-rr-shim-line" />
              <span />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/**
 * A report run while its card is read: what was asked beside the run's
 * ticket, the headline figures, the sheet and the actions, each where the
 * card draws it.
 */
export function ReportRunSkeleton() {
  const t = useT();
  return (
    <div className="dkrr" aria-busy="true" aria-label={t("Loading the report run…")}>
      <div className="dk-rr">
        <div className="dk-rr-head">
          <div>
            <span className="ui-shimmer dk-rr-shim-line w-36" />
            <span className="ui-shimmer dk-rr-shim-line mt-2.5 w-4/5" />
            <div className="dk-rr-params">
              <span className="ui-shimmer dk-rr-shim-chip w-44" />
              <span className="ui-shimmer dk-rr-shim-chip w-32" />
            </div>
            <span className="ui-shimmer dk-rr-shim-line mt-3 w-52" />
          </div>
          <div className="dk-rr-stamp">
            <span className="ui-shimmer dk-rr-shim-line w-20" />
            <div className="dk-rr-steps">
              {[0, 1, 2, 3].map((step) => (
                <span key={step} className="ui-shimmer dk-rr-shim-line" />
              ))}
            </div>
          </div>
        </div>
        <div className="dk-rr-kpis">
          {[0, 1, 2, 3].map((kpi) => (
            <div key={kpi} className="dk-rr-kpi">
              <span className="ui-shimmer dk-rr-shim-line w-3/5" />
              <span className="ui-shimmer dk-rr-shim-figure w-4/5" />
            </div>
          ))}
        </div>
        <ReportSheetSkeleton />
        <div className="dk-rr-acts">
          <span className="ui-shimmer dk-rr-shim-chip h-8 w-32" />
          <span className="ui-shimmer dk-rr-shim-chip h-8 w-24" />
        </div>
      </div>
    </div>
  );
}
