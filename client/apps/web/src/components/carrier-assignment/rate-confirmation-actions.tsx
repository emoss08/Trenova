import { useT } from "@trenova/shared/i18n/use-t";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { documentContentUrl } from "@/services/document";
import { apiService } from "@/services/api";
import { RateConfirmationStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  latestActiveRateConfirmation,
  RATE_CONFIRMATION_VIA_LABEL,
  type RateConfirmation,
} from "@trenova/shared/types/rate-confirmation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  File06Icon,
  FileCheck03Icon,
  Mail01Icon,
  RefreshCcw02Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useState, type ReactNode } from "react";
import {
  CarrierDocumentRow,
  CarrierDocumentRowSkeleton,
  SublineParts,
} from "./carrier-document-row";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";

export function rateConfirmationQueryKey(moveId: string) {
  return ["move-rate-confirmations", moveId] as const;
}

export function RateConfirmationActions({
  moveId,
  carrierAssignmentId,
}: {
  moveId: string;
  carrierAssignmentId: string;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [voidOpen, setVoidOpen] = useState(false);

  const { data: rateConfirmations, isLoading } = useQuery({
    queryKey: rateConfirmationQueryKey(moveId),
    queryFn: () => apiService.rateConfirmationService.listByMove(moveId),
  });

  // A move can accumulate rate confirmations from replaced assignments; only the
  // active assignment's paper is actionable here.
  const scopedRateConfirmations = rateConfirmations?.filter(
    (rateCon) => rateCon.carrierAssignmentId === carrierAssignmentId,
  );
  const latest = latestActiveRateConfirmation(scopedRateConfirmations);

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: rateConfirmationQueryKey(moveId) });
  };

  const generateMutation = useMutation({
    mutationFn: () => apiService.rateConfirmationService.generate(moveId),
    onSuccess: () => {
      invalidate();
    },
    onError: (error: Error) =>
      toast.error(error.message || t("Failed to generate rate confirmation")),
  });

  const sendMutation = useMutation({
    mutationFn: (rateConId: string) => apiService.rateConfirmationService.send(rateConId),
    onSuccess: (rateCon) => {
      toast.success(
        rateCon.sentToEmails
          ? t("Rate confirmation sent to {0}", rateCon.sentToEmails)
          : t("Rate confirmation sent"),
      );
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message || t("Failed to send rate confirmation")),
  });

  // One request at a time: a second send or generate while the first is out is ignored.
  const generate = useSingleFlight(() => generateMutation.mutateAsync());
  const send = useSingleFlight((rateConId: string) => sendMutation.mutateAsync(rateConId));

  if (isLoading) {
    return <CarrierDocumentRowSkeleton label={t("Loading rate confirmations…")} />;
  }

  const active = latest && latest.status !== "Voided" ? latest : null;
  const busy = generate.pending || send.pending;

  // The one step that moves the paper forward; everything else, sending included, waits in the
  // menu.
  const nextStep = !active ? (
    <Button
      type="button"
      variant="outline"
      size="xxs"
      isLoading={generate.pending}
      disabled={busy}
      onClick={() => void generate.run()}
      title={t("Generate the rate confirmation PDF from the current buy rate")}
    >
      {t("Generate")}
    </Button>
  ) : active.status === "Sent" ? (
    <Button
      type="button"
      variant="outline"
      size="xxs"
      onClick={() => setConfirmOpen(true)}
      title={t("Record that the carrier confirmed this rate")}
    >
      {t("Mark confirmed")}
    </Button>
  ) : null;

  const subline: { key: string; node: ReactNode }[] = [];
  if (!latest) {
    subline.push({ key: "none", node: t("Not generated yet") });
  } else {
    subline.push({
      key: "revision",
      node: <span className="tabular-nums">{t("Revision {0}", latest.revision)}</span>,
    });
    if (latest.generatedVia === "TenderAcceptance") {
      subline.push({
        key: "auto",
        node: (
          <span title={t("Generated automatically when the carrier accepted the tender")}>
            {t("Auto-issued from tender")}
          </span>
        ),
      });
    }
    if (latest.status === "Confirmed" && latest.confirmedVia) {
      subline.push({
        key: "via",
        node: (
          <span title={t("How the carrier's confirmation was captured")}>
            {RATE_CONFIRMATION_VIA_LABEL[latest.confirmedVia]}
          </span>
        ),
      });
    }
    if (latest.status === "Confirmed" && latest.confirmedByName) {
      subline.push({
        key: "by",
        node: (
          <span className="truncate">
            {latest.confirmedByTitle
              ? t("by {0}, {1}", latest.confirmedByName, latest.confirmedByTitle)
              : t("by {0}", latest.confirmedByName)}
          </span>
        ),
      });
    }
    if (latest.status === "Voided" && latest.voidReason) {
      subline.push({
        key: "void",
        node: <span className="truncate">{t("Voided: {0}", latest.voidReason)}</span>,
      });
    }
  }

  const documentId = latest?.documentId;
  const menu = [
    documentId ? (
      <DropdownMenuItem
        key="view"
        title={t("View PDF")}
        description={t("Open the filed rate confirmation in a new tab")}
        startContent={<File06Icon className="size-3.5" />}
        onClick={() =>
          window.open(documentContentUrl(documentId, "view"), "_blank", "noopener,noreferrer")
        }
      />
    ) : null,
    active && (active.status === "Generated" || active.status === "Sent") ? (
      <DropdownMenuItem
        key="send"
        title={active.status === "Sent" ? t("Resend") : t("Send")}
        description={
          active.status === "Sent"
            ? t("Email it again to the carrier's rate confirmation contacts")
            : t("Email it to the carrier's rate confirmation contacts")
        }
        startContent={<Mail01Icon className="size-3.5" />}
        disabled={busy}
        onClick={() => void send.run(active.id)}
      />
    ) : null,
    active?.status === "Generated" ? (
      <DropdownMenuItem
        key="confirm"
        title={t("Mark confirmed")}
        description={t("Record who at the carrier confirmed this rate")}
        startContent={<FileCheck03Icon className="size-3.5" />}
        onClick={() => setConfirmOpen(true)}
      />
    ) : null,
    active ? (
      <DropdownMenuItem
        key="regenerate"
        title={t("Regenerate")}
        description={t("Files a new revision and voids this one")}
        startContent={<RefreshCcw02Icon className="size-3.5" />}
        disabled={busy}
        onClick={() => void generate.run()}
      />
    ) : null,
    active ? <DropdownMenuSeparator key="separator" /> : null,
    active ? (
      <DropdownMenuItem
        key="void"
        title={t("Void")}
        description={t("Cancel this revision; generate again for a fresh one")}
        color="danger"
        startContent={<XCloseIcon className="size-3.5" />}
        onClick={() => setVoidOpen(true)}
      />
    ) : null,
  ].filter(Boolean);

  return (
    <>
      <CarrierDocumentRow
        icon={File06Icon}
        title={t("Rate confirmation")}
        subline={<SublineParts parts={subline} />}
        status={latest ? <RateConfirmationStatusBadge status={latest.status} /> : null}
        action={nextStep}
        menuLabel={t("Rate confirmation actions")}
        busy={busy}
        menu={menu}
      />
      {latest && (
        <>
          <MarkConfirmedDialog
            open={confirmOpen}
            rateConfirmation={latest}
            onOpenChange={setConfirmOpen}
            onChanged={invalidate}
          />
          <VoidRateConfirmationDialog
            open={voidOpen}
            rateConfirmation={latest}
            onOpenChange={setVoidOpen}
            onChanged={invalidate}
          />
        </>
      )}
    </>
  );
}

function MarkConfirmedDialog({
  open,
  rateConfirmation,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  rateConfirmation: RateConfirmation;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const t = useT();

  const [confirmedByName, setConfirmedByName] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      apiService.rateConfirmationService.confirm(rateConfirmation.id, confirmedByName.trim()),
    onSuccess: () => {
      setConfirmedByName("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || translate("Failed to confirm")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Mark rate confirmation confirmed")}</DialogTitle>
          <DialogDescription>
            {t(
              "Records who at the carrier confirmed revision {0} — from a signed copy, email reply, or phone confirmation.",
              rateConfirmation.revision,
            )}
          </DialogDescription>
        </DialogHeader>
        <Input
          value={confirmedByName}
          onChange={(event) => setConfirmedByName(event.target.value)}
          placeholder={t("Confirmed by (name at the carrier)")}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            disabled={!confirmedByName.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t("Mark confirmed")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function VoidRateConfirmationDialog({
  open,
  rateConfirmation,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  rateConfirmation: RateConfirmation;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const t = useT();

  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () => apiService.rateConfirmationService.void(rateConfirmation.id, reason.trim()),
    onSuccess: () => {
      setReason("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || translate("Failed to void")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Void rate confirmation")}</DialogTitle>
          <DialogDescription>
            {t(
              "Voids revision {0}. Generate again to file a fresh revision with the current buy rate.",
              rateConfirmation.revision,
            )}
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder={t("Reason (required)")}
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            variant="destructive"
            disabled={!reason.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t("Void")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
