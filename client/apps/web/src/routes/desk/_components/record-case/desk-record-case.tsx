import { withoutRecordIds } from "@/lib/record-ids";
import { recordSpace } from "@/stores/composer-draft-store";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { CaseUnworked } from "./case-unworked";
import { CaseOwner } from "./case-owner";
import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { isTypingTarget } from "@/lib/dom";
import {
  acceptDeskCaseRisk,
  bringBackDeskCase,
  putAwayDeskCase,
  chooseDeskCaseOption,
  type DeskCase,
  type DeskCaseOption,
} from "@/lib/graphql/desk-case";
import { assistant } from "@/lib/queries/assistant";
import { attention } from "@/lib/queries/attention";
import { deskAttention } from "@/lib/queries/desk-attention";
import { deskCase } from "@/lib/queries/desk-case";
import { watchtower } from "@/lib/queries/watchtower";
import { assistantService } from "@/services/assistant";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useDeskStore } from "@/stores/desk-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, Navigate, useLocation } from "react-router";
import { useClaimOnOpen } from "./use-claim-on-open";
import { casePath, startsCase } from "../attention/attention-record";
import { DeclineDialog } from "../attention/decline-dialog";
import { useDesk } from "../desk-layout";
import { AcceptRiskDialog } from "./accept-risk-dialog";
import { CaseDock } from "./case-dock";
import { CaseExhibits } from "./case-exhibits";
import { CaseClosed } from "./case-closed";
import { CaseFinding } from "./case-finding";
import { useCaseWriting } from "./case-writing";
import { CaseHead } from "./case-head";
import { CaseOptions } from "./case-options";
import { CasePlan, CasePlanDrafting, CasePlanNone } from "./case-plan";
import { CaseAwayNotice, CasePutAway } from "./case-put-away";
import { CaseSkeleton } from "./case-skeleton";
import {
  CASE_TRANSITION_CLASS,
  casePageContext,
  caseThreadOf,
  threadSubjectOf,
} from "./case-subject";
import { useCaseDecision } from "./use-case-decision";
import "./case.css";

/** How often the case is read again while something on it is in motion. */
const MOVING_REFETCH_MS = 2000;

// The shipment's own upload, read only when a person opens it from a case.
const CasePodUpload = lazy(() => import("./case-pod-upload"));

/** How long the page watches for an agent's work after handing a case to it. */
const HANDED_WATCH_MS = 30 * 1000;

/** True while an agent the person just put on the case has drafted nothing yet. */
function awaitingAgent(data: DeskCase | undefined, handedAt: number | null): boolean {
  return (
    handedAt !== null &&
    Date.now() - handedAt < HANDED_WATCH_MS &&
    data !== undefined &&
    data.decision === null &&
    data.options.length === 0
  );
}

/** True while the server is doing something the page is waiting to draw. */
function inMotion(data: DeskCase | undefined): boolean {
  if (!data) {
    return false;
  }
  const phase = data.decision?.phase;

  return (
    data.drafting ||
    phase === "Undoable" ||
    phase === "Running" ||
    data.finding?.readingStatus === "Writing" ||
    (data.handOff?.working === true && data.decision === null && data.options.length === 0)
  );
}

/**
 * A record's case: one record and every open signal about it, read as one
 * decision. Its head says what the record is and how long is left; under it,
 * what was found, each claim numbered to the exhibit it rests on, and why it
 * can be acted on; what could happen; and the plan to approve or decline as
 * written. The evidence runs down the side, and the box at the foot asks
 * Nova in the margin with the record attached. The page serves a late
 * shipment, an invoice past due and a disputed invoice alike: the server
 * decides what a case of each kind is made of.
 */
export function DeskRecordCase({ kind, id }: { kind: string; id: string }) {
  const t = useT();
  const desk = useDesk();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: mayResolve } = usePermission(Resource.Watchtower, Operation.Update);
  const { allowed: mayDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const preferredAgent = useDeskSetting("agent");
  // Opened from the Watchtower to be worked: the person takes the case as it
  // opens. No agent starts on it; that is theirs to ask for.
  const startOnOpen = startsCase(useLocation().state);

  // When the person handed the case to an agent from this page: the page
  // watches for what the agent drafts until it lands or the wait runs out.
  const handedAt = useRef<number | null>(null);
  const query = useQuery({
    ...deskCase.detail(kind, id, timezone),
    enabled: canWatch,
    refetchInterval: (current) =>
      inMotion(current.state.data) || awaitingAgent(current.state.data, handedAt.current)
        ? MOVING_REFETCH_MS
        : false,
  });
  const data = query.data;
  // A case someone else owns is read, not decided: its owner, or whoever
  // covers for them, works it.
  const ownCase = data?.ownership.canWork ?? true;
  const canDecide = mayDecide && ownCase;
  const canResolve = mayResolve && ownCase;
  const decision = data?.decision ?? null;
  const state = useCaseDecision(decision);
  const [lit, setLit] = useState<number | null>(null);
  const [declining, setDeclining] = useState(false);
  const [accepting, setAccepting] = useState(false);
  const [uploadingPod, setUploadingPod] = useState(false);
  const [acceptBusy, setAcceptBusy] = useState(false);
  const [acceptFailure, setAcceptFailure] = useState("");
  const [choosing, setChoosing] = useState<string | null>(null);
  const [chooseFailure, setChooseFailure] = useState("");
  const sending = useRef(false);
  const label = data?.record.label ?? "";
  const caseAgentId = decision?.agent?.id ?? data?.finding?.agent?.id ?? null;
  // The record's own page is offered only to someone who may read the record.
  const pageLink =
    isRecordEntityType(kind) && (data?.record.readable ?? false) ? recordPath(kind, id) : null;
  const { allowed: canFile } = usePermission(Resource.Document, Operation.Create);
  const chosen = data?.options.find((option) => option.chosen) ?? null;
  const agentName = decision?.agent?.name ?? data?.finding?.agent?.name ?? null;
  const optionsOpen =
    decision === null || decision.phase === "Awaiting" || decision.phase === "Declined";

  // The case is what ⌘J and Ask take with them, and it joins Open work.
  const setPageSubject = useDeskMarginStore((store) => store.setPageSubject);
  const setCasePage = useDeskMarginStore((store) => store.setCasePage);
  const noteOpenWork = useDeskMarginStore((store) => store.noteOpenWork);
  useEffect(() => {
    if (label === "") {
      return;
    }
    const subject = { kind, id, label, agentId: caseAgentId ?? undefined };
    setPageSubject(subject);
    setCasePage(subject);
    noteOpenWork(subject);
  }, [caseAgentId, id, kind, label, noteOpenWork, setCasePage, setPageSubject]);
  useEffect(
    () => () => {
      setPageSubject(null);
      setCasePage(null);
    },
    [setCasePage, setPageSubject],
  );

  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: deskCase._def }),
      queryClient.invalidateQueries({ queryKey: deskAttention._def }),
      queryClient.invalidateQueries({ queryKey: watchtower._def }),
      queryClient.invalidateQueries({ queryKey: attention._def }),
    ]);

  useClaimOnOpen({
    kind,
    id,
    ownership: data?.ownership,
    enabled: startOnOpen,
    onClaimed: refresh,
  });

  const choose = async (option: DeskCaseOption, redraft = false) => {
    if (sending.current || !canDecide || (option.chosen && !redraft)) {
      return;
    }
    sending.current = true;
    setChoosing(option.id);
    setChooseFailure("");
    try {
      await chooseDeskCaseOption(kind, id, option.id);
    } catch (error) {
      setChooseFailure(describeError(error).message);
    } finally {
      await refresh();
      sending.current = false;
      setChoosing(null);
    }
  };

  // Putting a case away takes it out of Needs you for everyone (its open
  // conditions are snoozed) and off this person's rail: their conversation on
  // the case is put away with it, and a row only this session added is
  // dropped. Bringing it back undoes both.
  const [awayBusy, setAwayBusy] = useState(false);
  const caseThread = caseThreadOf(desk.threads, kind, id, caseAgentId);
  const moveAway = async (until: number | null) => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    setAwayBusy(true);
    try {
      if (until === null) {
        await bringBackDeskCase(kind, id);
        if (caseThread?.case?.state === "Snoozed") {
          await assistantService.wakeCase(caseThread.id);
        }
      } else {
        await putAwayDeskCase(kind, id, until);
        if (caseThread?.case) {
          await assistantService.snoozeCase(caseThread.id, { anchor: "Time", until });
        }
        useDeskMarginStore.getState().forgetOpenWork(kind, id);
      }
    } catch (error) {
      handleMutationError({ error, resourceName: "Case" });
    } finally {
      await Promise.all([
        refresh(),
        queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey }),
      ]);
      sending.current = false;
      setAwayBusy(false);
    }
  };
  // The person has done what they can for now: the plan ran or was declined,
  // or the agent looked and proposed nothing. With something still open, the
  // case can be put away rather than left sitting in front.
  // What the agent on the case is writing, followed while one works it: the
  // run the case was handed to, or the one drafting the chosen option.
  const writing = useCaseWriting(
    kind,
    id,
    data !== undefined && (data.drafting || (data.handOff?.working ?? false)),
    () => void refresh(),
  );
  const restsForNow =
    decision !== null
      ? decision.phase === "Done" || decision.phase === "Declined"
      : (data?.handOff?.note ?? "") !== "";
  const canPutAway =
    canResolve && data !== undefined && data.away === null && data.open > 0 && restsForNow;

  const accept = async (reason: string) => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    setAcceptBusy(true);
    setAcceptFailure("");
    try {
      await acceptDeskCaseRisk(kind, id, reason);
      setAccepting(false);
    } catch (error) {
      setAcceptFailure(describeError(error).message);
    } finally {
      await refresh();
      sending.current = false;
      setAcceptBusy(false);
    }
  };

  const ask = (question: string, agentId?: string, extras?: DeskStartExtras) => {
    const trimmed = question.trim();
    if (trimmed === "" || label === "") {
      return;
    }
    const subject = { kind, id, label, agentId: caseAgentId ?? undefined };
    const caseType = threadSubjectOf(kind);
    const existing = caseThreadOf(desk.threads, kind, id, caseAgentId);
    const margin = useDeskMarginStore.getState();
    // The margin may be holding another conversation (another agent's, or
    // another record's); only the case's own is continued.
    const open = existing?.id ?? null;
    if (open !== null && (extras?.files.length ?? 0) === 0) {
      useDeskStore.getState().askAgent(open, trimmed);
      margin.showThread(open, subject);
      return;
    }
    // The agent on the case is the only one asked about it, whether it is
    // one people chat with or one that runs on its own. A record with no
    // agent on it takes the composer's pick.
    const target =
      caseAgentId ??
      agentId ??
      desk.agents.find((choice) => choice.id === preferredAgent)?.id ??
      desk.agents[0]?.id;
    margin.openMargin(subject);
    if (!target) {
      return;
    }
    desk.startInMargin(
      target,
      trimmed,
      {
        files: extras?.files ?? [],
        providerId: extras?.providerId ?? "",
        mentions: [{ type: kind, id, label }, ...(extras?.mentions ?? [])],
      },
      {
        subjectType: caseType ?? undefined,
        subjectId: caseType ? id : undefined,
        page: casePageContext(casePath(kind, id), kind, id, label),
      },
    );
  };

  /** "Change values" is said in words: the box at the foot takes it to the agent. */
  const focusAsk = () => {
    const box = document.querySelector<HTMLElement>('#dk-cz-ask [role="textbox"]');
    box?.focus();
    box?.scrollIntoView({ block: "nearest" });
  };

  // ⌘Enter approves the plan in front of the person, D declines it, and
  // 1–3 choose among what could happen.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.defaultPrevented || declining || accepting || !canDecide || !data) {
        return;
      }
      const awaiting = decision?.phase === "Awaiting";
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        // Words in the box at the foot are a question to send, not a plan to approve.
        const box = event.target instanceof HTMLElement ? event.target : null;
        if (box?.isContentEditable && box.textContent?.trim() !== "") {
          return;
        }
        if (awaiting && !state.blocked && state.busy === null && state.steps.length > 0) {
          event.preventDefault();
          void state.approve();
        }
        return;
      }
      if (event.metaKey || event.ctrlKey || event.altKey || isTypingTarget(event.target)) {
        return;
      }
      if (event.key.toLowerCase() === "d" && awaiting && state.busy === null) {
        event.preventDefault();
        setDeclining(true);
        return;
      }
      if (/^[1-3]$/.test(event.key) && optionsOpen) {
        const option = data.options[Number(event.key) - 1];
        if (option) {
          event.preventDefault();
          void choose(option);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  // An invoice in dispute is worked as its dispute: the server answers with
  // the case that holds the plan, and the address follows it.
  if (data && (data.record.kind !== kind || data.record.id !== id)) {
    return <Navigate replace to={casePath(data.record.kind, data.record.id)} />;
  }

  if (!canWatch) {
    return (
      <div className="dk-cz-scroll">
        <div className="dk-cz">
          <div className="dk-cz-main">
            <div className="dk-w-banner">
              <DeskIcon name="lock" size={14} />
              <span>{t("You need access to Needs you to read a record's case.")}</span>
            </div>
          </div>
        </div>
      </div>
    );
  }
  if (query.isPending) {
    return <CaseSkeleton />;
  }
  if (query.isError || !data) {
    return (
      <div className="dk-cz-scroll">
        <div className="dk-cz">
          <div className="dk-cz-main">
            <div className="dk-w-banner">
              <DeskIcon name="alert" size={14} />
              <span>
                <b className="font-medium">{t("This case could not be read.")}</b>{" "}
                {query.error ? describeError(query.error).message : ""}{" "}
                <Link className="dk-w-alnk ui-focus-ring" to="/desk/needs">
                  {t("Back to Needs you")}
                </Link>
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Why the reader cannot act on this case, said where the actions would be:
  // someone else owns it, or their role reads cases without working them.
  const locked = !ownCase
    ? t(
        "{0} owns this case, so only they can act on it. You can read it and ask about it; a case lead can give it to you.",
        data.ownership.owner?.name ?? t("Someone else"),
      )
    : !mayDecide && !mayResolve
      ? t("Your role can read this case but not act on it.")
      : "";
  const chosenOption = data.options.find((option) => option.chosen) ?? null;
  const lockedInPlan =
    !data.drafting &&
    chosenOption?.kind !== "Leave" &&
    ((data.decision?.phase === "Awaiting" && !canDecide) ||
      (data.decision === null &&
        chosenOption === null &&
        data.handOff !== null &&
        data.options.length === 0 &&
        !data.handOff.working &&
        !canResolve));

  const unchanged = data.finding?.trust.find((cell) => cell.key === "Unchanged")?.text ?? "";

  return (
    <div className={cn("dk-cz-page", CASE_TRANSITION_CLASS)}>
      {/* With no evidence there is no second column: the reading takes the page. */}
      <div className={cn("dk-cz animate-rise", data.exhibits.length === 0 && "grid-cols-1!")}>
        <ScrollArea className="dk-cz-col" maskVariant="card">
          <div className="dk-cz-main">
            <CaseHead
              data={data}
              now={now}
              recordPath={pageLink}
              owner={
                <>
                  <CaseOwner
                    kind={kind}
                    id={id}
                    label={label}
                    ownership={data.ownership}
                    onChanged={refresh}
                  />
                  {canPutAway && (
                    <CasePutAway
                      now={now}
                      pending={awayBusy}
                      onPutAway={(until) => void moveAway(until)}
                    />
                  )}
                </>
              }
            />
            {data.away && (
              <CaseAwayNotice
                until={data.away.until}
                timezone={timezone}
                canBringBack={canResolve}
                pending={awayBusy}
                onBringBack={() => void moveAway(null)}
              />
            )}
            {/* Said once, where the actions would be. A case with no plan
                card to say it in says it here. */}
            {locked !== "" && !lockedInPlan && (
              <p className="dk-cz-ro" role="status">
                <DeskIcon name="lock" size={13} />
                {locked}
              </p>
            )}
            {data.finding === null && data.decision === null && data.open === 0 ? (
              <CaseClosed recordId={id} label={label} recordPath={pageLink} />
            ) : (
              <CaseFinding data={data} timezone={timezone} lit={lit} onLight={setLit} />
            )}

            <CaseOptions
              options={data.options}
              disabled={!optionsOpen || !canDecide || state.busy !== null}
              pendingId={choosing}
              onChoose={(option) => void choose(option)}
            />
            {chooseFailure !== "" && (
              <div className="dk-w-banner">
                <DeskIcon name="alert" size={14} />
                <span>{chooseFailure}</span>
              </div>
            )}

            {chosen?.kind === "Leave" ? (
              <CasePlanNone
                title={
                  unchanged !== "" ? t("Nothing changes. {0}", unchanged) : t("Nothing changes.")
                }
                note={
                  data.open > 0
                    ? t(
                        "Nothing is drafted for this choice. Mark the risk accepted to clear it from Needs you, with a reason the team can read.",
                      )
                    : t("Nothing is drafted for this choice, and nothing is open about the record.")
                }
                action={
                  canResolve && data.open > 0
                    ? {
                        label: t("Accept the risk…"),
                        pending: false,
                        onClick: () => setAccepting(true),
                      }
                    : null
                }
              />
            ) : data.drafting || choosing !== null ? (
              <CasePlanDrafting agentName={agentName ?? t("The agent")} writing={writing} />
            ) : decision ? (
              <CasePlan
                data={data}
                decision={decision}
                state={state}
                canDecide={canDecide}
                locked={locked}
                timezone={timezone}
                recordPath={pageLink}
                onDecline={() => setDeclining(true)}
                onChange={focusAsk}
                onAsk={ask}
              />
            ) : chosen ? (
              <CasePlanNone
                title={
                  data.draftNote !== ""
                    ? t(
                        "{0} finished without drafting a plan for this option.",
                        agentName ?? t("The agent"),
                      )
                    : t("No plan is drafted for this option yet.")
                }
                note={
                  data.draftNote !== "" ? (
                    <AiMarkdown
                      content={withoutRecordIds(data.draftNote)}
                      className="text-xs"
                      deskSubset
                    />
                  ) : (
                    t("The agent drafts it against the record as it is now, for you to approve.")
                  )
                }
                action={
                  canDecide
                    ? {
                        label: data.draftNote !== "" ? t("Try again") : t("Draft the plan"),
                        pending: choosing !== null,
                        onClick: () => void choose(chosen, true),
                      }
                    : null
                }
              />
            ) : data.handOff && data.options.length === 0 ? (
              <CaseUnworked
                key={data.handOff.itemId}
                handOff={data.handOff}
                canHandOff={canResolve}
                locked={locked}
                writing={writing}
                agentRaised={data.handOff.kind === "AgentException"}
                onHanded={() => {
                  handedAt.current = Date.now();
                  void refresh();
                }}
                onAcceptRisk={canResolve && data.open > 0 ? () => setAccepting(true) : null}
                actions={
                  data.handOff.kind === "PodMissing" && kind === "shipment" && canFile
                    ? [
                        // Filing a paper takes document create, whoever owns the case.
                        { label: t("Upload the POD"), onClick: () => setUploadingPod(true) },
                      ]
                    : undefined
                }
              />
            ) : null}
            {decision && decision.others > 0 && (
              <div className="dk-cz-plan-more">
                {t(
                  "{0, plural, one {# other decision waits} other {# other decisions wait}} on this record.",
                  decision.others,
                )}{" "}
                <Link className="dk-w-alnk ui-focus-ring" to="/desk/needs?lens=decisions">
                  {t("See them in Needs you")}
                </Link>
              </div>
            )}
          </div>
        </ScrollArea>

        {data.exhibits.length > 0 && (
          <CaseExhibits data={data} lit={lit} now={now} timezone={timezone} />
        )}
      </div>

      <CaseDock
        space={recordSpace(kind, id)}
        label={label}
        starters={data.starters}
        agentId={caseAgentId}
        agentName={agentName}
        onAsk={ask}
      />

      <DeclineDialog
        open={declining}
        agentName={decision?.agent?.name ?? null}
        pending={state.busy === "decline"}
        onOpenChange={setDeclining}
        onDecline={async (reason) => {
          await state.decline(reason);
          setDeclining(false);
        }}
      />
      {uploadingPod && (
        <Suspense fallback={null}>
          <CasePodUpload
            shipmentId={id}
            open={uploadingPod}
            onClose={() => setUploadingPod(false)}
            onUploaded={() => void refresh()}
          />
        </Suspense>
      )}
      <AcceptRiskDialog
        open={accepting}
        label={label}
        pending={acceptBusy}
        failure={acceptFailure}
        onOpenChange={setAccepting}
        onAccept={accept}
      />
    </div>
  );
}
