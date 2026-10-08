import { verifyAIAuditChain, type AIAuditChainStatus } from "@/lib/graphql/ai-audit";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useEffect, useState, type ReactNode } from "react";
import { Callout } from "../edit/fields";
import { Hero } from "../kit/hero";
import { Ic } from "../kit/ic";
import { Figs, type Fig } from "../kit/layout";
import {
  VERIFY_POLL_MS,
  VERIFY_WAIT_MS,
  isVerificationPending,
  unsealedRows,
  verificationLabel,
  verificationTone,
  type VerificationRequest,
} from "./audit-model";

const TONE_CLASS: Partial<Record<ReturnType<typeof verificationTone>, string>> = {
  success: "t-k",
  danger: "t-d",
  warning: "t-w",
};

/**
 * The head of the audit trail: Nova's sentence on the chain the trail is written into,
 * whether it is signed, how far it is sealed and what its last check found, with a way
 * to check it now, and the same as figures. A check runs in the background; the sentence
 * says so until its result is stored.
 */
export function ChainStatusHeader() {
  const t = useT();
  const queryClient = useQueryClient();
  const statusQuery = queries.aiAudit.chainStatus();
  const [request, setRequest] = useState<VerificationRequest | null>(null);
  const [now] = useState(() => Date.now());

  const chain = useQuery({
    ...statusQuery,
    refetchInterval: (query) =>
      isVerificationPending(request, query.state.data ?? null) ? VERIFY_POLL_MS : false,
  });

  // A check that never reports back stops being waited on, so the button
  // comes back rather than spinning for the rest of the visit.
  useEffect(() => {
    if (request === null) {
      return;
    }
    const timer = window.setTimeout(() => setRequest(null), VERIFY_WAIT_MS);
    return () => window.clearTimeout(timer);
  }, [request]);

  const verify = useMutation({
    mutationFn: verifyAIAuditChain,
    onMutate: () => setRequest(null),
    onSuccess: (status) => {
      setRequest({ baseline: status.lastVerifiedAt ?? null });
      queryClient.setQueryData(statusQuery.queryKey, status);
    },
  });

  if (chain.isError) {
    return (
      <Callout tone="d">
        {t("The audit chain's status could not be loaded. Try again shortly.")}
      </Callout>
    );
  }
  if (!chain.data) {
    return null;
  }

  const status = chain.data;
  const pending = isVerificationPending(request, status) || verify.isPending;
  const busyLabel = verify.isPending ? t("Starting the check…") : t("Verifying…");

  return (
    <>
      <Hero
        context={t("Audit trail")}
        working={pending}
        control={
          <>
            <button
              type="button"
              className={pending ? "btn lg" : "btn ink lg"}
              disabled={pending}
              onClick={() => verify.mutate()}
            >
              <Ic n="shield" s={13} />
              {pending ? busyLabel : t("Verify now")}
            </button>
            <span>
              {pending
                ? t("The result appears here when the check finishes.")
                : t("Checked every night on its own.")}
            </span>
          </>
        }
      >
        <ChainSentence status={status} pending={pending} />
      </Hero>
      <Figs
        label={t("Audit chain")}
        items={[
          {
            label: t("Chain"),
            value: status.signed ? t("Signed") : t("Unsigned"),
            sub: status.signed
              ? t("Key {0}", status.activeKeyId ?? "—")
              : t("Plain SHA-256; no signing key is configured"),
            tone: status.signed ? undefined : "t-w",
          },
          {
            label: t("Sealed through"),
            value: status.lastSeq > 0 ? `#${status.sealedThroughSeq.toLocaleString()}` : "—",
            sub:
              status.lastSeq > 0
                ? t("of {0} recorded", status.lastSeq.toLocaleString())
                : t("Nothing recorded yet"),
          },
          {
            label: t("Last verified"),
            value: pending ? "…" : verificationLabel(t, status.lastVerificationStatus),
            sub: lastVerifiedSub(t, status, now),
            tone: pending ? undefined : TONE_CLASS[verificationTone(status.lastVerificationStatus)],
          } satisfies Fig,
        ]}
      />
      {verify.isError && (
        <Callout tone="d">
          {graphQLErrorMessage(verify.error, t("The check could not be started."))}
        </Callout>
      )}
      {status.lastVerificationStatus === "Mismatch" && (
        <Callout tone="d">
          {status.failedSeq != null
            ? t(
                "The trail no longer matches its chain at #{0}: {1}. The failure is in the audit log, and everyone who reads the trail was told.",
                status.failedSeq.toLocaleString(),
                status.detail ?? t("no detail was recorded"),
              )
            : t(
                "The trail no longer matches its chain: {0}. The failure is in the audit log, and everyone who reads the trail was told.",
                status.detail ?? t("no detail was recorded"),
              )}
        </Callout>
      )}
      {status.lastVerificationStatus === "KeyMissing" && (
        <Callout tone="w">
          {t(
            "A row names a signing key that is no longer configured, so the chain cannot be checked past it. Put the key back, then verify again.",
          )}
        </Callout>
      )}
    </>
  );
}

function ChainSentence({ status, pending }: { status: AIAuditChainStatus; pending: boolean }) {
  const t = useT();
  const rt = useRichT();
  const strong = (children: ReactNode) => <b>{children}</b>;

  if (pending) {
    return t("Checking the chain from the last sealed row…");
  }
  if (status.lastSeq === 0) {
    return t("Nothing has been written to the audit trail yet.");
  }

  const when =
    status.lastVerifiedAt == null ? null : formatUnixDateTimeShort(status.lastVerifiedAt);
  const unsealed = unsealedRows(status);

  return (
    <>
      {status.signed
        ? rt("Every agent action is written to a <b>signed chain</b>.", { b: strong })
        : rt(
            "Every agent action is written to a <b>chain</b>, unsigned because no signing key is configured.",
            { b: strong },
          )}{" "}
      {when === null || status.lastVerificationStatus === null
        ? t("It has not been checked yet.")
        : status.lastVerificationStatus === "Verified"
          ? rt(
              "It was last checked {0} and is <k>intact</k> through #{1}.",
              { k: (children) => <b className="t-k">{children}</b> },
              when,
              status.lastVerifiedSeq.toLocaleString(),
            )
          : status.lastVerificationStatus === "Mismatch"
            ? status.failedSeq != null
              ? rt(
                  "It was last checked {0} and <d>no longer matches</d> at #{1}.",
                  { d: (children) => <b className="t-d">{children}</b> },
                  when,
                  status.failedSeq.toLocaleString(),
                )
              : rt(
                  "It was last checked {0} and <d>no longer matches</d>.",
                  { d: (children) => <b className="t-d">{children}</b> },
                  when,
                )
            : rt(
                "It was last checked {0} and <w>could not be checked</w> past a key that is no longer configured.",
                { w: (children) => <b className="t-w">{children}</b> },
                when,
              )}{" "}
      {unsealed > 0
        ? t(
            "{0, plural, one {The newest row is sealed at the next check.} other {The # newest rows are sealed at the next check.}}",
            unsealed,
          )
        : t("Every row is sealed.")}
    </>
  );
}

function lastVerifiedSub(
  t: ReturnType<typeof useT>,
  status: AIAuditChainStatus,
  now: number,
): string {
  if (status.lastVerifiedAt == null) {
    return t("Never checked");
  }

  const at = formatUnixDateTimeShort(status.lastVerifiedAt);
  const ageDays = Math.floor((now / 1000 - status.lastVerifiedAt) / 86_400);
  return ageDays > 1
    ? t("{0} ({1} days ago), through #{2}", at, ageDays, status.lastVerifiedSeq.toLocaleString())
    : t("{0}, through #{1}", at, status.lastVerifiedSeq.toLocaleString());
}
