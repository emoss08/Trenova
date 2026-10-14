import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { bounceTenderPayloadSchema, type BounceTenderPayload } from "@trenova/shared/types/tender";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

/**
 * Records that the carrier holding an accepted tender walked it back. The
 * server cancels the carrier assignment, voids its rate confirmation and sends
 * the tender on to the carriers the acceptance cut short, so the dialog says
 * all of that before the dispatcher confirms.
 */
export function TenderBounceDialog({
  carrierName,
  isSubmitting,
  onOpenChange,
  onConfirm,
}: {
  carrierName: string | null;
  isSubmitting: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (payload: BounceTenderPayload) => void;
}) {
  const t = useT();

  const form = useForm({
    resolver: zodResolver(bounceTenderPayloadSchema),
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit } = form;

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onOpenChange(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Carrier backed out")}</DialogTitle>
          <DialogDescription>
            {t(
              "{0} accepted this tender and has now backed out. Their carrier assignment and rate confirmation are canceled, and the tender goes on to the next carrier in line.",
              carrierName ?? t("The carrier"),
            )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => onConfirm(values))(event);
          }}
        >
          <div className="pb-4">
            <TextareaField
              control={control}
              name="reason"
              label={t("Reason")}
              rules={{ required: true }}
              placeholder={t("e.g., Truck broke down before pickup")}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Keep the carrier")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={isSubmitting}
              loadingText={t("Recording...")}
            >
              {t("Record backout")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
