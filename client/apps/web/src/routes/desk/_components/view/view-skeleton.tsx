import { useT } from "@trenova/shared/i18n/use-t";
import "./view.css";

/** The composed view while it is read: the question's line, then the title and the body's blocks. */
export function DeskViewSkeleton() {
  const t = useT();
  return (
    <div className="dk-vw-scroll" aria-busy aria-label={t("Opening the view")}>
      <div className="dk-vw">
        <div className="dk-vw-sk">
          <span className="ui-shimmer" />
          <span className="ui-shimmer" />
          <span className="ui-shimmer" />
        </div>
      </div>
    </div>
  );
}
