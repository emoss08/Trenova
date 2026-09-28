import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  CAPTURE_PAIRING_CODE_LENGTH,
  formatPairingCode,
  normalizePairingCode,
} from "@/lib/capture";
import { approveCapturePairing, denyCapturePairing } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { CheckCircle2Icon, ShieldAlertIcon, XCircleIcon } from "lucide-react";
import { useId, useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";

type Outcome = { decision: "approved"; name: string } | { decision: "denied" };

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
  const inputId = useId();
  const nameId = useId();

  const [typed, setTyped] = useState(() => formatPairingCode(searchParams.get("code") ?? ""));
  const [submitted, setSubmitted] = useState(() => {
    const code = normalizePairingCode(searchParams.get("code") ?? "");
    return code.length === CAPTURE_PAIRING_CODE_LENGTH ? code : null;
  });
  const [deviceName, setDeviceName] = useState<string | null>(null);
  const [outcome, setOutcome] = useState<Outcome | null>(null);

  const accessQuery = useQuery(queries.capture.access());
  const previewQuery = useQuery({
    ...queries.capture.pairing(submitted ?? ""),
    enabled: submitted !== null && outcome === null,
    retry: false,
  });
  const preview = previewQuery.data;
  const name = deviceName ?? preview?.machineName ?? "";

  const approve = useApiMutation({
    mutationFn: () =>
      approveCapturePairing(submitted ?? "", name.trim() === "" ? null : name.trim()),
    onSuccess: () =>
      setOutcome({ decision: "approved", name: name.trim() || (preview?.machineName ?? "") }),
    resourceName: "Pairing",
  });
  const deny = useApiMutation({
    mutationFn: () => denyCapturePairing(submitted ?? ""),
    onSuccess: () => setOutcome({ decision: "denied" }),
    resourceName: "Pairing",
  });

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const code = normalizePairingCode(typed);
    if (code.length === CAPTURE_PAIRING_CODE_LENGTH) {
      setDeviceName(null);
      setSubmitted(code);
    }
  };

  const disabled =
    accessQuery.data !== undefined && (!accessQuery.data.enabled || !accessQuery.data.canCapture);

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
            <CheckCircle2Icon />
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

            <SectionPanel title={t("Code from Trenova Capture")}>
              <form onSubmit={onSubmit} className="flex items-end gap-2 p-3">
                <div className="flex flex-1 flex-col gap-1">
                  <label htmlFor={inputId} className="text-foreground-subtle text-xs font-medium">
                    {t("The eight letters the computer shows")}
                  </label>
                  <Input
                    id={inputId}
                    value={typed}
                    onChange={(event) => setTyped(formatPairingCode(event.target.value))}
                    placeholder="XXXX-XXXX"
                    autoComplete="off"
                    spellCheck={false}
                    className="font-mono uppercase"
                  />
                </div>
                <Button
                  type="submit"
                  variant="outline"
                  disabled={
                    normalizePairingCode(typed).length !== CAPTURE_PAIRING_CODE_LENGTH || disabled
                  }
                >
                  {t("Look up")}
                </Button>
              </form>
            </SectionPanel>

            {submitted !== null && previewQuery.isLoading && (
              <div className="flex flex-col gap-2" aria-busy="true">
                <Skeleton className="h-4 w-1/3" />
                <Skeleton className="h-24 w-full" />
              </div>
            )}

            {submitted !== null && previewQuery.isError && (
              <Alert variant="destructive" size="sm">
                <AlertDescription>
                  {t(
                    "That code is not valid or has expired. Codes last ten minutes; start again from Trenova Capture's tray icon for a new one.",
                  )}
                </AlertDescription>
              </Alert>
            )}

            {preview !== undefined && (
              <SectionPanel title={t("The computer asking")}>
                <div className="flex flex-col gap-4 p-3">
                  <DescriptionList columns={2}>
                    <DescriptionItem label={t("Computer")}>{preview.machineName}</DescriptionItem>
                    <DescriptionItem label={t("Windows user")}>
                      {preview.windowsUser === "" ? <DescriptionEmpty /> : preview.windowsUser}
                    </DescriptionItem>
                    <DescriptionItem label={t("Trenova Capture")} numeric>
                      {preview.agentVersion}
                    </DescriptionItem>
                    <DescriptionItem label={t("Windows")}>
                      {preview.osVersion === "" ? <DescriptionEmpty /> : preview.osVersion}
                    </DescriptionItem>
                    <DescriptionItem label={t("Asked from")} numeric>
                      {preview.clientIp === "" ? <DescriptionEmpty /> : preview.clientIp}
                    </DescriptionItem>
                    <DescriptionItem label={t("Code expires")} numeric>
                      {formatUnixTime(preview.expiresAt)}
                    </DescriptionItem>
                  </DescriptionList>

                  <Alert variant="warning" size="sm">
                    <ShieldAlertIcon />
                    <AlertDescription>
                      {t(
                        "Approve only a computer you are using now. Once paired it uploads documents as you, with your permissions, until it is revoked.",
                      )}
                    </AlertDescription>
                  </Alert>

                  <div className="flex flex-col gap-1">
                    <label htmlFor={nameId} className="text-foreground-subtle text-xs font-medium">
                      {t("Name it")}
                    </label>
                    <Input
                      id={nameId}
                      value={name}
                      maxLength={100}
                      onChange={(event) => setDeviceName(event.target.value)}
                    />
                  </div>

                  <div className="flex justify-end gap-2">
                    <Button
                      variant="outline"
                      onClick={() => deny.mutate(undefined)}
                      isLoading={deny.isPending}
                      disabled={approve.isPending}
                    >
                      {t("Deny")}
                    </Button>
                    <Button
                      onClick={() => approve.mutate(undefined)}
                      isLoading={approve.isPending}
                      loadingText={t("Pairing")}
                      disabled={deny.isPending || disabled}
                    >
                      {t("Approve")}
                    </Button>
                  </div>
                </div>
              </SectionPanel>
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}
