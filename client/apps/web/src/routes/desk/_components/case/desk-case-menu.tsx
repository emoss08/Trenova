import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { turnTime } from "@/components/desk-chat/turn-time";
import { caseStateOf } from "@/lib/case-state";
import type { AssistantThread, CaseParty } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { caseRecordTitle, caseStateLabel } from "./case-labels";
import { CaseRecordPicker } from "./case-record-picker";
import { snoozePresets, wallClockInputToUnix, type SnoozePresetKey } from "./case-snooze";
import { useDeskCase, type DeskCase } from "./use-desk-case";

type MenuView = "main" | "pick" | "time";

function presetLabel(key: SnoozePresetKey, t: TranslateFn): string {
  switch (key) {
    case "later":
      return t("In 3 hours");
    case "tomorrow":
      return t("Tomorrow morning");
    case "nextWeek":
      return t("Next week");
  }
}

function partyLabel(party: CaseParty, t: TranslateFn): string {
  return party.kind === "Customer"
    ? t("The customer, {0}", party.name)
    : t("The carrier, {0}", party.name);
}

/**
 * The top bar's case button. On an ordinary conversation it makes it a case
 * about a shipment, invoice or dispute; on a case it snoozes it until a time,
 * the next appointment or the ETA, parks it on a reply from the customer or a
 * carrier, wakes it, moves it to another record or makes it ordinary again.
 * The popover's content is mounted only while it is open, so nothing is read
 * until the person asks for it.
 */
export function DeskCaseMenu({ thread, timezone }: { thread: AssistantThread; timezone: string }) {
  const t = useT();
  const deskCase = useDeskCase(thread);
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<MenuView>("main");
  const isCase = thread.case !== undefined;

  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (!next) {
      setView("main");
    }
  };
  const run = (action: () => void) => {
    action();
    onOpenChange(false);
  };

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <DeskTip label={isCase ? t("Case") : t("Make this a case")}>
        <PopoverTrigger
          disabled={deskCase.busy || !thread.canContinue}
          render={
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(
                "size-7.5 rounded-lg text-dsk-subtle transition-colors duration-140 hover:bg-dsk-hover hover:text-dsk-fg",
                (open || isCase) && "bg-dsk-hover text-dsk-fg",
              )}
              aria-label={isCase ? t("Case") : t("Make this a case")}
            />
          }
        >
          <DeskIcon name="file" size={14} />
        </PopoverTrigger>
      </DeskTip>
      <PopoverContent
        align="end"
        sideOffset={6}
        className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
      >
        {open &&
          (view === "pick" || !isCase ? (
            <CaseRecordPicker
              heading={isCase ? t("Move the case to") : t("Make this a case about")}
              onPick={(subjectType, subjectId) => run(() => deskCase.bind(subjectType, subjectId))}
              onBack={isCase ? () => setView("main") : undefined}
            />
          ) : view === "time" ? (
            <SnoozeTimePicker
              timezone={timezone}
              onPick={(until) => run(() => deskCase.snooze("Time", until))}
              onBack={() => setView("main")}
            />
          ) : (
            <CaseActions
              thread={thread}
              deskCase={deskCase}
              timezone={timezone}
              onRun={run}
              onPickTime={() => setView("time")}
              onMove={() => setView("pick")}
            />
          ))}
      </PopoverContent>
    </Popover>
  );
}

function MenuGroup({ label }: { label: string }) {
  return <p className="text-muted-foreground px-2 pt-2 pb-1 text-xs font-medium">{label}</p>;
}

function MenuItem({
  icon,
  label,
  note,
  onClick,
}: {
  icon: DeskIconName;
  label: string;
  note?: ReactNode;
  onClick: () => void;
}) {
  return (
    <Button
      variant="bare"
      size="bare"
      role="menuitem"
      className="hover:bg-surface-hover flex w-full gap-2.5 rounded-md px-2 py-1.5 text-left"
      onClick={onClick}
    >
      <DeskIcon name={icon} size={13} />
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-sm">{label}</span>
        {note && <span className="text-muted-foreground truncate text-xs">{note}</span>}
      </span>
    </Button>
  );
}

function CaseActions({
  thread,
  deskCase,
  timezone,
  onRun,
  onPickTime,
  onMove,
}: {
  thread: AssistantThread;
  deskCase: DeskCase;
  timezone: string;
  onRun: (action: () => void) => void;
  onPickTime: () => void;
  onMove: () => void;
}) {
  const t = useT();
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const summary = thread.case;
  if (!summary) {
    return null;
  }

  const state = caseStateOf(summary, now);
  const settled = state === "Settled";
  const shipment = summary.record.type === "Shipment";
  const parties = deskCase.view?.parties ?? [];

  return (
    <div role="menu" className="flex max-h-[min(70vh,520px)] flex-col overflow-y-auto p-1">
      <div className="flex flex-col gap-0.5 px-2 pt-1.5 pb-1">
        <p className="truncate text-sm font-medium">{caseRecordTitle(summary.record, t)}</p>
        <p className="text-muted-foreground truncate text-xs">
          {caseStateLabel(summary, now, timezone, t)}
        </p>
      </div>
      {state === "Snoozed" && (
        <MenuItem icon="play" label={t("Wake it now")} onClick={() => onRun(deskCase.wake)} />
      )}
      {!settled && (
        <>
          <MenuGroup label={t("Snooze")} />
          {snoozePresets(now, timezone).map((preset) => (
            <MenuItem
              key={preset.key}
              icon="clock"
              label={presetLabel(preset.key, t)}
              note={turnTime(preset.until, timezone, t)}
              onClick={() => onRun(() => deskCase.snooze("Time", preset.until))}
            />
          ))}
          {shipment && (
            <>
              <MenuItem
                icon="truck"
                label={t("Until the next appointment")}
                note={t("Moves when the appointment does")}
                onClick={() => onRun(() => deskCase.snooze("Appointment"))}
              />
              <MenuItem
                icon="route"
                label={t("Until the ETA")}
                note={t("Follows the truck's estimate")}
                onClick={() => onRun(() => deskCase.snooze("ETA"))}
              />
            </>
          )}
          <MenuItem icon="clock" label={t("Pick a time…")} onClick={onPickTime} />
          {parties.length > 0 && (
            <>
              <MenuGroup label={t("Wait for a reply from")} />
              {parties.map((party) => (
                <MenuItem
                  key={`${party.kind}:${party.id}`}
                  icon={party.kind === "Customer" ? "headset" : "truck"}
                  label={partyLabel(party, t)}
                  note={t("Picks the case back up when they answer")}
                  onClick={() => onRun(() => deskCase.awaitReply(party))}
                />
              ))}
            </>
          )}
        </>
      )}
      <MenuGroup label={t("Case")} />
      <MenuItem icon="link" label={t("Move to another record")} onClick={onMove} />
      <MenuItem
        icon="x"
        label={t("Stop treating this as a case")}
        note={t("The conversation stays; it is no longer about the record")}
        onClick={() => onRun(deskCase.unbind)}
      />
    </div>
  );
}

function SnoozeTimePicker({
  timezone,
  onPick,
  onBack,
}: {
  timezone: string;
  onPick: (until: number) => void;
  onBack: () => void;
}) {
  const t = useT();
  const [value, setValue] = useState("");
  const [openedAt] = useState(() => Math.floor(Date.now() / 1000));
  const until = wallClockInputToUnix(value, timezone);
  const valid = until !== null && until > openedAt;

  return (
    <form
      className="flex flex-col gap-2 p-2"
      onSubmit={(event) => {
        event.preventDefault();
        if (valid && until !== null) {
          onPick(until);
        }
      }}
    >
      <div className="flex items-center gap-1">
        <Button size="icon-xs" variant="ghost" type="button" aria-label={t("Back")} onClick={onBack}>
          <DeskIcon name="chevL" size={13} />
        </Button>
        <p className="text-sm font-medium">{t("Snooze until")}</p>
      </div>
      <Input
        type="datetime-local"
        value={value}
        aria-label={t("Snooze until")}
        className="h-8 text-sm"
        onChange={(event) => setValue(event.target.value)}
      />
      <div className="flex justify-end">
        <Button type="submit" size="xs" disabled={!valid}>
          {t("Snooze")}
        </Button>
      </div>
    </form>
  );
}
