import { documentContentUrl } from "@/services/document";
import { apiService } from "@/services/api";
import { RateConfirmationStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
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
  type RateConfirmation,
} from "@trenova/shared/types/rate-confirmation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FileCheck2Icon, FileTextIcon, MailIcon, RefreshCcwIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function rateConfirmationQueryKey(moveId: string) {
  return ["move-rate-confirmations", moveId] as const;
}

export function RateConfirmationActions({ moveId }: { moveId: string }) {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [voidOpen, setVoidOpen] = useState(false);

  const { data: rateConfirmations, isLoading } = useQuery({
    queryKey: rateConfirmationQueryKey(moveId),
    queryFn: () => apiService.rateConfirmationService.listByMove(moveId),
  });

  const latest = latestActiveRateConfirmation(rateConfirmations);

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: rateConfirmationQueryKey(moveId) });
  };

  const generateMutation = useMutation({
    mutationFn: () => apiService.rateConfirmationService.generate(moveId),
    onSuccess: (rateCon) => {
      toast.success(
        rateCon.revision > 1
          ? `Rate confirmation regenerated — revision ${rateCon.revision} filed, prior revision voided`
          : "Rate confirmation generated and filed",
      );
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to generate rate confirmation"),
  });

  const sendMutation = useMutation({
    mutationFn: (rateConId: string) => apiService.rateConfirmationService.send(rateConId),
    onSuccess: (rateCon) => {
      toast.success(
        rateCon.sentToEmails
          ? `Rate confirmation sent to ${rateCon.sentToEmails}`
          : "Rate confirmation sent",
      );
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to send rate confirmation"),
  });

  if (isLoading) {
    return <p className="text-2xs text-muted-foreground">Loading rate confirmations…</p>;
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-2xs font-medium text-muted-foreground uppercase">Rate Con</span>
        {latest ? (
          <>
            <RateConfirmationStatusBadge status={latest.status} />
            <span className="text-2xs text-muted-foreground tabular-nums">
              rev {latest.revision}
            </span>
            {latest.status === "Confirmed" && latest.confirmedByName && (
              <span className="text-2xs text-muted-foreground">by {latest.confirmedByName}</span>
            )}
            {latest.status === "Voided" && latest.voidReason && (
              <span className="text-2xs text-muted-foreground">({latest.voidReason})</span>
            )}
          </>
        ) : (
          <span className="text-2xs text-muted-foreground">None generated</span>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-1">
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[10px]"
          disabled={generateMutation.isPending}
          onClick={() => generateMutation.mutate()}
          title={
            latest && latest.status !== "Voided"
              ? "Regenerate — files a new revision and voids the prior one"
              : "Generate the rate confirmation PDF from the current buy rate"
          }
        >
          <RefreshCcwIcon className="size-3" aria-hidden />
          {latest && latest.status !== "Voided" ? "Regenerate" : "Generate"}
        </Button>
        {latest && (latest.status === "Generated" || latest.status === "Sent") && (
          <>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="h-6 px-2 text-[10px]"
              disabled={sendMutation.isPending}
              onClick={() => sendMutation.mutate(latest.id)}
              title="Email the rate confirmation to the carrier's rate confirmation contacts"
            >
              <MailIcon className="size-3" aria-hidden />
              {latest.status === "Sent" ? "Resend" : "Send"}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="h-6 px-2 text-[10px]"
              onClick={() => setConfirmOpen(true)}
              title="Record that the carrier confirmed this rate"
            >
              <FileCheck2Icon className="size-3" aria-hidden />
              Mark Confirmed
            </Button>
          </>
        )}
        {latest && latest.status !== "Voided" && (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="h-6 px-2 text-[10px] text-red-600 hover:text-red-700 dark:text-red-400"
            onClick={() => setVoidOpen(true)}
          >
            <XIcon className="size-3" aria-hidden />
            Void
          </Button>
        )}
        {latest?.documentId && (
          <a
            href={documentContentUrl(latest.documentId, "view")}
            target="_blank"
            rel="noreferrer"
            className="inline-flex h-6 items-center gap-1 rounded-md border px-2 text-[10px] font-medium hover:bg-muted"
            title="Open the filed rate confirmation document"
          >
            <FileTextIcon className="size-3" aria-hidden />
            View PDF
          </a>
        )}
      </div>
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
    </div>
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
  const [confirmedByName, setConfirmedByName] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      apiService.rateConfirmationService.confirm(rateConfirmation.id, confirmedByName.trim()),
    onSuccess: () => {
      toast.success("Rate confirmation marked confirmed");
      setConfirmedByName("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to confirm"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Mark rate confirmation confirmed</DialogTitle>
          <DialogDescription>
            Records who at the carrier confirmed revision {rateConfirmation.revision} — from a
            signed copy, email reply, or phone confirmation.
          </DialogDescription>
        </DialogHeader>
        <Input
          value={confirmedByName}
          onChange={(event) => setConfirmedByName(event.target.value)}
          placeholder="Confirmed by (name at the carrier)"
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!confirmedByName.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Mark Confirmed
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
  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () => apiService.rateConfirmationService.void(rateConfirmation.id, reason.trim()),
    onSuccess: () => {
      toast.success("Rate confirmation voided");
      setReason("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to void"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Void rate confirmation</DialogTitle>
          <DialogDescription>
            Voids revision {rateConfirmation.revision}. Generate again to file a fresh revision with
            the current buy rate.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="Reason (required)"
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={!reason.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Void
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
