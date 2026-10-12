import type { InboundMailbox, InboundProvider } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  Copy01Icon,
  Inbox01Icon,
  LinkExternal01Icon,
  MessageChatSquareIcon,
  SwitchHorizontal01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useId, useReducer, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { mailboxName } from "../../folder-labels";
import { inlineErrorText } from "../../inline-error";
import { useTriageActions } from "../../use-triage-actions";
import { useTriageCounts, useTriageMailboxes } from "../../use-triage-data";
import {
  NUMBERED_STEPS,
  canContinue,
  canGoBack,
  connectReducer,
  initialConnect,
  receivingDomain,
  testArrived,
  type ConnectState,
} from "./connect-flow";

type SourceKind = { key: string; icon: IconComponent; soon: boolean };

const SOURCES: readonly SourceKind[] = [
  { key: "email", icon: Inbox01Icon, soon: false },
  { key: "edi", icon: SwitchHorizontal01Icon, soon: true },
  { key: "portal", icon: LinkExternal01Icon, soon: true },
  { key: "texts", icon: MessageChatSquareIcon, soon: true },
];

const PROVIDERS = ["google", "microsoft", "other"] as const;
type MailProvider = (typeof PROVIDERS)[number];

/** Where a source's settings are kept, for what Triage itself does not set. */
const SOURCES_PATH = "/admin/inbound-mailboxes";

function sourceText(t: TranslateFn, key: string): { title: string; detail: string } {
  switch (key) {
    case "email":
      return { title: t("Email"), detail: t("A shared address — dispatch@, pod@, ap@") };
    case "edi":
      return { title: t("EDI"), detail: t("204 tenders, 214 status, 210 invoices") };
    case "portal":
      return { title: t("Customer portals"), detail: t("Loads posted on a shipper's site") };
    default:
      return { title: t("Texts"), detail: t("Drivers texting photos and updates") };
  }
}

function providerText(t: TranslateFn, provider: MailProvider): { name: string; how: string } {
  switch (provider) {
    case "google":
      return {
        name: t("Google Workspace"),
        how: t(
          "In Gmail settings for the shared address, open Forwarding, add the address above, confirm it, and keep Gmail's copy in the inbox.",
        ),
      };
    case "microsoft":
      return {
        name: t("Microsoft 365"),
        how: t(
          "In the Exchange admin center, open the shared mailbox, turn on forwarding to the address above, and keep a copy in the mailbox.",
        ),
      };
    case "other":
      return {
        name: t("Other"),
        how: t(
          "Add a forwarding rule for every message to the address above, and keep a copy in the original inbox.",
        ),
      };
  }
}

/** The provider the organization's other sources use, so a new one is set up the same way. */
function providerOf(mailboxes: readonly InboundMailbox[]): InboundProvider {
  const resend = mailboxes.filter((mailbox) => mailbox.provider === "Resend").length;
  const postmark = mailboxes.length - resend;

  return resend > postmark ? "Resend" : "Postmark";
}

/**
 * Connecting a source: pick what kind (email today; EDI, portals and texts
 * soon), say which shared address Nova should watch and what to call it,
 * forward that address to the source Trenova receives on, then send a
 * message and wait for it to land. Creating a source is an administrator's
 * to do; anyone else is told so rather than shown a form that would refuse.
 */
export function ConnectSource({
  mailboxId,
  onClose,
}: {
  /** A source to reconnect: the flow starts at forwarding it. */
  mailboxId: string | null;
  onClose: () => void;
}) {
  const t = useT();
  const mailboxes = useTriageMailboxes().data;
  const existing =
    mailboxId === null ? null : (mailboxes?.find((row) => row.id === mailboxId) ?? null);
  const ready = mailboxId === null || mailboxes !== undefined;

  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent size="sm" className="max-w-125 gap-0 p-0 sm:max-w-125 [&>*]:min-w-0">
        {ready && (
          <ConnectBody
            key={existing?.id ?? "new"}
            existing={existing}
            mailboxes={mailboxes ?? []}
            onClose={onClose}
          />
        )}
        {!ready && (
          <DialogHeader className="p-4">
            <DialogTitle>{t("Add a source")}</DialogTitle>
          </DialogHeader>
        )}
      </DialogContent>
    </Dialog>
  );
}

function ConnectBody({
  existing,
  mailboxes,
  onClose,
}: {
  existing: InboundMailbox | null;
  mailboxes: readonly InboundMailbox[];
  onClose: () => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const canCreate = usePermissionStore((store) =>
    store.hasPermission(Resource.InboundMailbox, Operation.Create),
  );
  const [state, dispatch] = useReducer(connectReducer, null, () =>
    initialConnect(existing, receivingDomain(mailboxes)),
  );
  const blocked = existing === null && !canCreate;

  const create = () => {
    dispatch({ type: "saving" });
    actions
      .connectSource({
        name: state.name,
        address: state.receiveAt,
        watch: state.watch,
        provider: providerOf(mailboxes),
      })
      .then(
        (connected) =>
          dispatch({
            type: "created",
            mailbox: connected.mailbox,
            webhookUrl: connected.webhookUrl,
          }),
        (error: unknown) => dispatch({ type: "failed", message: inlineErrorText(t, error) }),
      );
  };

  const next = () => {
    if (state.step === "address") {
      create();
      return;
    }
    dispatch({ type: "continue" });
  };

  return (
    <>
      <DialogHeader className="border-border-subtle gap-0.5 border-b px-4 pt-3.5 pb-3">
        <DialogTitle className="text-base font-semibold">
          {existing === null ? t("Add a source") : t("Reconnect {0}", mailboxName(existing))}
        </DialogTitle>
        <DialogDescription className="text-foreground-subtle text-sm">
          {state.step === "source"
            ? t("Anywhere freight work arrives. Nova reads all of it the same way.")
            : t(
                "Takes a couple of minutes. Nothing changes for the people already using this address.",
              )}
        </DialogDescription>
      </DialogHeader>

      {state.step !== "source" && <Steps state={state} />}

      <div className="flex flex-col gap-3.5 px-4 py-3.5">
        {blocked ? (
          <Alert size="sm">
            <AlertDescription>
              {t(
                "Adding a source needs mailbox access, which administrators have. Ask one to add it, or to give you access.",
              )}
            </AlertDescription>
          </Alert>
        ) : (
          <StepBody state={state} dispatch={dispatch} />
        )}
        {state.error !== null && (
          <p role="alert" className="text-danger text-sm">
            {state.error}
          </p>
        )}
      </div>

      <DialogFooter className="mx-0 mb-0 flex-row items-center justify-end gap-1.5 px-3 py-2.5">
        {state.step === "done" ? (
          <Button onClick={onClose}>{t("Done")}</Button>
        ) : (
          <>
            <Button
              variant="ghost"
              onClick={() => (canGoBack(state) ? dispatch({ type: "back" }) : onClose())}
            >
              {canGoBack(state) ? t("Back") : t("Cancel")}
            </Button>
            {state.step !== "test" && (
              <Button
                disabled={blocked || !canContinue(state)}
                isLoading={state.saving}
                loadingText={t("Creating…")}
                onClick={next}
              >
                {t("Continue")}
              </Button>
            )}
          </>
        )}
      </DialogFooter>
    </>
  );
}

function Steps({ state }: { state: ConnectState }) {
  const t = useT();
  const current =
    state.step === "done" ? NUMBERED_STEPS.length : NUMBERED_STEPS.indexOf(state.step as never);

  return (
    <ol className="flex gap-4 px-4 pt-3">
      {NUMBERED_STEPS.map((step, index) => {
        const done = index < current;
        const on = index === current;
        return (
          <li
            key={step}
            aria-current={on ? "step" : undefined}
            className={cn(
              "flex items-center gap-1.75 text-sm",
              done ? "text-success" : on ? "text-foreground" : "text-foreground-subtle",
            )}
          >
            <i
              aria-hidden
              className={cn(
                "flex size-4.5 items-center justify-center rounded-full border font-mono text-2xs not-italic",
                done
                  ? "bg-success border-success text-foreground-on-solid"
                  : on
                    ? "bg-ink border-ink text-ink-foreground"
                    : "border-border-strong",
              )}
            >
              {done ? <CheckIcon className="size-2.5" strokeWidth={3} /> : index + 1}
            </i>
            {step === "address" ? t("Address") : step === "forward" ? t("Forward") : t("Test")}
          </li>
        );
      })}
    </ol>
  );
}

function Field({
  label,
  hint,
  children,
  id,
}: {
  label: string;
  hint?: string;
  children: ReactNode;
  id: string;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint !== undefined && <p className="text-foreground-subtle text-xs">{hint}</p>}
    </div>
  );
}

function StepBody({
  state,
  dispatch,
}: {
  state: ConnectState;
  dispatch: (event: Parameters<typeof connectReducer>[1]) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const actions = useTriageActions();
  const ids = useId();
  const [provider, setProvider] = useState<MailProvider | null>(null);
  const [testing, setTesting] = useState(false);

  switch (state.step) {
    case "source":
      return (
        <ul role="radiogroup" aria-label={t("Kind of source")} className="flex flex-col gap-1.5">
          {SOURCES.map((source) => {
            const text = sourceText(t, source.key);
            const Icon = source.icon;
            const chosen = source.key === "email";
            return (
              <li key={source.key}>
                <button
                  type="button"
                  role="radio"
                  aria-checked={chosen}
                  disabled={source.soon}
                  className={cn(
                    "ui-focus-ring flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors",
                    chosen ? "border-foreground bg-surface-hover" : "border-border",
                    source.soon ? "cursor-default opacity-55" : "hover:border-border-strong",
                  )}
                >
                  <span className="bg-surface-hover text-foreground-muted flex size-7.5 shrink-0 items-center justify-center rounded-md">
                    <Icon className="size-4" />
                  </span>
                  <span className="flex min-w-0 flex-1 flex-col leading-snug">
                    <b className="text-base font-medium">{text.title}</b>
                    <span className="text-foreground-subtle text-sm">{text.detail}</span>
                  </span>
                  {source.soon ? (
                    <em className="bg-surface-hover text-foreground-subtle rounded-full px-1.75 py-0.5 text-2xs not-italic">
                      {t("Soon")}
                    </em>
                  ) : (
                    <CheckIcon aria-hidden className="size-3.5" strokeWidth={2.4} />
                  )}
                </button>
              </li>
            );
          })}
        </ul>
      );
    case "address":
      return (
        <>
          <Field id={`${ids}-watch`} label={t("Which address should Nova watch?")}>
            <Input
              id={`${ids}-watch`}
              type="email"
              autoFocus
              autoComplete="off"
              placeholder="dispatch@yourcompany.com"
              value={state.watch}
              onChange={(event) =>
                dispatch({ type: "edit", field: "watch", value: event.target.value })
              }
            />
          </Field>
          <Field id={`${ids}-name`} label={t("Name it")}>
            <Input
              id={`${ids}-name`}
              autoComplete="off"
              placeholder={t("Dispatch")}
              value={state.name}
              onChange={(event) =>
                dispatch({ type: "edit", field: "name", value: event.target.value })
              }
            />
          </Field>
          <Field
            id={`${ids}-receive`}
            label={t("Trenova receives it at")}
            hint={
              state.domain === null
                ? t("The address your email provider receives Trenova's mail on.")
                : t("Named after the source, on the domain your other sources use.")
            }
          >
            <Input
              id={`${ids}-receive`}
              type="email"
              autoComplete="off"
              className="font-mono"
              placeholder="dispatch@inbound.yourcompany.com"
              value={state.receiveAt}
              onChange={(event) =>
                dispatch({ type: "edit", field: "receiveAt", value: event.target.value })
              }
            />
          </Field>
        </>
      );
    case "forward": {
      const address = state.mailbox?.address ?? state.receiveAt;
      return (
        <>
          <p className="text-foreground-muted text-sm leading-relaxed">
            {state.watch.trim() === ""
              ? t(
                  "In your email provider, forward the shared address to this one. Keep a copy in the original inbox.",
                )
              : rt(
                  "In your email provider, forward <b>{0}</b> to this address. Keep a copy in the original inbox.",
                  { b: (chunks) => <b className="text-foreground font-medium">{chunks}</b> },
                  state.watch.trim(),
                )}
          </p>
          <div className="border-border bg-sunken flex items-center justify-between gap-2 rounded-lg border py-2 pr-2 pl-3">
            <span className="truncate font-mono text-sm">{address}</span>
            <Button variant="ghost" size="sm" onClick={() => actions.copyText(address)}>
              <Copy01Icon className="size-3.5" />
              {t("Copy")}
            </Button>
          </div>
          <div className="flex flex-wrap gap-1">
            {PROVIDERS.map((key) => (
              <button
                key={key}
                type="button"
                aria-pressed={provider === key}
                onClick={() => setProvider(provider === key ? null : key)}
                className={cn(
                  "ui-focus-ring inline-flex h-5.5 items-center rounded-full px-2 text-xs transition-colors",
                  provider === key
                    ? "bg-ink text-ink-foreground"
                    : "bg-surface-hover text-foreground-muted hover:text-foreground",
                )}
              >
                {providerText(t, key).name}
              </button>
            ))}
          </div>
          {provider !== null && (
            <p className="text-foreground-subtle animate-rise text-sm">
              {providerText(t, provider).how}
            </p>
          )}
          {state.webhookUrl !== null && (
            <div className="border-border-subtle flex flex-col gap-1 border-t pt-3">
              <span className="text-foreground-subtle text-xs">
                {t("Your mail provider posts this source's mail to")}
              </span>
              <div className="flex items-center gap-2">
                <span className="text-foreground-muted min-w-0 flex-1 truncate font-mono text-xs">
                  {state.webhookUrl}
                </span>
                <Button
                  variant="ghost"
                  size="xs"
                  onClick={() =>
                    actions.copyText(state.webhookUrl ?? "", t("Webhook address copied"))
                  }
                >
                  <Copy01Icon className="size-3" />
                  {t("Copy")}
                </Button>
              </div>
              {state.mailbox?.provider === "Resend" && !state.mailbox.hasSigningSecret && (
                <p className="text-warning text-xs">
                  {rt(
                    "Resend signs what it posts: add its signing secret in <a>Source settings</a> before mail is read.",
                    {
                      a: (chunks) => (
                        <Link
                          to={SOURCES_PATH}
                          className="text-brand underline-offset-4 hover:underline"
                        >
                          {chunks}
                        </Link>
                      ),
                    },
                  )}
                </p>
              )}
            </div>
          )}
        </>
      );
    }
    case "test":
      return state.mailbox === null ? null : (
        <TestStep
          mailbox={state.mailbox}
          waiting={state.waiting}
          testing={testing}
          onPersonSent={() =>
            dispatch({ type: "waitFor", via: "person", since: Math.floor(Date.now() / 1000) })
          }
          onSendTest={(mailbox) => {
            setTesting(true);
            actions.testSource(mailbox).then(
              (updated) => {
                setTesting(false);
                dispatch({
                  type: "waitFor",
                  via: "test",
                  since: updated.testSentAt ?? Math.floor(Date.now() / 1000),
                });
              },
              (error: unknown) => {
                setTesting(false);
                dispatch({ type: "failed", message: inlineErrorText(t, error) });
              },
            );
          }}
          onArrived={() => dispatch({ type: "arrived" })}
          onStop={() => dispatch({ type: "stopWaiting" })}
        />
      );
    case "done": {
      const name = state.mailbox === null ? state.name : mailboxName(state.mailbox);
      return (
        <div className="bg-success-subtle text-success-subtle-foreground animate-confirm flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm">
          <span className="bg-success text-foreground-on-solid flex size-5 shrink-0 items-center justify-center rounded-full">
            <CheckIcon aria-hidden className="size-2.75" strokeWidth={2.6} />
          </span>
          <span>
            {state.watch.trim() === ""
              ? t("{0} is connected. New mail shows up in Waiting on you.", name)
              : t(
                  "{0} is connected. New mail on {1} shows up in Waiting on you.",
                  name,
                  state.watch.trim(),
                )}
          </span>
        </div>
      );
    }
  }
}

/**
 * Waiting for the first message. Nothing polls: the counts move over
 * realtime when a message lands on the source, and the source is read again
 * when they do, which is what says the mail (or Trenova's own test) arrived.
 */
function TestStep({
  mailbox,
  waiting,
  testing,
  onPersonSent,
  onSendTest,
  onArrived,
  onStop,
}: {
  mailbox: InboundMailbox;
  waiting: ConnectState["waiting"];
  testing: boolean;
  onPersonSent: () => void;
  onSendTest: (mailbox: InboundMailbox) => void;
  onArrived: () => void;
  onStop: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const counts = useTriageCounts().data;
  const live = useTriageMailboxes().data?.find((row) => row.id === mailbox.id) ?? mailbox;
  const received = counts?.byMailbox.find((row) => row.mailboxId === mailbox.id)?.total ?? 0;

  useEffect(() => {
    if (waiting === null) {
      return;
    }
    void queryClient.invalidateQueries({ queryKey: queries.inbox.mailboxes().queryKey });
  }, [received, waiting, queryClient]);

  useEffect(() => {
    if (waiting !== null && testArrived(live, waiting)) {
      onArrived();
    }
  }, [live, waiting, onArrived]);

  if (waiting !== null) {
    return (
      <div className="flex flex-col items-start gap-2.5">
        <TextShimmer as="p" className="text-sm" duration={1.6}>
          {waiting.via === "test"
            ? t("Sent. Waiting for it to arrive…")
            : t("Waiting for it to arrive…")}
        </TextShimmer>
        <p className="text-foreground-subtle text-xs">
          {t("It shows up here the moment it lands on {0}.", mailbox.address)}
        </p>
        <Button variant="ghost" size="sm" onClick={onStop}>
          {t("Stop waiting")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-start gap-2.5">
      <p className="text-foreground-muted text-sm">
        {t("Send any email to the address. We'll wait for it here.")}
      </p>
      <div className="flex flex-wrap gap-1.5">
        <Button size="sm" onClick={onPersonSent}>
          {t("I sent it")}
        </Button>
        <Button variant="outline" size="sm" isLoading={testing} onClick={() => onSendTest(live)}>
          {t("Send a test email")}
        </Button>
      </div>
    </div>
  );
}
