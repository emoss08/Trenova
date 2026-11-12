import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Activity, useId, useState } from "react";
import { PanelBody, PanelCard } from "./panel-parts";

type FindingPanelData = NonNullable<WatchtowerPanel["finding"]>;

/**
 * A finding: what to do about it first, and how it was worked out for whoever
 * wants to check, one press away.
 */
export function FindingPanel({ finding }: { finding: FindingPanelData }) {
  const t = useT();
  const detailId = useId();
  const [open, setOpen] = useState(false);
  const lead = finding.advice || finding.detail;
  const more = finding.advice !== "" && finding.detail !== "";

  return (
    <PanelCard
      eyebrow={t("Insight")}
      title={
        <span className="flex items-start gap-2">
          <AssistMark aria-hidden className="mt-0.5 size-3.5 flex-none text-dsk-subtle" />
          <span className="font-medium">{lead}</span>
        </span>
      }
    >
      {more && (
        <PanelBody>
          <button
            type="button"
            aria-expanded={open}
            aria-controls={detailId}
            onClick={() => setOpen((shown) => !shown)}
            className="inline-flex items-center gap-1 rounded-[4px] ui-focus-ring text-dsk-sm text-dsk-subtle hover:text-dsk-fg"
          >
            {open ? t("Hide how it was worked out") : t("How it was worked out")}
            <ChevronDownIcon
              aria-hidden
              className={cn("size-3 transition-transform duration-200", open && "rotate-180")}
            />
          </button>
          <Activity mode={open ? "visible" : "hidden"}>
            <p id={detailId} className="mt-2 animate-rise text-dsk-base text-dsk-fg2">
              {finding.detail}
            </p>
          </Activity>
        </PanelBody>
      )}
    </PanelCard>
  );
}
