import { InputField } from "@/components/fields/input-field";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { CAPTURE_REVOKE_REASON_MAX_BYTES, captureFailureKind } from "@/lib/capture";
import type { CaptureDevice } from "@/lib/graphql/capture";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { PlugOffIcon } from "@trenova/shared/components/icons";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver, type UseFormReturn } from "react-hook-form";
import { revokeDeviceFormSchema, type RevokeDeviceFormValues } from "./capture-forms";

const DEFAULT_VALUES: RevokeDeviceFormValues = { reason: "" };

function isPromiseLike(value: unknown): value is PromiseLike<unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as { then?: unknown }).then === "function"
  );
}

function isFieldValidation(error: unknown): error is GraphQLRequestError | ApiRequestError {
  return (
    (error instanceof GraphQLRequestError || error instanceof ApiRequestError) &&
    error.isValidationError()
  );
}

/**
 * Puts a failed revoke where the person is looking: a rejected reason on the
 * Reason field, anything else above it, so the dialog they are still in says
 * what happened instead of a toast behind it.
 */
function showRevokeFailure(
  t: TranslateFn,
  revokeForm: UseFormReturn<RevokeDeviceFormValues>,
  error: unknown,
) {
  if (isFieldValidation(error)) {
    handleMutationError({ error, form: revokeForm });
    return;
  }

  revokeForm.setError("root", { type: "server", message: revokeFailureMessage(t, error) });
}

function revokeFailureMessage(t: TranslateFn, error: unknown): string {
  switch (captureFailureKind(error)) {
    case "forbidden":
      return t("You do not have permission to revoke this computer.");
    case "not-found":
      return t("This computer is no longer paired. Reload the list to see where it stands.");
    case "invalid":
      // A business rule the server refused on; its message is already in the person's language.
      if (error instanceof GraphQLRequestError || error instanceof ApiRequestError) {
        return error.normalize().message;
      }
      return t("Trenova could not revoke it just now. Try again in a moment.");
    case "unreachable":
      return t("Trenova could not revoke it just now. Try again in a moment.");
  }
}

export function RevokeDeviceDialog({
  device,
  onClose,
  onConfirm,
}: {
  device: CaptureDevice | null;
  onClose: () => void;
  onConfirm: (reason: string) => unknown;
}) {
  const t = useT();
  const schema = useMemo(() => revokeDeviceFormSchema(t), [t]);
  const form = useForm<RevokeDeviceFormValues>({
    resolver: zodResolver(schema) as Resolver<RevokeDeviceFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;
  const deviceId = device?.id;

  useEffect(() => {
    if (deviceId !== undefined) {
      reset(DEFAULT_VALUES);
    }
  }, [deviceId, reset]);

  const submit = handleSubmit(async (values) => {
    const outcome = onConfirm(values.reason);
    if (!isPromiseLike(outcome)) {
      onClose();
      return;
    }
    try {
      await outcome;
      onClose();
    } catch (error) {
      showRevokeFailure(t, form, error);
    }
  });

  return (
    <AlertDialog
      open={device !== null}
      onOpenChange={(open) => {
        if (!open && !isSubmitting) {
          onClose();
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-danger-subtle text-destructive">
            <PlugOffIcon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Revoke {0}?", device?.name ?? "")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "It stops working at once, including a scan it is in the middle of. Pages it already uploaded stay in Intake. To use it again, pair it again.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.stopPropagation();
              void submit(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <InputField<RevokeDeviceFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  aria-label={t("Reason")}
                  description={t("Optional. Kept in the audit trail.")}
                  maxLength={CAPTURE_REVOKE_REASON_MAX_BYTES}
                  autoComplete="off"
                  disabled={isSubmitting}
                />
              </FormControl>
            </FormGroup>
            <AlertDialogFooter>
              <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
                {t("Keep it")}
              </Button>
              <Button
                type="submit"
                variant="destructive"
                isLoading={isSubmitting}
                loadingText={t("Revoking")}
              >
                {t("Revoke")}
              </Button>
            </AlertDialogFooter>
          </Form>
        </FormProvider>
      </AlertDialogContent>
    </AlertDialog>
  );
}
