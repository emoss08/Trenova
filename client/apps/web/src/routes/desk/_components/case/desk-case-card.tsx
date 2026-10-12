import { caseStateOf } from "@/lib/case-state";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { DeskDock } from "@/components/desk-chat/desk-thread";
import type { AssistantThread, ChecklistItemState, StepAbility } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { Link } from "react-router";
import {
  caseRecordPath,
  caseRecordTitle,
  caseStateLabel,
  checklistItemDetail,
  checklistItemLabel,
  checklistTitle,
} from "./case-labels";
import { CaseStepAction, onlyPeopleCanTake } from "./case-step-action";
import type { DeskCase } from "./use-desk-case";

const NO_ABILITIES: Record<string, StepAbility> = {};

const ITEM_ICONS: Record<ChecklistItemState, DeskIconName> = {
  Done: "check",
  Blocked: "alert",
  Pending: "clock",
  NotNeeded: "check",
};

/**
 * The case above the composer: the record it is about, where it stands, and
 * what stands between the record and what comes next. Unchecked items block,
 * and the next step asks the agent to take it, in the person's own words.
 */
export function DeskCaseCard({
  thread,
  deskCase,
  dock,
  now,
  timezone,
}: {
  thread: AssistantThread;
  deskCase: DeskCase;
  dock: DeskDock;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const summary = deskCase.view?.summary ?? thread.case;
  if (!summary) {
    return null;
  }

  const checklist = deskCase.view?.checklist ?? null;
  const state = caseStateOf(summary, now);
  const settled = state === "Settled";
  const path = caseRecordPath(summary.record);
  const title = caseRecordTitle(summary.record, t);
  const counted =
    checklist?.items.filter((item) => item.state !== "NotNeeded" && !item.optional) ?? [];
  const done = counted.filter((item) => item.state === "Done").length;
  const next = !settled && checklist?.next ? checklist.next : "";
  const abilities = deskCase.view?.abilities ?? NO_ABILITIES;
  const peopleOnly = onlyPeopleCanTake(checklist, abilities);

  return (
    <section className={cn("dk-cs", `dk-cs-${state.toLowerCase()}`)} aria-label={t("Case")}>
      <div className="dk-cs-hd">
        <DeskIcon name="file" size={12} />
        {path ? (
          <Link className="dk-cs-rec" to={path}>
            {title}
          </Link>
        ) : (
          <span className="dk-cs-rec">{title}</span>
        )}
        <span className="dk-cs-st">{caseStateLabel(summary, now, timezone, t)}</span>
        <span className="dk-cs-sp" />
        {checklist && (
          <Button
            variant="bare"
            size="bare"
            className="h-6.5 gap-1.5 rounded-md px-2 text-xs text-dsk-subtle hover:bg-dsk-hover hover:text-dsk-fg [&_svg]:transition-transform [&_svg]:duration-240 [&_svg]:ease-(--dk-settle) motion-reduce:[&_svg]:transition-none aria-expanded:[&_svg]:rotate-180"
            aria-expanded={open}
            onClick={() => setOpen((value) => !value)}
          >
            {checklistTitle(checklist, t)}
            <b className="font-mono text-xs font-normal text-dsk-fg2">
              {t("{0} of {1}", done, counted.length)}
            </b>
            <DeskIcon name="down" size={11} />
          </Button>
        )}
        {next !== "" && (
          <CaseStepAction
            primary
            step={next}
            ability={abilities[next]}
            record={summary.record}
            checklist={checklist}
            thread={thread}
            dock={dock}
          />
        )}
      </div>
      {!settled && peopleOnly && (
        <p className="dk-cs-note">
          {t("No agent you can use can take this checklist's steps; each opens where you do it.")}
        </p>
      )}
      {deskCase.failed && (
        <p className="dk-cs-note">{t("The checklist could not be read. It will try again.")}</p>
      )}
      {checklist && (
        <div className={cn("dk-cs-body", open && "dk-open")} inert={!open}>
          <div className="dk-cs-clip">
            <ol className="dk-cs-list">
              {checklist.items.map((item) => {
                const detail = checklistItemDetail(item, timezone, t);
                return (
                  <li key={item.key} className={cn("dk-cs-it", `dk-cs-${item.state.toLowerCase()}`)}>
                    {item.manual ? (
                      <Button
                        variant="bare"
                        size="bare"
                        role="checkbox"
                        aria-checked={item.state === "Done"}
                        aria-label={checklistItemLabel(item, t)}
                        className="dk-cs-ic justify-center inset-ring-[1.5px] inset-ring-dsk-b-strong transition-[background-color,box-shadow] duration-140 enabled:hover:inset-ring-dsk-fg2 disabled:opacity-100 aria-checked:bg-dsk-success-sub aria-checked:text-dsk-success-fg aria-checked:inset-ring-0"
                        disabled={deskCase.busy || !thread.canContinue || settled}
                        onClick={() => deskCase.tick(item.key, item.state !== "Done")}
                      >
                        {item.state === "Done" && <DeskIcon name="check" size={12} stroke={2} />}
                      </Button>
                    ) : (
                      <span className="dk-cs-ic">
                        <DeskIcon name={ITEM_ICONS[item.state]} size={12} stroke={2} />
                      </span>
                    )}
                    <span className="dk-cs-lb">
                      {checklistItemLabel(item, t)}
                      {item.optional && <em className="dk-cs-opt">{t("Optional")}</em>}
                    </span>
                    {detail && <span className="dk-cs-dt">{detail}</span>}
                    {item.step && item.state === "Blocked" && !settled && item.step !== next && (
                      <CaseStepAction
                        primary={false}
                        step={item.step}
                        ability={abilities[item.step]}
                        record={summary.record}
                        checklist={checklist}
                        thread={thread}
                        dock={dock}
                      />
                    )}
                  </li>
                );
              })}
            </ol>
          </div>
        </div>
      )}
    </section>
  );
}
