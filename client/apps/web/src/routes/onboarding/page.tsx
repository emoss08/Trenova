import "./_components/onboarding.css";
import { Metadata } from "@/components/metadata";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fetchGraphQLSelectOptions, type SelectOption } from "@/lib/graphql/select-options";
import {
  BOLD_MARKS,
  boldSegments,
  firstNameOf,
  novaLine,
  plainSegments,
  segmentsText,
  type NovaSegment,
} from "@/lib/onboarding-copy";
import {
  browserTimezone,
  firstInvalidOnboardingStep,
  nextOnboardingStep,
  ONBOARDING_STEPS,
  onboardingFormDefaults,
  onboardingProgress,
  OPERATION_TYPES,
  operationTypeDefinition,
  REVIEW_STEP_INDEX,
  SAMPLE_DATA_RECORD_COUNT,
  onboardingTimezoneLabel,
  type OnboardingStepId,
} from "@/lib/onboarding-form";
import { onboardingStateQueryOptions } from "@/lib/queries/onboarding";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { onboardingService } from "@/services/onboarding";
import {
  onboardingFormSchema,
  toCompleteOnboardingRequest,
  type OnboardingFormOutput,
  type OnboardingFormValues,
  type OnboardingState,
} from "@/types/onboarding";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Edit02Icon } from "@trenova/shared/components/icons";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { FormProvider, useForm, useWatch, type Control } from "react-hook-form";
import { useNavigate } from "react-router";
import { AddressAsk } from "./_components/address-ask";
import { BackButton } from "./_components/back-button";
import { BuildNarration, type BuildOutcome } from "./_components/build-narration";
import { ChoiceAsk } from "./_components/choice-ask";
import { ComposerAsk } from "./_components/composer-ask";
import { IdsAsk } from "./_components/ids-ask";
import { Kbd } from "./_components/keyboard-hint";
import { NovaMessage } from "./_components/nova-message";
import { ReadyCard } from "./_components/ready-card";
import { ReviewCard, type ReviewGroup } from "./_components/review-card";
import { ModuleChips, SampleMeters } from "./_components/sample-meters";
import { TimezoneAsk } from "./_components/timezone-ask";

type Phase = "setup" | "building" | "ready" | "failed";
type Position = { current: number; furthest: number };

const BUILD_TURN = "build";
const FAILURE_TURN = "failure";
const REWIND_DELAY_MS = 900;
const STATE_LABEL_STALE_MS = 5 * 60 * 1000;

function currentOrganizationName(
  user: ReturnType<typeof useAuthStore.getState>["user"],
): string | undefined {
  return user?.memberships?.find(
    (membership) => membership.organizationId === user.currentOrganizationId,
  )?.organization?.name;
}

/**
 * The welcome a new Trenova Cloud organization finishes once, as a conversation with
 * Nova, a scripted setup guide. The route loader only lets a pending organization in;
 * finishing it marks the organization complete on the server and opens the app.
 */
export function OnboardingPage() {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const organizationId = user?.currentOrganizationId ?? "";
  const stateQuery = useQuery(onboardingStateQueryOptions(organizationId));

  return (
    <>
      <Metadata title={t("Welcome")} description={t("Set up your Trenova organization")} />
      {stateQuery.isPending ? (
        <OnboardingShell progress={4}>
          <section className="nv-turn">
            <NovaPending />
          </section>
        </OnboardingShell>
      ) : stateQuery.isError ? (
        <OnboardingShell progress={4}>
          <section className="nv-turn">
            <NovaMessage
              animate={false}
              segments={plainSegments(
                t(
                  "I couldn't load your setup. Refresh the page to try again. Your account and organization are safe.",
                ),
              )}
            />
          </section>
        </OnboardingShell>
      ) : (
        <Conversation
          state={stateQuery.data}
          organizationId={organizationId}
          organizationName={currentOrganizationName(user)}
          firstName={firstNameOf(user?.name)}
        />
      )}
    </>
  );
}

function NovaPending() {
  const t = useT();
  return (
    <div>
      <div className="nv-who">
        <span className="nv-mark" data-busy="true" aria-hidden="true" />
        <b>{t("Nova")}</b>
        <span>{t("Setup guide")}</span>
      </div>
      <div className="nv-thk">{t("Typing")}</div>
    </div>
  );
}

function OnboardingShell({
  progress,
  back,
  live,
  flowRef,
  scrollRef,
  rootRef,
  children,
}: {
  progress: number;
  back?: ReactNode;
  live?: string;
  flowRef?: React.Ref<HTMLElement>;
  scrollRef?: React.Ref<HTMLDivElement>;
  rootRef?: React.Ref<HTMLDivElement>;
  children: ReactNode;
}) {
  return (
    <div className="nv" ref={rootRef}>
      {back}
      <span className="nv-tbar" aria-hidden="true">
        <i style={{ width: `${progress}%` }} />
      </span>
      <div className="nv-live" aria-live="polite" aria-atomic="true">
        {live}
      </div>
      <div className="nv-room">
        <div className="nv-scroll" ref={scrollRef}>
          <div className="nv-grid">
            <main className="nv-flow" ref={flowRef}>
              {children}
            </main>
          </div>
        </div>
      </div>
    </div>
  );
}

function useStateOption(stateId: string, picked: SelectOption | null) {
  const usePicked = picked !== null && picked.id === stateId;
  const query = useQuery({
    queryKey: ["select-option-labels", "US_STATE", [stateId]],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: "US_STATE", ids: [stateId], initialLimit: 1 },
        { signal },
      ),
    enabled: stateId !== "" && !usePicked,
    staleTime: STATE_LABEL_STALE_MS,
  });
  const option = usePicked
    ? picked
    : (query.data?.results.find((candidate) => candidate.id === stateId) ?? null);
  return {
    name: option?.label ?? "",
    abbreviation: option ? selectOptionMetaString(option, "abbreviation") || option.label : "",
  };
}

type Answers = {
  company: string;
  timezone: string;
  address: string;
  cityState: string;
  ids: string;
  operation: string;
  operationLower: string;
  operationModules: number;
  sample: string;
};

function useAnswers(
  control: Control<OnboardingFormValues, unknown, OnboardingFormOutput>,
  pickedState: SelectOption | null,
  t: TranslateFn,
): { values: OnboardingFormValues; answers: Answers } {
  const values = useWatch({ control }) as OnboardingFormValues;
  const organization = values.organization;
  const state = useStateOption(organization.stateId ?? "", pickedState);
  const operation = operationTypeDefinition(values.operationType);
  const operationLabel = operation ? t(operation.label) : "";

  const scac = (organization.scacCode ?? "").trim().toUpperCase();
  const dot = (organization.dotNumber ?? "").trim();
  const zipLine = [state.abbreviation, (organization.postalCode ?? "").trim()]
    .filter(Boolean)
    .join(" ");

  return {
    values,
    answers: {
      company: (organization.name ?? "").trim(),
      timezone: organization.timezone ? t(onboardingTimezoneLabel(organization.timezone)) : "",
      address: [(organization.addressLine1 ?? "").trim(), (organization.city ?? "").trim(), zipLine]
        .filter(Boolean)
        .join(", "),
      cityState: [(organization.city ?? "").trim(), state.abbreviation].filter(Boolean).join(", "),
      ids: [scac && t("SCAC {0}", scac), dot && t("USDOT {0}", dot)].filter(Boolean).join(" · "),
      operation: operationLabel,
      operationLower: operationLabel.toLocaleLowerCase(),
      operationModules: operation?.modules.length ?? 0,
      sample: values.loadSampleData ? t("Load sample data") : t("Start empty"),
    },
  };
}

function answerFor(id: OnboardingStepId, answers: Answers): string {
  switch (id) {
    case "name":
      return answers.company;
    case "timezone":
      return answers.timezone;
    case "address":
      return answers.address;
    case "ids":
      return answers.ids;
    case "operation":
      return answers.operation;
    case "sample-data":
      return answers.sample;
    case "review":
      return "";
  }
}

function Conversation({
  state,
  organizationId,
  organizationName,
  firstName,
}: {
  state: OnboardingState;
  organizationId: string;
  organizationName: string | undefined;
  firstName: string;
}) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const checkAuth = useAuthStore((store) => store.checkAuth);
  const fetchManifest = usePermissionStore((store) => store.fetchManifest);
  const { config } = usePublicConfig();
  const [detected] = useState(browserTimezone);
  const [position, setPosition] = useState<Position>({ current: 0, furthest: 0 });
  const [seen, setSeen] = useState<ReadonlySet<string>>(() => new Set());
  const [announcement, setAnnouncement] = useState("");
  const [phase, setPhase] = useState<Phase>("setup");
  const [outcome, setOutcome] = useState<BuildOutcome>("pending");
  const [settledAt, setSettledAt] = useState<number | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [rewinding, setRewinding] = useState(false);
  const [opening, setOpening] = useState(false);
  const [pickedState, setPickedState] = useState<SelectOption | null>(null);
  const completedRef = useRef<OnboardingState | null>(null);
  const finishingRef = useRef(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const flowRef = useRef<HTMLElement>(null);
  const { current, furthest } = position;

  const defaultValues = useMemo(
    () =>
      onboardingFormDefaults({
        state,
        organizationName,
        fallbackTimezone: detected,
      }),
    [state, organizationName, detected],
  );

  const form = useForm<OnboardingFormValues, unknown, OnboardingFormOutput>({
    resolver: zodResolver(onboardingFormSchema),
    defaultValues,
    mode: "onTouched",
  });
  const { handleSubmit, trigger, setValue, getFieldState } = form;
  const { values, answers } = useAnswers(form.control, pickedState, t);

  const { mutate } = useApiMutation({
    mutationFn: (submitted: OnboardingFormOutput) =>
      onboardingService.complete(toCompleteOnboardingRequest(submitted)),
    form,
    resourceName: "Onboarding",
  });

  useEffect(() => {
    const flow = flowRef.current;
    const scroller = scrollRef.current;
    if (!flow || !scroller || typeof ResizeObserver === "undefined") {
      return undefined;
    }
    const observer = new ResizeObserver(() => {
      scroller.scrollTo({ top: scroller.scrollHeight, behavior: "smooth" });
    });
    observer.observe(flow);
    return () => observer.disconnect();
  }, []);

  const seenRef = useRef(seen);

  const markSeen = useCallback((id: string, text: string) => {
    if (seenRef.current.has(id)) {
      return;
    }
    const next = new Set(seenRef.current).add(id);
    seenRef.current = next;
    setSeen(next);
    setAnnouncement(text);
  }, []);

  const submitStep = (index: number) => async () => {
    const step = ONBOARDING_STEPS[index];
    const valid = step.fields.length === 0 || (await trigger([...step.fields]));
    if (!valid) {
      return false;
    }
    setPosition((previous) => {
      const next = nextOnboardingStep(previous.current, previous.furthest);
      return { current: next, furthest: Math.max(previous.furthest, next) };
    });
    return true;
  };

  const back = useCallback(() => {
    if (phase !== "setup") {
      return;
    }
    setPosition((previous) =>
      previous.current === 0
        ? previous
        : { current: previous.current - 1, furthest: previous.current - 1 },
    );
  }, [phase]);

  const edit = (index: number) => {
    if (phase !== "setup") {
      return;
    }
    setPosition((previous) => ({ current: index, furthest: previous.furthest }));
  };

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) {
        return;
      }
      const target = event.target;
      const inside =
        target === document.body ||
        (target instanceof Node && rootRef.current?.contains(target) === true);
      if (inside) {
        back();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [back]);

  const rewindToFirstInvalid = useCallback(() => {
    const index = firstInvalidOnboardingStep((field) => getFieldState(field).error !== undefined);
    if (index === -1) {
      return false;
    }
    setPhase("setup");
    setPosition({ current: index, furthest: REVIEW_STEP_INDEX });
    return true;
  }, [getFieldState]);

  const startBuild = (submitted: OnboardingFormOutput) => {
    completedRef.current = null;
    setRewinding(false);
    setOutcome("pending");
    setSettledAt(null);
    setAttempt((previous) => previous + 1);
    setPhase("building");
    mutate(submitted, {
      onSuccess: (completed) => {
        completedRef.current = completed;
        setSettledAt(performance.now());
        setOutcome("success");
      },
      onError: () => setOutcome("error"),
    });
  };

  const finish = () => {
    if (finishingRef.current || phase === "building" || phase === "ready") {
      return;
    }
    finishingRef.current = true;
    void handleSubmit(startBuild, () => {
      finishingRef.current = false;
      rewindToFirstInvalid();
    })();
  };

  const onFailureTyped = useCallback(() => {
    const index = firstInvalidOnboardingStep((field) => getFieldState(field).error !== undefined);
    if (index !== -1) {
      setRewinding(true);
    }
  }, [getFieldState]);

  useEffect(() => {
    if (!rewinding) {
      return undefined;
    }
    const id = window.setTimeout(() => {
      setRewinding(false);
      rewindToFirstInvalid();
    }, REWIND_DELAY_MS);
    return () => window.clearTimeout(id);
  }, [rewinding, rewindToFirstInvalid]);

  const open = async () => {
    setOpening(true);
    queryClient.setQueryData(onboardingStateQueryOptions(organizationId).queryKey, {
      ...(completedRef.current ?? state),
      status: "completed",
    });
    await Promise.allSettled([checkAuth(), fetchManifest()]);
    await queryClient.invalidateQueries({
      predicate: (query) => query.queryKey[0] !== "onboarding",
    });
    void navigate("/", { replace: true });
  };

  const company = answers.company || t("your company");
  const context = {
    firstName,
    company,
    browserZoneLabel: detected ? t(onboardingTimezoneLabel(detected)) : "",
  };
  const settingUp = phase === "setup";
  const visibleSteps = ONBOARDING_STEPS.slice(0, settingUp ? current + 1 : REVIEW_STEP_INDEX + 1);

  const buildLines: NovaSegment[][] = [
    boldSegments(t("Creating {0}", ...BOLD_MARKS), [company]),
    boldSegments(t("Setting the clock to {0}", ...BOLD_MARKS), [answers.timezone]),
    boldSegments(t("Saving {0} as headquarters", ...BOLD_MARKS), [answers.cityState]),
    boldSegments(t("Turning on {0} for {1}", ...BOLD_MARKS), [
      t("{0, plural, one {# module} other {# modules}}", answers.operationModules),
      answers.operationLower,
    ]),
    values.loadSampleData
      ? boldSegments(t("Loading {0}", ...BOLD_MARKS), [
          t(
            "{0, plural, one {# sample record} other {# sample records}}",
            SAMPLE_DATA_RECORD_COUNT,
          ),
        ])
      : plainSegments(t("Leaving records empty")),
    plainSegments(t("Making you the owner")),
  ];

  const reviewGroups: ReviewGroup[] = [
    {
      title: t("Company profile"),
      rows: [
        { label: t("Company name"), value: answers.company, step: 0 },
        { label: t("Timezone"), value: answers.timezone, step: 1 },
        { label: t("Address"), value: answers.address, step: 2 },
        {
          label: t("SCAC code"),
          value: (values.organization.scacCode ?? "").trim().toUpperCase(),
          step: 3,
          mono: true,
        },
        {
          label: t("USDOT number"),
          value: (values.organization.dotNumber ?? "").trim(),
          step: 3,
          mono: true,
        },
      ],
    },
    {
      title: t("Setup"),
      rows: [
        { label: t("Operation type"), value: answers.operation, step: 4 },
        { label: t("Sample data"), value: answers.sample, step: 5 },
      ],
    },
  ];

  const ask = (id: OnboardingStepId, index: number): ReactNode => {
    switch (id) {
      case "name":
        return <ComposerAsk onSubmit={submitStep(index)} />;
      case "timezone":
        return <TimezoneAsk detected={detected} onSubmit={submitStep(index)} />;
      case "address":
        return <AddressAsk onSubmit={submitStep(index)} onStateOptionChange={setPickedState} />;
      case "ids":
        return <IdsAsk onSubmit={submitStep(index)} />;
      case "operation":
        return (
          <ChoiceAsk
            label={t("Operation type")}
            error={getFieldState("operationType").error?.message}
            value={furthest > index ? values.operationType : undefined}
            items={OPERATION_TYPES.map((type) => ({
              value: type.value,
              title: t(type.label),
              description: t(type.description),
              extra: <ModuleChips modules={type.modules} />,
            }))}
            onPick={(value) => {
              setValue("operationType", value, { shouldDirty: true });
              return submitStep(index)();
            }}
          />
        );
      case "sample-data":
        return (
          <ChoiceAsk
            label={t("Sample data")}
            error={getFieldState("loadSampleData").error?.message}
            value={furthest > index ? values.loadSampleData : undefined}
            items={[
              {
                value: true,
                title: t("Load sample data"),
                description: t(
                  "A few customers, locations, equipment and shipments, so every screen has something to show.",
                ),
                extra: <SampleMeters limits={config.freePlan.limits} />,
              },
              {
                value: false,
                title: t("Start empty"),
                description: t("A clean workspace. Add your own records from day one."),
              },
            ]}
            onPick={(value) => {
              setValue("loadSampleData", value, { shouldDirty: true });
              return submitStep(index)();
            }}
          />
        );
      case "review":
        return <ReviewCard groups={reviewGroups} onEdit={edit} onFinish={finish} />;
    }
  };

  const buildLine = boldSegments(
    t("Creating the workspace for {0}. This only takes a moment.", ...BOLD_MARKS),
    [company],
  );
  const failureLine = plainSegments(
    t("Something went wrong while I was setting things up. Nothing was lost — let's fix it."),
  );
  const rootError = form.formState.errors.root?.message;
  const failureNeedsRetry =
    phase === "failed" &&
    seen.has(`${FAILURE_TURN}-${attempt}`) &&
    !rewinding &&
    firstInvalidOnboardingStep((field) => getFieldState(field).error !== undefined) === -1;

  return (
    <FormProvider {...form}>
      <OnboardingShell
        progress={onboardingProgress(current, phase)}
        rootRef={rootRef}
        scrollRef={scrollRef}
        flowRef={flowRef}
        live={announcement}
        back={<BackButton hidden={!settingUp || current === 0} onBack={back} />}
      >
        {visibleSteps.map((step, index) => {
          const isCurrent = settingUp && index === current;
          const answered = (index < current || !settingUp) && step.id !== "review";
          const answer = answered ? answerFor(step.id, answers) : "";
          const segments = novaLine(t, step.id, context);
          return (
            <section key={step.id} className="nv-turn">
              <NovaMessage
                segments={segments}
                animate={!seen.has(step.id)}
                onDone={() => markSeen(step.id, segmentsText(segments))}
              />
              {isCurrent && seen.has(step.id) ? (
                <div className="nv-ask">{ask(step.id, index)}</div>
              ) : null}
              {answered ? (
                <div className="nv-ans" data-last={settingUp && index === current - 1}>
                  {settingUp ? (
                    <button type="button" className="nv-ans-e" onClick={() => edit(index)}>
                      <Edit02Icon size={12} strokeWidth={1.5} aria-hidden="true" />
                      {t("Edit")}
                    </button>
                  ) : null}
                  <span key={answer} className="nv-ans-b" data-skipped={!answer}>
                    {answer || t("Skip for now")}
                  </span>
                </div>
              ) : null}
              {index === 0 && current === 1 && settingUp ? (
                <div className="nv-fix">
                  {boldSegments(
                    t("Made a typo? Click Edit, or press {0} to go back.", ...BOLD_MARKS),
                    ["esc"],
                  ).map((segment, segmentIndex) =>
                    segment.bold ? (
                      <Kbd key={segmentIndex}>{segment.text}</Kbd>
                    ) : (
                      <span key={segmentIndex}>{segment.text}</span>
                    ),
                  )}
                </div>
              ) : null}
              {step.id === "review" && !settingUp ? (
                <div className="nv-ans">
                  <span className="nv-ans-b">{t("Finish setup")}</span>
                </div>
              ) : null}
            </section>
          );
        })}
        {!settingUp ? (
          <section className="nv-turn">
            <NovaMessage
              segments={buildLine}
              animate={!seen.has(BUILD_TURN)}
              onDone={() => markSeen(BUILD_TURN, segmentsText(buildLine))}
            />
            {seen.has(BUILD_TURN) ? (
              <BuildNarration
                key={attempt}
                lines={buildLines}
                outcome={outcome}
                settledAt={settledAt}
                onReady={() => setPhase("ready")}
                onFailed={() => {
                  finishingRef.current = false;
                  setPhase("failed");
                }}
              />
            ) : null}
            {phase === "ready" ? (
              <ReadyCard
                title={firstName ? t("You're all set, {0}", firstName) : t("{0} is ready", company)}
                detail={
                  values.loadSampleData
                    ? t("{0} is live. Sample data is loaded and ready to explore.", company)
                    : t("{0} is live and ready for your first records.", company)
                }
                opening={opening}
                onOpen={() => void open()}
              />
            ) : null}
          </section>
        ) : null}
        {phase === "failed" ? (
          <section className="nv-turn">
            <NovaMessage
              key={attempt}
              segments={failureLine}
              animate={!seen.has(`${FAILURE_TURN}-${attempt}`)}
              onDone={() => {
                markSeen(`${FAILURE_TURN}-${attempt}`, segmentsText(failureLine));
                onFailureTyped();
              }}
            />
            {failureNeedsRetry ? (
              <div className="nv-ask nv-retry">
                {rootError ? (
                  <p className="nv-err" role="alert" style={{ marginBottom: 12 }}>
                    {rootError}
                  </p>
                ) : null}
                <button type="button" className="nv-bt" data-ink="true" onClick={finish}>
                  {t("Try again")}
                </button>
              </div>
            ) : null}
          </section>
        ) : null}
      </OnboardingShell>
    </FormProvider>
  );
}
