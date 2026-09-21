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
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ProposalField } from "@/types/assistant";
import { useId, useMemo, useState } from "react";
import {
  changedValues,
  draftFromArguments,
  validateDraft,
  type ProposalDraft,
} from "./proposal-edits";

export type ProposalEditorRequest = {
  /** The sentence the card shows, so the person knows which change they are editing. */
  summary: string;
  fields: readonly ProposalField[];
  arguments: Record<string, unknown> | null | undefined;
  /** When set, a reason is asked for beside the values and passed along. */
  withReason?: { label: string; required: boolean };
  onConfirm: (modifications: Record<string, unknown>, reason: string) => Promise<void> | void;
};

/**
 * Approve with changes.
 *
 * The agent proposed values; the person wanted most of them. Rejecting and
 * asking again is the long way round, so the card opens the values as a form
 * built from the tool's own schema, and approval carries what was changed.
 * Only real changes are sent: the server records them on the decision, runs
 * the tool with them, and tells the model what actually ran.
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
      <DialogContent size="md">
        {request && <EditorForm request={request} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}

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
  const reasonMissing = request.withReason?.required === true && reason.trim() === "";
  const canConfirm =
    !isPending && Object.keys(errors).length === 0 && changeCount > 0 && !reasonMissing;

  const set = (name: string, value: string) => {
    setDraft((current) => ({ ...current, [name]: value }));
    setTouched((current) => (current[name] ? current : { ...current, [name]: true }));
  };

  const confirm = async () => {
    setSubmitted(true);
    if (!canConfirm) {
      return;
    }
    setIsPending(true);
    setFailure(null);
    try {
      await request.onConfirm(changes, reason.trim());
      onClose();
    } catch (error) {
      setFailure(error instanceof Error ? error.message : t("The change could not be saved."));
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

      <div className="flex max-h-[60vh] flex-col gap-3 overflow-y-auto py-1">
        {request.fields.map((field) => {
          const id = `${idPrefix}-${field.name}`;
          const error = touched[field.name] || submitted ? errors[field.name] : undefined;

          return (
            <div key={field.name} className="flex flex-col gap-1.5">
              <Label htmlFor={id}>
                {field.label}
                {field.required ? "" : ` (${t("optional")})`}
              </Label>
              <FieldControl
                id={id}
                field={field}
                value={draft[field.name] ?? ""}
                invalid={error !== undefined}
                onChange={(value) => set(field.name, value)}
              />
              {field.description !== "" && (
                <p className="text-muted-foreground text-xs">{field.description}</p>
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
  onChange,
}: {
  id: string;
  field: ProposalField;
  value: string;
  invalid: boolean;
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
          className={cn(field.kind === "JSON" && "font-mono text-xs")}
        />
      );
    case "Boolean":
      return (
        <div className="flex items-center gap-2">
          <Switch
            id={id}
            checked={value === "true"}
            onCheckedChange={(checked) => onChange(checked ? "true" : "false")}
          />
          <span className="text-muted-foreground text-xs">
            {value === "true" ? t("Yes") : t("No")}
          </span>
        </div>
      );
    case "Choice":
      return (
        <Select value={value === "" ? null : value} onValueChange={(next) => onChange(next ?? "")}>
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
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={invalid}
        />
      );
    case "List":
      return (
        <Input
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={
            field.options.length > 0 ? field.options.join(", ") : t("Comma-separated values")
          }
          aria-invalid={invalid}
        />
      );
    default:
      return (
        <Input
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          maxLength={field.maxLength ?? undefined}
          aria-invalid={invalid}
        />
      );
  }
}
