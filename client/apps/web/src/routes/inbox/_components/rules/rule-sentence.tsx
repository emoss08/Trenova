import type { InboundClassification, TriageRuleAction } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useRichT, type RichTags } from "@trenova/shared/i18n/rich";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { useMemo, useState, type ReactNode } from "react";
import { ruleActionPhrase } from "../action-copy";
import {
  CLASSIFICATION_DOT,
  INBOUND_CLASSIFICATIONS,
  classificationLabel,
} from "../classification";
import { mailboxName } from "../folder-labels";
import { useTriageAssignees, useTriageMailboxes } from "../use-triage-data";
import { RuleBlank, type BlankOption } from "./rule-blank";
import { senderKey, senderOptions, type RuleDraft, type SenderChoice } from "./rule-model";

/** Every action a rule may give Nova, in the order the blank lists them. */
const ACTIONS: readonly TriageRuleAction[] = [
  "FileOnShipment",
  "FileAndQueuePay",
  "DraftReply",
  "DraftLoad",
  "LinkBySender",
  "Ignore",
  "Assign",
];

const ANY_SOURCE = "*";
const SEARCH_DEBOUNCE_MS = 200;

function senderLabel(t: TranslateFn, choice: SenderChoice): string {
  switch (choice.scope) {
    case "Anyone":
      return t("anyone");
    case "AnyCustomer":
      return t("any customer");
    case "AnyCarrier":
      return t("any carrier");
    case "Domain":
    case "Address":
      return choice.value;
  }
}

function senderHint(t: TranslateFn, choice: SenderChoice, typed: boolean): string | undefined {
  switch (choice.scope) {
    case "Domain":
      return typed ? t("Everyone at this domain, once they write") : t("Everyone at this domain");
    case "Address":
      return t("Only this address");
    default:
      return undefined;
  }
}

type Tags = RichTags;

/**
 * The sentence, whole, for each kind. A sentence is one message so a
 * language can move the blanks and inflect the kind; the article before a
 * kind is part of the sentence, never chosen from the English spelling.
 */
function kindSentence(
  rt: ReturnType<typeof useRichT>,
  kind: InboundClassification | null,
  tags: Tags,
  sender: string,
  source: string,
  action: string,
): ReactNode {
  switch (kind) {
    case null:
      return rt(
        "When <k>something</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "Tender":
      return rt(
        "When a <k>tender</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "RateConfirmation":
      return rt(
        "When a <k>rate confirmation</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "ProofOfDelivery":
      return rt(
        "When a <k>proof of delivery</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "Invoice":
      return rt(
        "When an <k>invoice</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "StatusRequest":
      return rt(
        "When a <k>status request</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "DetentionDispute":
      return rt(
        "When a <k>detention dispute</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
    case "Other":
      return rt(
        "When <k>anything else</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>{2}</a>.",
        tags,
        sender,
        source,
        action,
      );
  }
}

/** The same sentences for a rule that hands the message to someone, who is a blank of their own. */
function assignSentence(
  rt: ReturnType<typeof useRichT>,
  kind: InboundClassification | null,
  tags: Tags,
  sender: string,
  source: string,
  person: string,
): ReactNode {
  switch (kind) {
    case null:
      return rt(
        "When <k>something</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "Tender":
      return rt(
        "When a <k>tender</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "RateConfirmation":
      return rt(
        "When a <k>rate confirmation</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "ProofOfDelivery":
      return rt(
        "When a <k>proof of delivery</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "Invoice":
      return rt(
        "When an <k>invoice</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "StatusRequest":
      return rt(
        "When a <k>status request</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "DetentionDispute":
      return rt(
        "When a <k>detention dispute</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
    case "Other":
      return rt(
        "When <k>anything else</k> arrives from <s>{0}</s> on <m>{1}</m>, Nova should <a>hand it</a> to <p>{2}</p>.",
        tags,
        sender,
        source,
        person,
      );
  }
}

export type RuleSentenceProps = {
  draft: RuleDraft;
  onChange: (draft: RuleDraft) => void;
  /** A paused rule reads quieter. */
  quiet?: boolean;
  /** On the add row, a blank that was just filled lands with a settle. */
  fresh?: boolean;
  className?: string;
};

/**
 * A rule as the sentence it is, every bold blank an inline choice. The add
 * row and each saved rule draw the same sentence; what a change does (fill
 * the draft, or save the rule) is the caller's.
 */
export function RuleSentence({
  draft,
  onChange,
  quiet = false,
  fresh = false,
  className,
}: RuleSentenceProps) {
  const t = useT();
  const rt = useRichT();
  const mailboxes = useTriageMailboxes().data;
  const assignees = useTriageAssignees().data;
  const [search, setSearch] = useState("");
  const settled = useDebounce(search.trim(), SEARCH_DEBOUNCE_MS);
  const domains = useQuery({
    ...queries.inbox.senderDomains(settled),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  }).data;

  const kindOptions = useMemo<BlankOption[]>(
    () =>
      INBOUND_CLASSIFICATIONS.map((kind) => ({
        key: kind,
        label: toSentenceFragment(classificationLabel(t, kind)),
        leading: (
          <span
            aria-hidden
            className={cn("size-2 shrink-0 rounded-full", CLASSIFICATION_DOT[kind])}
          />
        ),
      })),
    [t],
  );
  const senders = useMemo(() => senderOptions(domains ?? [], search), [domains, search]);
  const senderBlankOptions = useMemo<BlankOption[]>(
    () =>
      senders.map((option) => ({
        key: option.key,
        label: senderLabel(t, option.choice),
        hint: senderHint(t, option.choice, option.typed),
      })),
    [senders, t],
  );
  const sourceOptions = useMemo<BlankOption[]>(
    () => [
      { key: ANY_SOURCE, label: t("any source"), hint: t("Every address mail arrives on") },
      ...(mailboxes ?? []).map((mailbox) => ({
        key: mailbox.id,
        label: mailboxName(mailbox),
        hint: mailbox.address,
      })),
    ],
    [mailboxes, t],
  );
  const actionOptions = useMemo<BlankOption[]>(
    () =>
      ACTIONS.map((action) => ({
        key: action,
        label: action === "Assign" ? t("hand it to someone") : ruleActionPhrase(t, action),
      })),
    [t],
  );
  const assigneeOptions = useMemo<BlankOption[]>(
    () =>
      (assignees ?? []).map((row) => ({
        key: row.user.id,
        label: row.user.name,
        hint: row.user.emailAddress,
      })),
    [assignees],
  );

  const sourceKey = draft.mailboxId ?? ANY_SOURCE;
  const mailbox = mailboxes?.find((row) => row.id === draft.mailboxId);
  const assignee = assignees?.find((row) => row.user.id === draft.assigneeId)?.user;
  const selectedSender = draft.sender === null ? null : senderKey(draft.sender);

  const tags: Tags = {
    k: (chunks) => (
      <RuleBlank
        label={t("Kind of message")}
        filled={draft.classification !== null}
        selected={draft.classification}
        options={kindOptions}
        onSelect={(key) => onChange({ ...draft, classification: key as InboundClassification })}
        quiet={quiet}
        fresh={fresh && draft.classification !== null}
      >
        {chunks}
      </RuleBlank>
    ),
    s: (chunks) => (
      <RuleBlank
        label={t("Who it comes from")}
        filled={draft.sender !== null}
        selected={selectedSender}
        options={senderBlankOptions}
        onSelect={(key) => {
          const picked = senders.find((option) => option.key === key);
          if (picked !== undefined) {
            onChange({ ...draft, sender: picked.choice });
          }
          setSearch("");
        }}
        searchable
        search={search}
        onSearch={setSearch}
        searchPlaceholder={t("A domain or an address")}
        emptyText={t("Type a domain, like acme.com, or an address")}
        quiet={quiet}
        fresh={fresh && draft.sender !== null}
      >
        {chunks}
      </RuleBlank>
    ),
    m: (chunks) => (
      <RuleBlank
        label={t("Which source")}
        filled
        selected={sourceKey}
        options={sourceOptions}
        onSelect={(key) => onChange({ ...draft, mailboxId: key === ANY_SOURCE ? null : key })}
        quiet={quiet}
      >
        {chunks}
      </RuleBlank>
    ),
    a: (chunks) => (
      <RuleBlank
        label={t("What Nova should do")}
        filled={draft.action !== null}
        selected={draft.action}
        options={actionOptions}
        onSelect={(key) => {
          const action = key as TriageRuleAction;
          onChange({ ...draft, action, assigneeId: action === "Assign" ? draft.assigneeId : null });
        }}
        quiet={quiet}
        fresh={fresh && draft.action !== null}
      >
        {chunks}
      </RuleBlank>
    ),
    p: (chunks) => (
      <RuleBlank
        label={t("Who to hand it to")}
        filled={draft.assigneeId !== null}
        selected={draft.assigneeId}
        options={assigneeOptions}
        onSelect={(key) => onChange({ ...draft, assigneeId: key })}
        emptyText={t("Nobody can be assigned messages")}
        quiet={quiet}
        fresh={fresh && draft.assigneeId !== null}
      >
        {chunks}
      </RuleBlank>
    ),
  };

  const sender = draft.sender === null ? t("someone") : senderLabel(t, draft.sender);
  const source =
    draft.mailboxId === null
      ? t("any source")
      : mailbox === undefined
        ? t("a source")
        : mailboxName(mailbox);

  return (
    <p
      className={cn(
        "m-0 text-2xl leading-normal text-pretty @max-2xl:text-xl",
        quiet ? "text-foreground-subtle" : "text-foreground-muted",
        className,
      )}
    >
      {draft.action === "Assign"
        ? assignSentence(
            rt,
            draft.classification,
            tags,
            sender,
            source,
            assignee?.name ?? t("someone"),
          )
        : kindSentence(
            rt,
            draft.classification,
            tags,
            sender,
            source,
            draft.action === null ? t("do something") : ruleActionPhrase(t, draft.action),
          )}
    </p>
  );
}
