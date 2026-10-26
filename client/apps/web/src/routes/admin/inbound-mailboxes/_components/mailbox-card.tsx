import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePendingActions } from "@/hooks/use-pending-actions";
import type { InboundMailbox } from "@/lib/graphql/inbox";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  CheckIcon,
  ChevronDownIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  Edit02Icon,
  InfoCircleIcon,
  Key01Icon,
  LockKeyholeCircleIcon,
  Mail01Icon,
  RefreshCw02Icon,
  Send01Icon,
  Shield01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Link } from "react-router";
import { reviewPolicyLabel } from "./mailbox-form-dialog";

type Translate = ReturnType<typeof useT>;

const DAY_SECONDS = 86_400;

type Fix = "secret" | "apiKey" | "test";

type Issue = { variant: keyof typeof ISSUE_ICON; text: string; fix: Fix };

type Step = { key: Fix; label: string; done: boolean; detail: string };

type FixAction = { label: string; run: () => void; pending: boolean };

const ISSUE_ICON = {
  destructive: XCircleIcon,
  warning: AlertTriangleIcon,
  info: InfoCircleIcon,
} as const;

function ago(at: number, now: number): string {
  return formatRelativeTime(at - now);
}

function testArrived(mailbox: InboundMailbox): boolean {
  return (
    mailbox.testSentAt !== null &&
    mailbox.testReceivedAt !== null &&
    mailbox.testReceivedAt >= mailbox.testSentAt
  );
}

/**
 * Where the mailbox stands, worst first: off, then refusing every delivery
 * for want of a secret, then whatever the server worked out from when mail
 * last arrived.
 */
function mailboxState(
  t: Translate,
  mailbox: InboundMailbox,
): { label: string; variant: BadgeVariant } {
  if (mailbox.status !== "Active") {
    return { label: t("Paused"), variant: "neutral" };
  }
  if (!mailbox.hasSigningSecret) {
    return { label: t("Refusing mail"), variant: "danger" };
  }

  switch (mailbox.health) {
    case "Healthy":
      return { label: t("Receiving"), variant: "success" };
    case "Quiet":
      return { label: t("Gone quiet"), variant: "warning" };
    case "NeverReceived":
      return { label: t("Waiting for mail"), variant: "info" };
    default:
      return { label: t("Paused"), variant: "neutral" };
  }
}

/**
 * The one thing most worth fixing on a listening mailbox, or nothing. A
 * missing secret outranks everything, because until it is set every delivery
 * is refused; a Resend mailbox without a key comes next, because its mail
 * lands without a body; silence comes last.
 */
function topIssue(t: Translate, mailbox: InboundMailbox, now: number): Issue | null {
  if (mailbox.status !== "Active") {
    return null;
  }
  if (!mailbox.hasSigningSecret) {
    return {
      variant: "destructive",
      text: t("No signing secret is set, so every delivery to this mailbox is refused."),
      fix: "secret",
    };
  }
  if (mailbox.provider === "Resend" && !mailbox.hasApiKey) {
    return {
      variant: "warning",
      text: t(
        "No API key is set, so mail arrives without its body or attachments and waits for a person.",
      ),
      fix: "apiKey",
    };
  }
  if (mailbox.health === "Quiet" && mailbox.lastReceivedAt !== null) {
    return {
      variant: "warning",
      text: t(
        "No mail in {0, plural, one {# day} other {# days}}. Send a test email to check the address still works.",
        Math.floor((now - mailbox.lastReceivedAt) / DAY_SECONDS),
      ),
      fix: "test",
    };
  }
  if (mailbox.health === "NeverReceived") {
    if (mailbox.testSentAt !== null && !testArrived(mailbox)) {
      return {
        variant: "info",
        text: t("A test email was sent {0} and hasn't arrived yet.", ago(mailbox.testSentAt, now)),
        fix: "test",
      };
    }

    return {
      variant: "info",
      text: t("Nothing has arrived yet. Send a test email to check the address works."),
      fix: "test",
    };
  }

  return null;
}

/** What a mailbox needs before mail flows through it, in the order it needs it. */
function setupSteps(t: Translate, mailbox: InboundMailbox, now: number): Step[] {
  const delivered = mailbox.lastReceivedAt !== null || testArrived(mailbox);
  const steps: Step[] = [
    {
      key: "secret",
      label: t("Signing secret"),
      done: mailbox.hasSigningSecret,
      detail: mailbox.hasSigningSecret
        ? t("Deliveries are verified")
        : t("Every delivery is refused until it is set"),
    },
  ];
  if (mailbox.provider === "Resend") {
    steps.push({
      key: "apiKey",
      label: t("API key"),
      done: mailbox.hasApiKey,
      detail: mailbox.hasApiKey
        ? t("Bodies and attachments are read")
        : t("Mail arrives without its body or attachments"),
    });
  }
  steps.push({
    key: "test",
    label: t("First delivery"),
    done: delivered,
    detail: delivered
      ? t("Mail has arrived")
      : mailbox.testSentAt !== null
        ? t("Test sent {0}, not arrived yet", ago(mailbox.testSentAt, now))
        : t("Nothing has arrived yet"),
  });

  return steps;
}

/**
 * One address and what it is trusted with. A mailbox that works says so in a
 * line; one that does not leads with the single thing most worth fixing and
 * the button that fixes it, and a mailbox still being set up shows how far
 * along it is, with every step a click away.
 */
export function MailboxCard({
  mailbox,
  canEdit,
  now = Math.floor(Date.now() / 1000),
  onEdit,
  onRotate,
  onSecret,
  onApiKey,
  onSendTest,
}: {
  mailbox: InboundMailbox;
  canEdit: boolean;
  now?: number;
  onEdit: () => void;
  onRotate: () => void;
  onSecret: () => void;
  onApiKey: () => void;
  onSendTest: () => Promise<unknown>;
}) {
  const t = useT();
  const listening = mailbox.status === "Active";
  const readsContent = mailbox.provider === "Resend";
  const state = mailboxState(t, mailbox);
  const issue = topIssue(t, mailbox, now);
  const steps = setupSteps(t, mailbox, now);
  const { pending, run } = usePendingActions();
  const sendingTest = pending.has("test");
  const sendTest = () => void run("test", "test", onSendTest);
  const fixes: Record<Fix, FixAction> = {
    secret: { label: t("Set signing secret"), run: onSecret, pending: false },
    apiKey: { label: t("Set API key"), run: onApiKey, pending: false },
    test: { label: t("Send a test email"), run: sendTest, pending: sendingTest },
  };

  return (
    <article className="bg-card border-border flex flex-col rounded-lg border">
      <div className="flex flex-col gap-3 p-4">
        <header className="flex items-start gap-3">
          <span className="bg-muted text-foreground-muted flex size-9 shrink-0 items-center justify-center rounded-md">
            <Mail01Icon className="size-4" aria-hidden />
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex min-w-0 items-center gap-2">
              <h3 className="truncate text-sm font-semibold">{mailbox.name}</h3>
              <Badge variant={state.variant}>{state.label}</Badge>
            </div>
            <AddressLine address={mailbox.address} />
            <p className="text-foreground-subtle truncate text-xs">
              {mailbox.provider}
              {" · "}
              {mailbox.lastReceivedAt === null
                ? t("Never received")
                : t("Last mail {0}", ago(mailbox.lastReceivedAt, now))}
              {" · "}
              {t("Created {0}", formatUnixDate(mailbox.createdAt))}
            </p>
            {mailbox.purpose !== "" && (
              <p className="text-foreground-muted mt-1 line-clamp-2 text-xs">{mailbox.purpose}</p>
            )}
          </div>
          {canEdit && (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    className="-mt-0.5 -mr-1.5 shrink-0"
                    aria-label={t("Mailbox actions")}
                  >
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                }
              />
              <DropdownMenuContent side="bottom" align="end">
                <DropdownMenuItem
                  startContent={<Edit02Icon className="size-3.5" />}
                  title={t("Edit")}
                  onClick={onEdit}
                />
                <DropdownMenuItem
                  startContent={<Key01Icon className="size-3.5" />}
                  title={mailbox.hasSigningSecret ? t("Replace secret") : t("Set signing secret")}
                  onClick={onSecret}
                />
                {readsContent && (
                  <DropdownMenuItem
                    startContent={<LockKeyholeCircleIcon className="size-3.5" />}
                    title={mailbox.hasApiKey ? t("Replace API key") : t("Set API key")}
                    onClick={onApiKey}
                  />
                )}
                {listening && (
                  <DropdownMenuItem
                    startContent={<Send01Icon className="size-3.5" />}
                    title={t("Send a test email")}
                    disabled={sendingTest}
                    onClick={sendTest}
                  />
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  startContent={<RefreshCw02Icon className="size-3.5" />}
                  title={t("Rotate URL")}
                  color="danger"
                  onClick={onRotate}
                />
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </header>

        {issue !== null && (
          <IssueCallout issue={issue} fix={canEdit ? fixes[issue.fix] : undefined} />
        )}

        <SetupProgress
          steps={steps}
          fixFor={(step) =>
            canEdit && (step.key !== "test" || listening) ? fixes[step.key] : undefined
          }
        />
      </div>

      <footer className="border-border-subtle mt-auto flex items-center gap-3 border-t px-4 py-2.5">
        <span className="text-foreground-muted flex min-w-0 flex-1 items-center gap-1.5 text-xs">
          <Shield01Icon className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">
            {reviewPolicyLabel(t, mailbox.reviewPolicy)}
            {mailbox.reviewPolicy === "ReviewBelowConfidence" &&
              ` · ${t("bar {0}%", Math.round(mailbox.minConfidence * 100))}`}
          </span>
        </span>
        <Button
          size="xs"
          variant="ghost"
          className="-mr-1.5 shrink-0"
          nativeButton={false}
          render={<Link to={`/inbox?mailbox=${encodeURIComponent(mailbox.id)}`} />}
        >
          {t("Open its mail")}
          <ArrowRightIcon className="size-3.5" />
        </Button>
      </footer>
    </article>
  );
}

function IssueCallout({ issue, fix }: { issue: Issue; fix?: FixAction }) {
  const Icon = ISSUE_ICON[issue.variant];

  return (
    <Alert size="sm" variant={issue.variant}>
      <Icon aria-hidden />
      <AlertDescription>{issue.text}</AlertDescription>
      {fix !== undefined && (
        <AlertAction>
          <PendingFixButton fix={fix} />
        </AlertAction>
      )}
    </Alert>
  );
}

/**
 * How far along setup is, shown only while it is not finished: a mailbox that
 * has its secret, its key and its first delivery has nothing left to say here.
 */
function SetupProgress({
  steps,
  fixFor,
}: {
  steps: Step[];
  fixFor: (step: Step) => FixAction | undefined;
}) {
  const t = useT();
  const done = steps.filter((step) => step.done).length;
  if (done === steps.length) {
    return null;
  }

  return (
    <Collapsible>
      <CollapsibleTrigger className="group/setup ui-focus-ring flex w-full items-center gap-3 rounded-md text-left">
        <span className="text-foreground-muted text-xs font-medium">{t("Setup")}</span>
        <span className="flex gap-0.5" aria-hidden>
          {steps.map((step) => (
            <span
              key={step.key}
              className={cn("h-1.5 w-5 rounded-full", step.done ? "bg-success" : "bg-muted")}
            />
          ))}
        </span>
        <span className="text-foreground-muted text-xs tabular-nums">
          {t("{0} of {1} done", done, steps.length)}
        </span>
        <span className="text-foreground-subtle group-hover/setup:text-foreground ml-auto flex items-center gap-1 text-xs">
          {t("Steps")}
          <ChevronDownIcon
            className="size-3.5 transition-transform group-data-[panel-open]/setup:rotate-180"
            aria-hidden
          />
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ol className="mt-2 flex flex-col gap-2.5">
          {steps.map((step) => {
            const fix = step.done ? undefined : fixFor(step);

            return (
              <li key={step.key} className="flex items-start gap-2.5">
                {step.done ? (
                  <CheckCircleIcon className="text-success mt-0.5 size-4 shrink-0" aria-hidden />
                ) : (
                  <span
                    className="border-border-strong mt-0.5 size-4 shrink-0 rounded-full border border-dashed"
                    aria-hidden
                  />
                )}
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="text-sm font-medium">{step.label}</span>
                  <span className="text-foreground-muted text-xs">{step.detail}</span>
                </div>
                {fix !== undefined && <PendingFixButton fix={fix} className="shrink-0" />}
              </li>
            );
          })}
        </ol>
      </CollapsibleContent>
    </Collapsible>
  );
}

/** A fix's button, held while its request is in flight so it cannot fire twice. */
function PendingFixButton({ fix, className }: { fix: FixAction; className?: string }) {
  const t = useT();

  return (
    <Button
      size="xs"
      variant="outline"
      className={className}
      isLoading={fix.pending}
      loadingText={t("Sending…")}
      onClick={fix.run}
    >
      {fix.label}
    </Button>
  );
}

function AddressLine({ address }: { address: string }) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="flex min-w-0 items-center gap-1">
      <span className="text-foreground-muted truncate font-mono text-xs">{address}</span>
      <Button
        size="icon-xs"
        variant="ghost"
        className="text-foreground-subtle shrink-0"
        aria-label={isCopied ? t("Copied") : t("Copy address")}
        onClick={() => void copy(address)}
      >
        {isCopied ? (
          <CheckIcon className="text-success size-3" />
        ) : (
          <Copy01Icon className="size-3" />
        )}
      </Button>
    </div>
  );
}
