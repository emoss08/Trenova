import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 4;

/** The Scheduled page's list while it is read: its group heading and rows, in outline. */
export function DeskScheduledSkeleton() {
  const t = useT();

  return (
    <div className="dk-sd-g" aria-busy="true" aria-label={t("Loading scheduled requests")}>
      <span className="dk-sk dk-sd-sk-h" />
      <ul className="dk-sd-list">
        {Array.from({ length: ROWS }, (_, index) => (
          <li key={index} className="dk-sd-r">
            <span className="dk-sk dk-sd-sk-i" />
            <span className="dk-sd-t">
              <span className="dk-sk dk-sd-sk-p" />
              <span className="dk-sk dk-sd-sk-w" />
              <span className="dk-sk dk-sd-sk-m" />
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
