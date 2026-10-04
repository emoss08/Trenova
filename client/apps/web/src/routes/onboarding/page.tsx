import logoRainbow from "@/assets/logo.webp";
import { Metadata } from "@/components/metadata";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSignOut } from "@/hooks/use-sign-out";
import {
  browserTimezone,
  ONBOARDING_STEPS,
  onboardingFormDefaults,
  type OnboardingStepId,
} from "@/lib/onboarding-form";
import { onboardingStateQueryOptions } from "@/lib/queries/onboarding";
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
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Stepper, type StepperStep } from "@trenova/shared/components/ui/stepper";
import { AlertCircleIcon, ArrowLeftIcon, ArrowRightIcon } from "@trenova/shared/components/icons";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { CompanyStep } from "./_components/company-step";
import { OperationStep } from "./_components/operation-step";
import { ReviewStep } from "./_components/review-step";
import { SampleDataStep } from "./_components/sample-data-step";

function currentOrganizationName(
  user: ReturnType<typeof useAuthStore.getState>["user"],
): string | undefined {
  return user?.memberships?.find(
    (membership) => membership.organizationId === user.currentOrganizationId,
  )?.organization?.name;
}

/**
 * The welcome wizard a new Trenova Cloud organization finishes once: profile,
 * operation type, sample data, review. The route loader only lets a pending
 * organization in; finishing it marks the organization complete on the server and
 * sends the person home.
 */
export function OnboardingPage() {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const organizationId = user?.currentOrganizationId ?? "";
  const stateQuery = useQuery(onboardingStateQueryOptions(organizationId));

  return (
    <>
      <Metadata title={t("Welcome")} description={t("Set up your Trenova organization")} />
      <OnboardingFrame>
        {stateQuery.isPending ? (
          <OnboardingSkeleton />
        ) : stateQuery.isError ? (
          <Alert variant="destructive">
            <AlertCircleIcon />
            <AlertTitle>{t("We couldn't load your setup")}</AlertTitle>
            <AlertDescription>
              {t("Refresh the page to try again. Your account and organization are safe.")}
            </AlertDescription>
          </Alert>
        ) : (
          <OnboardingWizard
            state={stateQuery.data}
            organizationId={organizationId}
            organizationName={currentOrganizationName(user)}
          />
        )}
      </OnboardingFrame>
    </>
  );
}

function OnboardingFrame({ children }: { children: React.ReactNode }) {
  const t = useT();
  const signOut = useSignOut();

  return (
    <div className="bg-canvas text-foreground min-h-svh">
      <header className="border-border bg-card flex h-12 items-center justify-between border-b px-4">
        <div className="flex items-center gap-2.5">
          <img src={logoRainbow} alt="" className="size-5 object-contain" />
          <span className="text-base font-semibold">{t("Trenova")}</span>
        </div>
        <Button variant="ghost" size="sm" onClick={() => void signOut()}>
          {t("Sign out")}
        </Button>
      </header>
      <main className="mx-auto w-full max-w-4xl px-4 py-8 sm:py-12">{children}</main>
    </div>
  );
}

function OnboardingWizard({
  state,
  organizationId,
  organizationName,
}: {
  state: OnboardingState;
  organizationId: string;
  organizationName: string | undefined;
}) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const checkAuth = useAuthStore((store) => store.checkAuth);
  const fetchManifest = usePermissionStore((store) => store.fetchManifest);
  const { config } = usePublicConfig();
  const [stepIndex, setStepIndex] = useState(0);
  const [stateLabel, setStateLabel] = useState("");

  const defaultValues = useMemo(
    () =>
      onboardingFormDefaults({
        state,
        organizationName,
        fallbackTimezone: browserTimezone(),
      }),
    [state, organizationName],
  );

  const form = useForm<OnboardingFormValues, unknown, OnboardingFormOutput>({
    resolver: zodResolver(onboardingFormSchema),
    defaultValues,
    mode: "onTouched",
  });
  const { handleSubmit, trigger, control } = form;

  const step = ONBOARDING_STEPS[stepIndex];
  const isLast = stepIndex === ONBOARDING_STEPS.length - 1;

  // A refusal (from zod on the last step, or from the server) can belong to a field on
  // an earlier step; the wizard goes back to the first step that holds one, so the
  // message is on screen rather than behind a "Back" button.
  // getFieldState reads the live error store; form.formState is the snapshot of the last
  // render, which does not yet hold the errors this very submit just set.
  const showFirstInvalidStep = () => {
    const firstInvalid = ONBOARDING_STEPS.findIndex((candidate) =>
      candidate.fields.some((field) => form.getFieldState(field).error !== undefined),
    );
    if (firstInvalid !== -1) {
      setStepIndex(firstInvalid);
    }
  };

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: (values: OnboardingFormOutput) =>
      onboardingService.complete(toCompleteOnboardingRequest(values)),
    form,
    resourceName: "Onboarding",
    onError: showFirstInvalidStep,
  });

  const steps: StepperStep[] = ONBOARDING_STEPS.map((definition, index) => ({
    id: definition.id,
    label: t(definition.label),
    detail: t(definition.detail),
    state: index < stepIndex ? "done" : index === stepIndex ? "active" : "pending",
  }));

  const goNext = async () => {
    const valid = step.fields.length === 0 || (await trigger([...step.fields]));
    if (valid) {
      setStepIndex((current) => Math.min(current + 1, ONBOARDING_STEPS.length - 1));
    }
  };

  const goTo = (id: OnboardingStepId) => {
    const index = ONBOARDING_STEPS.findIndex((definition) => definition.id === id);
    if (index !== -1) {
      setStepIndex(index);
    }
  };

  const complete = async (values: OnboardingFormOutput) => {
    let completed: OnboardingState | null;
    try {
      completed = await mutateAsync(values);
    } catch {
      return;
    }

    // Written before navigating so the protected loader, which reads this entry, sees
    // the organization as finished and does not send the person straight back.
    queryClient.setQueryData(onboardingStateQueryOptions(organizationId).queryKey, {
      ...(completed ?? state),
      status: "completed",
    });
    // The profile, the capability flags and possibly a sample data set all changed, so
    // nothing cached under the old organization profile is worth keeping.
    await Promise.allSettled([checkAuth(), fetchManifest()]);
    await queryClient.invalidateQueries({
      predicate: (query) => query.queryKey[0] !== "onboarding",
    });
    void navigate("/", { replace: true });
  };

  return (
    <FormProvider {...form}>
      <div className="mb-8 flex flex-col gap-1">
        <h1 className="m-0 text-2xl font-semibold">{t("Welcome to Trenova")}</h1>
        <p className="text-muted-foreground m-0 text-sm">
          {t(
            "A few details and your workspace is ready. Everything here can be changed later in Organization settings.",
          )}
        </p>
      </div>

      <div className="grid gap-8 sm:grid-cols-[12rem_minmax(0,1fr)]">
        <Stepper steps={steps} aria-label={t("Setup steps")} className="hidden sm:flex" />
        <p className="text-muted-foreground m-0 text-xs sm:hidden">
          {t("Step {0} of {1}", String(stepIndex + 1), String(ONBOARDING_STEPS.length))} ·{" "}
          {t(step.label)}
        </p>

        <Form
          onSubmit={(event) => {
            if (!isLast) {
              event.preventDefault();
              void goNext();
              return;
            }
            void handleSubmit(complete, showFirstInvalidStep)(event);
          }}
          className="flex min-w-0 flex-col gap-4"
        >
          {step.id === "company" ? (
            <CompanyStep control={control} onStateLabelChange={setStateLabel} />
          ) : step.id === "operation" ? (
            <OperationStep control={control} />
          ) : step.id === "sample-data" ? (
            <SampleDataStep control={control} freePlanLimits={config.freePlan.limits} />
          ) : (
            <ReviewStep control={control} stateLabel={stateLabel} onEdit={goTo} />
          )}

          <div className="flex items-center justify-between gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setStepIndex((current) => Math.max(0, current - 1))}
              disabled={stepIndex === 0 || isPending}
            >
              <ArrowLeftIcon className="size-4" />
              {t("Back")}
            </Button>
            <Button type="submit" isLoading={isPending} loadingText={t("Setting up")}>
              {isLast ? t("Finish setup") : t("Continue")}
              {isLast ? null : <ArrowRightIcon className="size-4" />}
            </Button>
          </div>
        </Form>
      </div>
    </FormProvider>
  );
}

function OnboardingSkeleton() {
  return (
    <div className="grid gap-8 sm:grid-cols-[12rem_minmax(0,1fr)]">
      <div className="hidden flex-col gap-4 sm:flex">
        {ONBOARDING_STEPS.map((step) => (
          <Skeleton key={step.id} className="h-8" />
        ))}
      </div>
      <Skeleton className="h-96" />
    </div>
  );
}
