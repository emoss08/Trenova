import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useForm } from "react-hook-form";
import { supportAccess } from "../../lib/queries/support-access";
import { supportAccessService } from "../../services/support-access";
import {
  elevateRequestSchema,
  type ElevateRequest,
  type SessionView,
} from "../../types/support-access";

const EMPTY: ElevateRequest = { password: "", code: "", reason: "", ticketReference: "" };

/**
 * Raises a read-only support session to read-write for a short window. The staff
 * member signs in again with their password and authenticator code and records why,
 * against a ticket, before anything in the organization can be changed.
 */
export function ElevateDialog({
  open,
  onClose,
  elevationMinutes,
}: {
  open: boolean;
  onClose: () => void;
  elevationMinutes: number;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<ElevateRequest>({
    resolver: zodResolver(elevateRequestSchema),
    defaultValues: EMPTY,
  });

  const elevate = useApiMutation<SessionView, ElevateRequest, unknown, ElevateRequest>({
    mutationFn: (values) => supportAccessService.elevate(values),
    form,
    resourceName: "Support session",
    onSuccess: async () => {
      form.reset(EMPTY);
      await queryClient.invalidateQueries({ queryKey: supportAccess.currentSession().queryKey });
      await queryClient.invalidateQueries();
      onClose();
    },
  });

  const close = () => {
    form.reset(EMPTY);
    onClose();
  };
  const submit = form.handleSubmit((values) => void elevate.mutateAsync(values));

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Elevate to write")}</DialogTitle>
          <DialogDescription>
            {t(
              "Changes you make are recorded in the organization's audit log under your name. Write access ends after {0} minutes.",
              elevationMinutes,
            )}
          </DialogDescription>
        </DialogHeader>
        <Alert variant="warning" size="sm">
          <AlertTriangleIcon />
          <AlertDescription>
            {t("Sign in again with your password and authenticator code to continue.")}
          </AlertDescription>
        </Alert>
        <Form onSubmit={submit}>
          <FormGroup cols={1}>
            <FormControl>
              <SensitiveField
                control={form.control}
                name="password"
                label={t("Password")}
                placeholder={t("Enter your password")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="code"
                label={t("Authentication code")}
                placeholder="123 456"
                autoComplete="one-time-code"
                inputMode="numeric"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="ticketReference"
                label={t("Ticket")}
                placeholder={t("SUP-1234")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={t("What will you change?")}
                placeholder={t("Correct the duplicated invoice the customer reported")}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
        </Form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={close}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            isLoading={elevate.isPending}
            loadingText={t("Verifying...")}
            onClick={submit}
          >
            {t("Elevate to write")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
