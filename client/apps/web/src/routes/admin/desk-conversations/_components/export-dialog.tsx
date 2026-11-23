import {
  DESK_TRANSCRIPT_EXPORT_LIST_KEY,
  requestDeskTranscriptExport,
} from "@/lib/graphql/desk-transcripts";
import { deskTranscripts } from "@/lib/queries/desk-transcripts";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useRef, useState } from "react";
import {
  AgentScopePicker,
  OriginPicker,
  PersonPicker,
  TimeframePicker,
} from "./conversation-scope-bar";
import { scopeFilter, type ConversationScope } from "./conversation-scope";
import { exportSummary } from "./export-coverage";

type ExportDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The list's filters, which the dialog opens with. */
  scope: ConversationScope;
  canPickAgent: boolean;
  onExportStarted: () => void;
};

/**
 * Takes a timeframe's conversations out of Trenova as one file. It opens with
 * the list's filters, says how many conversations that is before it starts,
 * and refuses more than one export can hold.
 */
export function ExportDialog({
  open,
  onOpenChange,
  scope: listScope,
  canPickAgent,
  onExportStarted,
}: ExportDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Export Desk conversations")}</DialogTitle>
          <DialogDescription>
            {t(
              "Conversations are chosen by when their last message was sent, in your organization's time zone. The file can be downloaded by you for seven days, and the export is recorded in the audit log.",
            )}
          </DialogDescription>
        </DialogHeader>
        {open ? (
          <ExportForm
            initialScope={listScope}
            canPickAgent={canPickAgent}
            onCancel={() => onOpenChange(false)}
            onStarted={() => {
              onOpenChange(false);
              onExportStarted();
            }}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ExportForm({
  initialScope,
  canPickAgent,
  onCancel,
  onStarted,
}: {
  initialScope: ConversationScope;
  canPickAgent: boolean;
  onCancel: () => void;
  onStarted: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [scope, setScope] = useState<ConversationScope>(initialScope);
  const [pending, setPending] = useState(false);
  const sending = useRef(false);
  const filter = scopeFilter(scope);
  const preview = useQuery({ ...deskTranscripts.exportPreview(filter), retry: false });
  const blocked =
    preview.data === undefined || !preview.data.withinLimit || preview.data.count === 0;

  const start = async () => {
    if (sending.current || blocked) {
      return;
    }
    sending.current = true;
    setPending(true);
    try {
      await requestDeskTranscriptExport({ filter });
      await queryClient.invalidateQueries({ queryKey: [DESK_TRANSCRIPT_EXPORT_LIST_KEY] });
      toast.success(t("Export started"), {
        description: t("You will be told when the file is ready to download."),
      });
      onStarted();
    } catch (error) {
      toast.error(t("The export could not be started"), {
        description: graphQLErrorMessage(error, t("Try again shortly.")),
      });
    } finally {
      sending.current = false;
      setPending(false);
    }
  };

  return (
    <>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1">
          <span className="text-foreground-subtle text-xs font-medium">{t("Timeframe")}</span>
          <TimeframePicker
            className="w-full"
            timeframe={scope.timeframe}
            onChange={(timeframe) => setScope({ ...scope, timeframe })}
          />
        </div>
        <PersonPicker
          personId={scope.personId}
          onChange={(personId) => setScope({ ...scope, personId })}
        />
        {canPickAgent ? (
          <div className="flex flex-col gap-1">
            <span className="text-foreground-subtle text-xs font-medium">{t("Agent")}</span>
            <AgentScopePicker
              agent={scope.agent}
              onChange={(agent) => setScope({ ...scope, agent })}
            />
          </div>
        ) : null}
        <div className="flex flex-col gap-1">
          <span className="text-foreground-subtle text-xs font-medium">{t("Where it began")}</span>
          <OriginPicker
            className="w-full"
            origin={scope.origin}
            onChange={(origin) => setScope({ ...scope, origin })}
          />
        </div>
        {preview.isError ? (
          <Alert variant="destructive" size="sm">
            <AlertDescription>
              {graphQLErrorMessage(preview.error, t("The conversations could not be counted."))}
            </AlertDescription>
          </Alert>
        ) : (
          <Alert
            variant={preview.data && !preview.data.withinLimit ? "warning" : "info"}
            size="sm"
            aria-live="polite"
          >
            <AlertDescription>{exportSummary(t, preview.data, scope)}</AlertDescription>
          </Alert>
        )}
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button type="button" disabled={blocked} isLoading={pending} onClick={() => void start()}>
          {t("Export")}
        </Button>
      </DialogFooter>
    </>
  );
}
