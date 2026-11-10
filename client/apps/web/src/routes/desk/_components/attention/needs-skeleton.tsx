import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 6;

/** Needs you while it is read: the list's head, its lens chips and rows, and an empty detail. */
export function NeedsSkeleton() {
  const t = useT();

  return (
    <div className="dk-w-nq" aria-busy aria-label={t("Reading what needs you")}>
      <div className="dk-w-nq-l">
        <div>
          <div className="dk-w-nq-h">
            <span className="dk-w-sk-line dk-w-sk-short ui-shimmer" />
          </div>
          <div className="dk-w-nq-bar" />
        </div>
        <div className="dk-w-lens">
          {Array.from({ length: 4 }, (_, index) => (
            <span key={index} className="dk-w-sk-chip ui-shimmer" />
          ))}
        </div>
        <div className="dk-w-nq-list">
          {Array.from({ length: ROWS }, (_, index) => (
            <span key={index} className="dk-w-sk-row ui-shimmer" />
          ))}
        </div>
      </div>
      <div className="dk-w-nq-d" />
    </div>
  );
}
