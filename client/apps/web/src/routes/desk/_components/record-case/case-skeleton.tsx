import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CASE_TRANSITION_CLASS } from "./case-subject";

/** The case before it is read: its head, the finding, the plan and the exhibits, in place. */
export function CaseSkeleton() {
  const t = useT();

  return (
    <div
      className={cn("dk-cz-scroll", CASE_TRANSITION_CLASS)}
      aria-busy
      aria-label={t("Reading the case")}
    >
      <div className="dk-cz">
        <div className="dk-cz-main">
          <div className="dk-cz-sk-head">
            <i className="dk-sk" />
            <i className="dk-sk" />
            <i className="dk-sk" />
          </div>
          <i className="dk-sk dk-cz-sk-block" />
          <i className="dk-sk dk-cz-sk-plan" />
        </div>
        <div className="dk-cz-side">
          <i className="dk-sk dk-cz-sk-exh" />
          <i className="dk-sk dk-cz-sk-exh" />
          <i className="dk-sk dk-cz-sk-exh" />
        </div>
      </div>
    </div>
  );
}
