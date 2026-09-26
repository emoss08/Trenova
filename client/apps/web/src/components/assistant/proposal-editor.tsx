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
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import type { PreviewScope } from "@/lib/graphql/agent-preview";
import { modificationsKey } from "@/lib/queries/agent-preview";
import type { ProposalField } from "@/types/assistant";
import { CircleAlertIcon, LockIcon } from "lucide-react";
import { humanizeKey } from "./readable-values";
import { useId, useMemo, useState } from "react";
import {
  changedValues,
  draftFromArguments,
  parseParamPath,
  readNested,
  validateDraft,
  writeNested,
  type ProposalDraft,
} from "./proposal-edits";
import {
  canApprove,
  gateDigest,
  isPreviewRefusal,
  type ApprovalGate,
} from "./proposal-preview/preview-gate";
import { PreviewLoadState, ProposalPreview } from "./proposal-preview/proposal-preview";
import { useApprovalGate, useProposalPreview } from "./proposal-preview/use-proposal-preview";
import { previewOutcomes } from "./record-subset";
import { RecordSubsetField } from "./record-subset-field";

/** How long typing settles before the draft is previewed. */
export const PREVIEW_DEBOUNCE_MS = 400;

/**
 * The value the editor opens on: a parameter path (status, shipment.bol)
 * and the words it is known by. A path into an object parameter gets an
 * input of its own above the object, so a person changes the one value a
 * refusal named without editing JSON.
 */
export type EditorFocus = { param: string; label: string };

export type ProposalEditorRequest = {
  /** The sentence the card shows, so the person knows which change they are editing. */
  summary: string;
  fields: readonly ProposalField[];
  arguments: Record<string, unknown> | null | undefined;
  /** The value to open on, when a reason named one. */
  focus?: EditorFocus;
  /** When set, a reason is asked for beside the values and passed along. */
  withReason?: { label: string; required: boolean };
  /**
   * Where the draft's preview is read. The editor previews each valid draft
   * and approves against the preview it shows, sending its digest.
   */
  preview?: { scope: PreviewScope; proposalId: string };
  onConfirm: (
    modifications: Record<string, unknown>,
    reason: string,
    previewDigest: string | undefined,
  ) => Promise<void> | void;
};

/**
 * Approve with changes.
 *
 * The agent proposed values; the person wanted most of them. Rejecting and
 * asking again is the long way round, so the card opens the values as a form
 * built from the tool's own schema, and approval carries what was changed.
 * Only real changes are sent: the server records them on the decision, runs
 * the tool with them, and tells the model what actually ran.
 *
 * Beside the values sits what they would do: once typing settles, the draft
 * is previewed by the tool's own code, and approval waits for the preview of
 * exactly the values in the form. The record the change is about stays as
 * proposed — a change may alter what is done to it, never which record it is.
 */
export function ProposalEditor({
  request,
  onClose,
}: {
  request: ProposalEditorRequest | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={request !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="lg">
        {request && <EditorForm request={request} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}

const LOADING: ApprovalGate = { state: "loading" };
const NO_OUTCOMES: ReadonlyMap<string, string> = new Map();

// Mounted only while open, so each request starts from the proposed values
// without an effect having to reset the draft.
function EditorForm({ request, onClose }: { request: ProposalEditorRequest; onClose: () => void }) {
  const t = useT();
  const idPrefix = useId();
  const [draft, setDraft] = useState<ProposalDraft>(() =>
    draftFromArguments(request.fields, request.arguments),
  );
  const [reason, setReason] = useState("");
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [submitted, setSubmitted] = useState(false);
  const [isPending, setIsPending] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  const errors = useMemo(() => validateDraft(request.fields, draft), [request.fields, draft]);
  const changes = useMemo(
    () => changedValues(request.fields, request.arguments, draft),
    [request.fields, request.arguments, draft],
  );
  const changeCount = Object.keys(changes).length;
  const valid = Object.keys(errors).length === 0;

  // A valid draft is previewed once typing settles; an invalid one is not
  // sent at all. The preview of an unchanged draft is the proposal as
  // proposed, which the card beside the editor has usually read already.
  const draftModifications = valid ? changes : null;
  const settledModifications = useDebounce(draftModifications, PREVIEW_DEBOUNCE_MS);
  const target = request.preview;
  const previewQuery = useProposalPreview({
    scope: target?.scope ?? "mine",
    id: target?.proposalId ?? "",
    modifications: settledModifications,
    enabled: target !== undefined && settledModifications !== null,
    keepPrevious: true,
  });
  const approval = useApprovalGate(previewQuery);

  // A record-subset field lists every record proposed, each with what the
  // preview says happens to it. The preview as proposed names the records a
  // person unticks, so its word on them stays on their rows once they are
  // out of the draft; the draft's own preview has the last word on the rest.
  const hasSubset = request.fields.some((field) => field.kind === "RecordSubset");
  const proposedPreview = useProposalPreview({
    scope: target?.scope ?? "mine",
    id: target?.proposalId ?? "",
    enabled: target !== undefined && hasSubset,
  });
  const outcomes = useMemo(
    () => (hasSubset ? previewOutcomes([proposedPreview.data, previewQuery.data], t) : NO_OUTCOMES),
    [hasSubset, previewQuery.data, proposedPreview.data, t],
  );

  // What is on screen is the preview of the values in the form only once the
  // debounce has caught up and the read for them has landed.
  const previewCurrent =
    draftModifications !== null &&
    settledModifications !== null &&
    modificationsKey(settledModifications) === modificationsKey(draftModifications) &&
    !previewQuery.isPlaceholderData;
  const refused = previewCurrent && isPreviewRefusal(previewQuery.error);
  const gate = previewCurrent ? approval.gate : LOADING;

  const reasonMissing = request.withReason?.required === true && reason.trim() === "";
  const previewAllows = target === undefined || (canApprove(gate) && !refused);
  const canConfirm = !isPending && valid && changeCount > 0 && !reasonMissing && previewAllows;

  const set = (name: string, value: string) => {
    setDraft((current) => ({ ...current, [name]: value }));
    setTouched((current) => (current[name] ? current : { ...current, [name]: true }));
  };

  // A refusal names one value; the editor opens on it. A path into an object
  // parameter is edited in an input of its own, and the object follows.
  const focus = useMemo(() => {
    const parsed = request.focus ? parseParamPath(request.focus.param) : null;
    if (parsed === null) {
      return null;
    }
    const field = request.fields.find((entry) => entry.name === parsed.field);
    if (field === undefined || field.readOnly === true) {
      return null;
    }

    return { field, steps: parsed.steps, label: request.focus?.label ?? "" };
  }, [request.fields, request.focus]);
  const nested =
    focus !== null && focus.steps.length > 0 && focus.field.kind === "JSON"
      ? readNested(draft, focus.field.name, focus.steps)
      : undefined;
  const setNested = (value: string) => {
    if (focus === null) {
      return;
    }
    setDraft((current) => writeNested(current, focus.field.name, focus.steps, value));
    setTouched((current) =>
      current[focus.field.name] ? current : { ...current, [focus.field.name]: true },
    );
  };

  const confirm = async () => {
    setSubmitted(true);
    if (!canConfirm) {
      return;
    }
    approval.acknowledge();
    setIsPending(true);
    setFailure(null);
    try {
      await request.onConfirm(changes, reason.trim(), target ? gateDigest(gate) : undefined);
      onClose();
    } catch (error) {
      // What these values would do moved while the person read it: the
      // server recorded nothing, the preview is read again, and the form
      // stays open on the new one.
      if (!(target && approval.handleDecisionError(error))) {
        setFailure(graphQLErrorMessage(error, t("The change could not be saved.")));
      }
    } finally {
      setIsPending(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("Approve with changes")}</DialogTitle>
        <DialogDescription>{request.summary}</DialogDescription>
      </DialogHeader>

      <div className="flex max-h-[65vh] flex-col gap-4 overflow-y-auto py-1">
        <div className="flex flex-col gap-3">
          {request.fields.map((field) => {
            const id = `${idPrefix}-${field.name}`;
            const subset = field.kind === "RecordSubset";
            const error =
              !subset && (touched[field.name] || submitted) ? errors[field.name] : undefined;
            const readOnly = field.readOnly === true;

            const focused = focus?.field.name === field.name;
            const nestedHere = focused && nested !== undefined;

            return (
              <div key={field.name} className="flex flex-col gap-1.5">
                {nestedHere && (
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor={`${id}-focus`}>
                      {focus.label !== "" ? focus.label : humanizeKey(String(focus.steps.at(-1)))}
                    </Label>
                    <Input
                      id={`${id}-focus`}
                      value={nested}
                      autoFocus
                      onChange={(event) => setNested(event.target.value)}
                    />
                    <p className="text-muted-foreground text-xs">
                      {t(
                        "Part of {0}; the rest stays as proposed unless you change it below.",
                        field.label,
                      )}
                    </p>
                  </div>
                )}
                <Label id={`${id}-label`} htmlFor={subset ? undefined : id}>
                  {field.label}
                  {field.required || subset ? "" : ` (${t("optional")})`}
                </Label>
                {subset ? (
                  <RecordSubsetField
                    labelId={`${id}-label`}
                    field={field}
                    proposed={request.arguments?.[field.name]}
                    value={draft[field.name] ?? ""}
                    outcomes={outcomes}
                    readOnly={readOnly}
                    onChange={(value) => set(field.name, value)}
                  />
                ) : (
                  <FieldControl
                    id={id}
                    field={field}
                    value={draft[field.name] ?? ""}
                    invalid={error !== undefined}
                    readOnly={readOnly}
                    autoFocus={focused && !nestedHere && focus.steps.length === 0}
                    onChange={(value) => set(field.name, value)}
                  />
                )}
                {readOnly ? (
                  <p className="text-foreground-subtle flex items-center gap-1 text-xs">
                    <LockIcon aria-hidden className="size-3 shrink-0" />
                    {t("Stays as proposed: a change can't point this at a different record.")}
                  </p>
                ) : (
                  field.description !== "" && (
                    <p className="text-muted-foreground text-xs">{field.description}</p>
                  )
                )}
                {error && <p className="text-danger text-xs">{error}</p>}
              </div>
            );
          })}

          {request.withReason && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`${idPrefix}-reason`}>
                {request.withReason.label}
                {request.withReason.required ? "" : ` (${t("optional")})`}
              </Label>
              <Textarea
                id={`${idPrefix}-reason`}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                minRows={2}
                maxLength={500}
                placeholder={t("Why the values changed, for the audit trail.")}
              />
            </div>
          )}
        </div>

        {target && (
          <section
            className="border-border-subtle flex flex-col gap-2 border-t pt-3"
            aria-live="polite"
          >
            <h3 className="text-foreground-subtle text-xs font-medium">
              {changeCount > 0 ? t("What your values would do") : t("What it would do as proposed")}
            </h3>
            {!valid ? (
              <p className="text-foreground-muted text-xs">
                {t("Fix the values above to see what they would do.")}
              </p>
            ) : refused ? (
              <Alert size="sm" variant="destructive">
                <CircleAlertIcon />
                <AlertDescription>
                  {t(
                    "These values would not go through: {0}",
                    graphQLErrorMessage(previewQuery.error, t("The values are not valid.")),
                  )}
                </AlertDescription>
              </Alert>
            ) : (
              <div
                className={cn("min-w-0", !previewCurrent && "opacity-60")}
                aria-busy={!previewCurrent}
              >
                <PreviewLoadState query={previewQuery} changed={approval.changed}>
                  {(preview) => <ProposalPreview preview={preview} density="full" />}
                </PreviewLoadState>
              </div>
            )}
          </section>
        )}
      </div>

      {failure && <p className="text-danger text-xs">{failure}</p>}

      <DialogFooter className="items-center">
        <span className="text-muted-foreground mr-auto text-xs">
          {changeCount === 0
            ? t("Nothing changed yet")
            : changeCount === 1
              ? t("1 value changed")
              : t("{0} values changed", changeCount)}
        </span>
        <Button type="button" variant="outline" onClick={onClose} disabled={isPending}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          onClick={() => void confirm()}
          disabled={!canConfirm}
          isLoading={isPending}
        >
          {t("Approve with changes")}
        </Button>
      </DialogFooter>
    </>
  );
}

function FieldControl({
  id,
  field,
  value,
  invalid,
  readOnly,
  autoFocus = false,
  onChange,
}: {
  id: string;
  field: ProposalField;
  value: string;
  invalid: boolean;
  readOnly: boolean;
  /** The control the editor was opened on takes focus. */
  autoFocus?: boolean;
  onChange: (value: string) => void;
}) {
  const t = useT();

  switch (field.kind) {
    case "Multiline":
    case "JSON":
      return (
        <Textarea
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          minRows={field.kind === "JSON" ? 4 : 3}
          maxLength={field.maxLength ?? undefined}
          isInvalid={invalid}
          readOnly={readOnly}
          autoFocus={autoFocus}
          className={cn(field.kind === "JSON" && "font-mono text-xs")}
        />
      );
    case "Boolean":
      return (
        <div className="flex items-center gap-2">
          <Switch
            id={id}
            checked={value === "true"}
            disabled={readOnly}
            onCheckedChange={(checked) => onChange(checked ? "true" : "false")}
          />
          <span className="text-muted-foreground text-xs">
            {value === "true" ? t("Yes") : t("No")}
          </span>
        </div>
      );
    case "Choice":
      return (
        <Select
          value={value === "" ? null : value}
          disabled={readOnly}
          onValueChange={(next) => onChange(next ?? "")}
        >
          <SelectTrigger id={id} aria-invalid={invalid}>
            <SelectValue placeholder={t("Choose…")} />
          </SelectTrigger>
          <SelectContent>
            {field.options.map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    case "Integer":
    case "Number":
      return (
        <Input
          id={id}
          type="number"
          inputMode={field.kind === "Integer" ? "numeric" : "decimal"}
          step={field.kind === "Integer" ? 1 : "any"}
          min={field.minimum ?? undefined}
          max={field.maximum ?? undefined}
          value={value}
          readOnly={readOnly}
          autoFocus={autoFocus}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={invalid}
        />
      );
    case "List":
      return (
        <Input
          id={id}
          value={value}
          readOnly={readOnly}
          onChange={(event) => onChange(event.target.value)}
          placeholder={
            field.options.length > 0 ? field.options.join(", ") : t("Comma-separated values")
          }
          autoFocus={autoFocus}
          aria-invalid={invalid}
        />
      );
    default:
      return (
        <Input
          id={id}
          value={value}
          readOnly={readOnly}
          autoFocus={autoFocus}
          onChange={(event) => onChange(event.target.value)}
          maxLength={field.maxLength ?? undefined}
          aria-invalid={invalid}
        />
      );
  }
}
