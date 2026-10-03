import { PageLayout } from "@/components/navigation/sidebar-layout";
import {
  CAPTURE_PAIRING_CODE_LENGTH,
  captureFailureKind,
  formatPairingCode,
  normalizePairingCode,
} from "@/lib/capture";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { Link, useSearchParams } from "react-router";
import { PairingCodeForm } from "./_components/pairing-code-form";
import { PairingReview, type PairingOutcome } from "./_components/pairing-review";

function LookupFailure({
  error,
  retrying,
  onRetry,
}: {
  error: unknown;
  retrying: boolean;
  onRetry: () => void;
}) {
  const t = useT();

  switch (captureFailureKind(error)) {
    case "forbidden":
      return (
        <Alert variant="warning" size="sm">
          <AlertDescription>
            {t(
              "You do not have permission to pair computers. Ask an administrator for scanning access.",
            )}
          </AlertDescription>
        </Alert>
      );
    case "unreachable":
      return (
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {t("Trenova could not look the code up. Try again in a moment.")}
          </AlertDescription>
          <AlertAction>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={onRetry}
              isLoading={retrying}
            >
              {t("Try again")}
            </Button>
          </AlertAction>
        </Alert>
      );
    case "not-found":
    case "invalid":
      return (
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {t(
              "That code is not valid or has expired. Codes last ten minutes; start again from Trenova Capture's tray icon for a new one.",
            )}
          </AlertDescription>
        </Alert>
      );
  }
}

/**
 * Where a person approves a computer running Trenova Capture.
 *
 * The companion shows a code and opens this page with it filled in. The person
 * sees which machine is asking — its name, its Windows user, where the request
 * came from — before approving, so a code read off somebody else's screen is
 * recognisably not theirs. Approving binds the computer to the person signed
 * in here and the organization they are in; nothing the machine sent decides
 * either.
 */
export function CapturePairPage() {
  const t = useT();
  const [searchParams] = useSearchParams();
  const linkedCode = searchParams.get("code") ?? "";

  const [submitted, setSubmitted] = useState(() => {
    const code = normalizePairingCode(linkedCode);
    return code.length === CAPTURE_PAIRING_CODE_LENGTH ? code : null;
  });
  const [outcome, setOutcome] = useState<PairingOutcome | null>(null);

  const accessQuery = useQuery(queries.capture.access());
  const previewQuery = useQuery({
    ...queries.capture.pairing(submitted ?? ""),
    enabled: submitted !== null && outcome === null,
    retry: false,
  });
  const preview = previewQuery.data;

  const disabled =
    accessQuery.data !== undefined && (!accessQuery.data.enabled || !accessQuery.data.canCapture);

  const lookUp = (code: string) => {
    if (code === submitted) {
      void previewQuery.refetch();
      return;
    }
    setSubmitted(code);
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Pair a computer"),
        description: t(
          "Approve a computer running Trenova Capture so it can scan and print into Trenova as you",
        ),
      }}
    >
      <div className="mx-auto flex w-full max-w-xl flex-col gap-4">
        {outcome?.decision === "approved" ? (
          <Alert variant="success">
            <CheckCircleIcon />
            <AlertTitle>{t("{0} is paired", outcome.name)}</AlertTitle>
            <AlertDescription>
              {t("Trenova Capture signs in on its own in a few seconds. You can close this tab.")}{" "}
              <Link to="/capture/devices" className="ui-focus-ring text-brand hover:underline">
                {t("My scanners")}
              </Link>
            </AlertDescription>
          </Alert>
        ) : outcome?.decision === "denied" ? (
          <Alert>
            <XCircleIcon />
            <AlertTitle>{t("Pairing refused")}</AlertTitle>
            <AlertDescription>
              {t("That computer was not paired. If it was yours, start again from its tray icon.")}
            </AlertDescription>
          </Alert>
        ) : (
          <>
            {disabled && (
              <Alert variant="warning" size="sm">
                <AlertDescription>
                  {accessQuery.data?.enabled
                    ? t("You do not have permission to scan into Trenova. Ask an administrator.")
                    : t(
                        "Scanning and printing into Trenova is turned off for your organization. Ask an administrator to turn it on.",
                      )}
                </AlertDescription>
              </Alert>
            )}

            <PairingCodeForm
              initialCode={formatPairingCode(linkedCode)}
              disabled={disabled}
              prominent={preview === undefined}
              onLookUp={lookUp}
            />

            {submitted !== null && previewQuery.isLoading && (
              <div className="flex flex-col gap-2" aria-busy="true">
                <Skeleton className="h-4 w-1/3" />
                <Skeleton className="h-24 w-full" />
              </div>
            )}

            {submitted !== null && previewQuery.isError && (
              <LookupFailure
                error={previewQuery.error}
                retrying={previewQuery.isRefetching}
                onRetry={() => void previewQuery.refetch()}
              />
            )}

            {submitted !== null && preview !== undefined && !previewQuery.isError && (
              <PairingReview
                code={submitted}
                preview={preview}
                disabled={disabled}
                onDecided={setOutcome}
              />
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}
