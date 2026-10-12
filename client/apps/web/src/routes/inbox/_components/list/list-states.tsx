import type { InboundMailbox } from "@/lib/graphql/inbox";
import { AlertTriangleIcon, Copy01Icon, Inbox01Icon, Send01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { mailboxName } from "../folder-labels";
import { useTriageActions } from "../use-triage-actions";

const SKELETON_ROWS = [0, 1, 2, 3, 4, 5];

/** The folder loading: rows drawn as shimmering lines where messages will be. */
export function ListSkeleton() {
  const t = useT();

  return (
    <div aria-busy="true" aria-label={t("Loading messages")} className="flex flex-col gap-0.5 px-1.5 py-2">
      {SKELETON_ROWS.map((index) => (
        <div key={index} className="flex flex-col gap-2 p-3">
          <Skeleton className="h-2.5" style={{ width: 90 + ((index * 37) % 70) }} />
          <Skeleton className="h-2.5 w-4/5" />
          <Skeleton className="h-2.5" style={{ width: 120 + ((index * 53) % 90) }} />
        </div>
      ))}
    </div>
  );
}

/** A quiet sentence where the rows would be, with what to do next. */
export function ListEmpty({
  icon,
  title,
  description,
  children,
}: {
  icon?: ReactNode;
  title: string;
  description: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="text-foreground-subtle flex flex-col items-start gap-1.5 px-5 py-10 text-sm leading-relaxed">
      {icon !== undefined && (
        <span className="bg-surface-hover text-foreground-muted mb-1 flex size-10 items-center justify-center rounded-lg">
          {icon}
        </span>
      )}
      <p className="text-foreground text-base font-semibold">{title}</p>
      <div>{description}</div>
      {children !== undefined && <div className="mt-2 flex flex-col items-start gap-2">{children}</div>}
    </div>
  );
}

/** No source connected yet: the three steps to the first one. */
export function ConnectFirstSource({ onAdd }: { onAdd: () => void }) {
  const t = useT();
  const steps = [t("Pick an address to watch"), t("Forward it to Trenova"), t("Send a test email")];

  return (
    <ListEmpty
      icon={<Inbox01Icon className="size-4.5" />}
      title={t("Connect your first source")}
      description={t(
        "Start with a shared address — dispatch@, pod@, ap@ — and Nova reads everything that lands there. EDI, portals and texts plug in the same way.",
      )}
    >
      <ol className="flex flex-col gap-1.5">
        {steps.map((step, index) => (
          <li key={step} className="text-foreground-muted flex items-center gap-2 text-sm">
            <span className="border-border text-foreground-subtle flex size-4.5 items-center justify-center rounded-sm border font-mono text-2xs">
              {index + 1}
            </span>
            {step}
          </li>
        ))}
      </ol>
      <Button size="sm" onClick={onAdd}>
        {t("Add a source")}
      </Button>
    </ListEmpty>
  );
}

/** Sources exist but nothing has arrived yet. */
export function FirstMessage({ mailbox }: { mailbox: InboundMailbox }) {
  const t = useT();
  const actions = useTriageActions();

  return (
    <ListEmpty
      icon={<Send01Icon className="size-4" />}
      title={t("Waiting for your first message")}
      description={t("Forward anything to {0} and it shows up here, already read.", mailbox.address)}
    >
      <Button size="sm" variant="outline" onClick={() => actions.sendTestEmail(mailbox)}>
        {t("Send a test email")}
      </Button>
    </ListEmpty>
  );
}

const DAY_SECONDS = 86_400;

/**
 * A source whose mail has stopped: how long it has been quiet, and the two
 * things that tell whether it works — a test email, and the address to give
 * the sender again.
 */
export function SourceHealthBanner({ mailbox, now }: { mailbox: InboundMailbox; now: number }) {
  const t = useT();
  const actions = useTriageActions();
  if (mailbox.health !== "Quiet" && mailbox.health !== "NeverReceived") {
    return null;
  }

  const name = mailboxName(mailbox);
  const days =
    mailbox.lastReceivedAt === null ? 0 : Math.floor((now - mailbox.lastReceivedAt) / DAY_SECONDS);
  const title =
    mailbox.health === "NeverReceived" || mailbox.lastReceivedAt === null
      ? t("{0} hasn't received any mail yet", name)
      : t("{0} hasn't received mail in {1, plural, one {# day} other {# days}}", name, days);

  return (
    <div
      role="status"
      className="border-warning-border bg-warning-subtle animate-rise mx-3 mb-2 rounded-lg border px-3 py-2.5"
    >
      <div className="text-warning-subtle-foreground flex items-center gap-1.5 text-sm">
        <AlertTriangleIcon aria-hidden className="size-3.5" />
        <span className="font-semibold">{title}</span>
      </div>
      <p className="text-foreground-muted mt-1.5 text-sm leading-relaxed">
        {t("Ask the sender to forward to {0}, then send a test email to check it arrives.", mailbox.address)}
      </p>
      <div className="mt-2.5 flex gap-1">
        <Button size="xs" onClick={() => actions.sendTestEmail(mailbox)}>
          {t("Send a test email")}
        </Button>
        <Button size="xs" variant="ghost" onClick={() => actions.copyText(mailbox.address)}>
          <Copy01Icon aria-hidden className="size-3" />
          {t("Copy address")}
        </Button>
      </div>
    </div>
  );
}

/** Nova is off: messages arrive but nothing reads them. */
export function NovaOffBanner({
  onTurnOn,
  onOpenControl,
}: {
  onTurnOn: () => void;
  onOpenControl: () => void;
}) {
  const t = useT();

  return (
    <div
      role="status"
      className="border-border bg-sunken animate-rise mx-3 mb-2 rounded-lg border px-3 py-2.5"
    >
      <p className="text-foreground text-sm font-semibold">{t("Nova is off")}</p>
      <p className="text-foreground-muted mt-1.5 text-sm leading-relaxed">
        {t(
          "Messages still arrive, but nothing reads them — no kinds, no matches, no suggested actions. Everything waits for a person.",
        )}
      </p>
      <div className="mt-2.5 flex gap-1">
        <Button size="xs" onClick={onTurnOn}>
          {t("Turn on Nova")}
        </Button>
        <Button size="xs" variant="ghost" onClick={onOpenControl}>
          {t("AI control")}
        </Button>
      </div>
    </div>
  );
}
