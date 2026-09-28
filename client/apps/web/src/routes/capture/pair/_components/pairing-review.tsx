import {
  approvePairingFormSchema,
  type ApprovePairingFormValues,
} from "@/components/capture/capture-forms";
import { InputField } from "@/components/fields/input-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { CAPTURE_DEVICE_NAME_MAX_BYTES } from "@/lib/capture";
import {
  approveCapturePairing,
  denyCapturePairing,
  type CapturePairingPreview,
} from "@/lib/graphql/capture";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { ShieldAlertIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";

export type PairingOutcome = { decision: "approved"; name: string } | { decision: "denied" };

function orEmpty(value: string) {
  return value === "" ? <DescriptionEmpty /> : value;
}

/**
 * The computer asking to pair, and the decision on it. The person sees which
 * machine it is before approving, so a code read off somebody else's screen
 * is recognisably not theirs.
 */
export function PairingReview({
  code,
  preview,
  disabled,
  onDecided,
}: {
  code: string;
  preview: CapturePairingPreview;
  disabled: boolean;
  onDecided: (outcome: PairingOutcome) => void;
}) {
  const t = useT();
  const schema = useMemo(() => approvePairingFormSchema(t), [t]);
  const form = useForm<ApprovePairingFormValues>({
    resolver: zodResolver(schema) as Resolver<ApprovePairingFormValues>,
    defaultValues: { deviceName: preview.machineName },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    reset({ deviceName: preview.machineName });
  }, [preview.machineName, preview.userCode, reset]);

  const approve = useApiMutation<
    boolean,
    ApprovePairingFormValues,
    unknown,
    ApprovePairingFormValues
  >({
    form,
    resourceName: "Pairing",
    mutationFn: async ({ deviceName }) => {
      await approveCapturePairing(code, deviceName === "" ? null : deviceName);
      return true;
    },
    onSuccess: (_, { deviceName }) =>
      onDecided({
        decision: "approved",
        name: deviceName === "" ? preview.machineName : deviceName,
      }),
  });
  const deny = useApiMutation({
    resourceName: "Pairing",
    mutationFn: () => denyCapturePairing(code),
    onSuccess: () => onDecided({ decision: "denied" }),
  });
  const busy = approve.isPending || deny.isPending;

  return (
    <SectionPanel title={t("The computer asking")}>
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => approve.mutateAsync(values).catch(() => undefined))(
              event,
            );
          }}
        >
          <div className="flex flex-col gap-4 p-3">
            <DescriptionList columns={2}>
              <DescriptionItem label={t("Computer")}>{preview.machineName}</DescriptionItem>
              <DescriptionItem label={t("Windows user")}>
                {orEmpty(preview.windowsUser)}
              </DescriptionItem>
              <DescriptionItem label={t("Trenova Capture")} numeric>
                {preview.agentVersion}
              </DescriptionItem>
              <DescriptionItem label={t("Windows")}>{orEmpty(preview.osVersion)}</DescriptionItem>
              <DescriptionItem label={t("Asked from")} numeric>
                {orEmpty(preview.clientIp)}
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

            <FormGroup cols={1}>
              <FormControl>
                <InputField<ApprovePairingFormValues>
                  control={control}
                  name="deviceName"
                  label={t("Name it")}
                  aria-label={t("Name it")}
                  description={t(
                    "How it appears in My scanners. Leave it blank to use the computer's own name.",
                  )}
                  maxLength={CAPTURE_DEVICE_NAME_MAX_BYTES}
                  autoComplete="off"
                  disabled={busy}
                />
              </FormControl>
            </FormGroup>
          </div>
          <div className="border-border flex justify-end gap-2 border-t px-3 py-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => deny.mutate(undefined)}
              isLoading={deny.isPending}
              loadingText={t("Denying")}
              disabled={approve.isPending}
            >
              {t("Deny")}
            </Button>
            <Button
              type="submit"
              isLoading={approve.isPending}
              loadingText={t("Pairing")}
              disabled={deny.isPending || disabled}
            >
              {t("Approve")}
            </Button>
          </div>
        </Form>
      </FormProvider>
    </SectionPanel>
  );
}
