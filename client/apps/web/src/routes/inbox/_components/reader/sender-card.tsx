import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { InboundMessageDetail, InboundSenderRole } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { senderName } from "@/lib/triage/message-state";
import { useQuery } from "@tanstack/react-query";
import { ZapIcon } from "@trenova/shared/components/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatCompactAge, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { ruleActionLabel } from "../action-copy";
import { PersonAvatar } from "../person-avatar";
import { useTriageNavigation } from "../triage-navigation";
import { KindTag } from "./kind-tag";

function roleLabel(t: TranslateFn, role: InboundSenderRole): string {
  switch (role) {
    case "Customer":
      return t("Customer");
    case "Broker":
      return t("Broker");
    case "Carrier":
      return t("Carrier");
    case "Driver":
      return t("Driver");
    case "Unknown":
      return t("Unknown sender");
  }
}

function replyTime(t: TranslateFn, minutes: number): string {
  if (minutes < 60) {
    return t("{0}m", minutes);
  }
  if (minutes < 1440) {
    return t("{0}h", Math.round(minutes / 60));
  }

  return t("{0}d", Math.round(minutes / 1440));
}

function Chip({ children }: { children: string }) {
  return (
    <span className="bg-surface-hover text-foreground-muted inline-flex h-5.5 items-center rounded-full px-2 text-xs">
      {children}
    </span>
  );
}

function CardLabel({ children }: { children: string }) {
  return <div className="text-foreground-subtle mt-1 text-xs">{children}</div>;
}

function SenderCardBody({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const navigation = useTriageNavigation();
  const query = useQuery(inbox.senderProfile(message.id));
  const profile = query.data;
  const name = senderName(message);
  const now = useNowSeconds();

  return (
    <div className="flex w-75 flex-col gap-2 p-0.5">
      <div className="flex items-center gap-2.5">
        <PersonAvatar name={name} size="lg" />
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-base font-semibold">{name}</span>
          <span className="text-foreground-subtle truncate text-xs">{message.fromAddress}</span>
        </div>
      </div>
      {profile === undefined ? (
        query.isError ? (
          <p className="text-foreground-subtle text-xs">
            {t("Could not load what is known about this sender.")}
          </p>
        ) : (
          <div aria-busy="true" className="flex flex-col gap-2">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-4 w-40" />
          </div>
        )
      ) : (
        <>
          <div className="flex flex-wrap gap-1">
            <Chip>{roleLabel(t, profile.role)}</Chip>
            {profile.since !== null && (
              <Chip>
                {t("Since {0}", formatUnixInUserTimezone(profile.since, { year: "numeric" }, ""))}
              </Chip>
            )}
          </div>
          <div className="grid grid-cols-2 gap-1">
            <div className="bg-surface-hover flex flex-col rounded-md px-2.5 py-2">
              <span className="font-mono text-lg font-semibold tabular-nums">
                {profile.messagesLast90Days}
              </span>
              <span className="text-foreground-subtle text-2xs">{t("messages in 90 days")}</span>
            </div>
            <div className="bg-surface-hover flex flex-col rounded-md px-2.5 py-2">
              <span className="font-mono text-lg font-semibold tabular-nums">
                {profile.usualReplyMinutes === null ? "—" : replyTime(t, profile.usualReplyMinutes)}
              </span>
              <span className="text-foreground-subtle text-2xs">{t("your usual reply time")}</span>
            </div>
          </div>
          {profile.kinds.length > 0 && (
            <>
              <CardLabel>{t("Usually sends")}</CardLabel>
              <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs">
                {profile.kinds.map((kind) => (
                  <KindTag key={kind.classification} kind={kind.classification} />
                ))}
              </div>
            </>
          )}
          <CardLabel>{t("Rules for {0}", profile.domain)}</CardLabel>
          {profile.rules.length === 0 ? (
            <span className="text-foreground-subtle text-sm">{t("None yet")}</span>
          ) : (
            profile.rules.map((rule) => (
              <span key={rule.id} className="flex items-center gap-1.5 text-sm">
                <ZapIcon aria-hidden className="text-accent-violet size-3" />
                {ruleActionLabel(t, rule.action)}
              </span>
            ))
          )}
          {profile.recent.length > 0 && (
            <>
              <CardLabel>{t("Recent")}</CardLabel>
              <div className="-mx-1 flex flex-col">
                {profile.recent.map((sample) => (
                  <button
                    key={sample.messageId}
                    type="button"
                    className="hover:bg-surface-hover ui-focus-ring flex h-7 items-center gap-2 rounded-md px-1 text-left text-sm"
                    onClick={() => navigation.open(sample.messageId)}
                  >
                    <span className="min-w-0 flex-1 truncate">
                      {sample.subject === "" ? t("(no subject)") : sample.subject}
                    </span>
                    <span className="text-foreground-subtle font-mono text-xs tabular-nums">
                      {formatCompactAge(now - sample.receivedAt)}
                    </span>
                  </button>
                ))}
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}

/**
 * The sender's name in the reader, which opens what Trenova knows about them:
 * who they are to the organization, how much they send and how fast it is
 * usually answered, what kinds they send, the rules for their domain and
 * their recent mail.
 */
export function SenderName({ message }: { message: InboundMessageDetail }) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            className="hover:bg-surface-hover data-popup-open:bg-surface-hover ui-focus-ring -mx-0.75 rounded-sm px-0.75 font-medium whitespace-nowrap underline-offset-3 hover:underline data-popup-open:underline"
          />
        }
      >
        {senderName(message)}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-2">
        <SenderCardBody message={message} />
      </PopoverContent>
    </Popover>
  );
}
