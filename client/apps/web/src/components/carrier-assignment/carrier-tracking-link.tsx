import { InputField } from "@/components/fields/input-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  issueCarrierTrackingLink,
  revokeCarrierTrackingLink,
  type IssuedCarrierTrackingLink,
} from "@/lib/graphql/check-calls";
import { shipment } from "@/lib/queries/shipment";
import { issueTrackingLinkFormSchema, type IssueTrackingLinkFormValues } from "@/types/check-call";
import { zodResolver } from "@hookform/resolvers/zod";
import { Copy01Icon, Link01Icon, XCloseIcon } from "@trenova/shared/components/icons";
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
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";

export function CarrierTrackingLink({
  moveId,
  carrierName,
}: {
  moveId: string;
  carrierName?: string | null;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [issueOpen, setIssueOpen] = useState(false);
  const linkQuery = shipment.trackingLink(moveId);

  const { data: link, isPending, isError } = useQuery({ ...linkQuery });

  const revokeMutation = useMutation({
    mutationFn: () => revokeCarrierTrackingLink(moveId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: linkQuery.queryKey });
    },
    onError: (error: Error) =>
      toast.error(error.message || t("The tracking link could not be revoked")),
  });

  if (isPending) {
    return <p className="text-2xs text-muted-foreground">{t("Loading tracking link…")}</p>;
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-xs font-medium text-muted-foreground">{t("Tracking link")}</span>
        {isError ? (
          <span className="text-2xs text-danger-foreground">
            {t("The tracking link could not be loaded.")}
          </span>
        ) : link ? (
          <span className="text-2xs text-muted-foreground">
            {[
              link.recipientEmail ? t("Sent to {0}", link.recipientEmail) : t("Shared"),
              t("expires {0}", formatUnixDateTime(link.expiresAt)),
              link.lastUsedAt
                ? t("last used {0}", formatUnixDateTime(link.lastUsedAt))
                : t("not used yet"),
            ].join(" · ")}
          </span>
        ) : (
          <span className="text-2xs text-muted-foreground">{t("None sent")}</span>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-1">
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-6 px-2 text-2xs"
          onClick={() => setIssueOpen(true)}
          title={t("Send the carrier a link to post arrivals, departures and position updates")}
        >
          <Link01Icon className="size-3" aria-hidden />
          {link ? t("Send new link") : t("Send tracking link")}
        </Button>
        {link ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="h-6 px-2 text-2xs text-danger-foreground hover:text-danger-foreground"
            disabled={revokeMutation.isPending}
            onClick={() => revokeMutation.mutate()}
          >
            <XCloseIcon className="size-3" aria-hidden />
            {t("Revoke")}
          </Button>
        ) : null}
      </div>
      <IssueTrackingLinkDialog
        open={issueOpen}
        onOpenChange={setIssueOpen}
        moveId={moveId}
        carrierName={carrierName}
        replacing={!!link}
      />
    </div>
  );
}

function IssueTrackingLinkDialog({
  open,
  onOpenChange,
  moveId,
  carrierName,
  replacing,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moveId: string;
  carrierName?: string | null;
  replacing: boolean;
}) {
  const t = useT();
  const close = () => onOpenChange(false);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && close()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Send tracking link")}</DialogTitle>
          <DialogDescription>
            {replacing
              ? t(
                  "A new link replaces the one already out, which stops working. The carrier posts arrivals, departures and position updates through it until the load is done.",
                )
              : t(
                  "The carrier posts arrivals, departures and position updates through this link until the load is done.",
                )}
          </DialogDescription>
        </DialogHeader>
        {open ? (
          <IssueTrackingLinkBody
            moveId={moveId}
            carrierName={carrierName}
            replacing={replacing}
            onClose={close}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function IssueTrackingLinkBody({
  moveId,
  carrierName,
  replacing,
  onClose,
}: {
  moveId: string;
  carrierName?: string | null;
  replacing: boolean;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [issued, setIssued] = useState<IssuedCarrierTrackingLink | null>(null);

  const form = useForm<IssueTrackingLinkFormValues>({
    resolver: zodResolver(issueTrackingLinkFormSchema),
    defaultValues: { recipientEmail: "" },
  });
  const {
    control,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: IssueTrackingLinkFormValues) =>
      issueCarrierTrackingLink({
        shipmentMoveId: moveId,
        recipientEmail: values.recipientEmail.trim() || undefined,
      }),
    resourceName: "Tracking link",
    form,
    onSuccess: (result) => {
      void queryClient.invalidateQueries({
        queryKey: shipment.trackingLink(moveId).queryKey,
      });
      setIssued(result);
    },
  });

  const onSubmit = useCallback(
    async (values: IssueTrackingLinkFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  if (issued) {
    return <IssuedLink issued={issued} onClose={onClose} />;
  }

  return (
    <FormProvider {...form}>
      <Form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit(onSubmit)(event);
        }}
      >
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="recipientEmail"
              type="email"
              label={t("Email to")}
              placeholder={t("dispatch@carrier.com")}
              description={
                carrierName
                  ? t("Leave empty to copy the link and share it with {0} yourself.", carrierName)
                  : t("Leave empty to copy the link and share it yourself.")
              }
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button type="submit" isLoading={isSubmitting} loadingText={t("Creating...")}>
            {replacing ? t("Send new link") : t("Create link")}
          </Button>
        </DialogFooter>
      </Form>
    </FormProvider>
  );
}

function IssuedLink({
  issued,
  onClose,
}: {
  issued: IssuedCarrierTrackingLink;
  onClose: () => void;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="flex flex-col gap-3">
      {issued.emailed ? (
        <Alert size="sm" variant="success">
          <AlertDescription>
            {t("The link was emailed to {0}.", issued.link.recipientEmail ?? "")}
          </AlertDescription>
        </Alert>
      ) : issued.emailError ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("The link works, but the email could not be sent: {0}", issued.emailError)}
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex items-center gap-2">
        <Input
          readOnly
          value={issued.url}
          aria-label={t("Tracking link")}
          className="font-mono text-xs"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            void copy(issued.url, { withToast: true });
          }}
        >
          <Copy01Icon className="size-3.5" aria-hidden />
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        {t("This link is shown once. Send a new one if it is lost.")}
      </p>
      <DialogFooter>
        <Button type="button" onClick={onClose}>
          {t("Done")}
        </Button>
      </DialogFooter>
    </div>
  );
}
