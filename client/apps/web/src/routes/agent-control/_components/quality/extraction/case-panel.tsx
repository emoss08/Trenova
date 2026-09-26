import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  EXTRACTION_EVAL_CASE_DETAIL_KEY,
  fetchExtractionEvalCase,
  type ExtractionEvalCaseDetail,
  type ExtractionEvalCaseRow,
} from "@/lib/graphql/extraction-eval";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AlertTriangleIcon, Trash2Icon } from "lucide-react";
import { useId, useState } from "react";
import { CaseStatusBadge } from "./extraction-badges";
import { CASE_STATUS_ACTION, documentKindLabel, nextCaseStatuses } from "./extraction-model";
import { SnapshotView } from "./snapshot-view";
import { useCaseMutations } from "./use-case-mutations";

const MAX_TITLE = 200;
const MAX_NOTES = 2000;

function CaseDetails({ evalCase }: { evalCase: ExtractionEvalCaseDetail }) {
  const t = useT();

  return (
    <DescriptionList columns={3}>
      <DescriptionItem label={t("Status")}>
        <CaseStatusBadge value={evalCase.status} t={t} />
      </DescriptionItem>
      <DescriptionItem label={t("Document")}>
        {documentKindLabel(evalCase.documentKind, t)}
      </DescriptionItem>
      <DescriptionItem label={t("File")}>{evalCase.fileName || "—"}</DescriptionItem>
      <DescriptionItem label={t("Issuer")}>{evalCase.documentFingerprint || "—"}</DescriptionItem>
      <DescriptionItem label={t("Pages frozen")} numeric>
        {evalCase.pageCount}
      </DescriptionItem>
      <DescriptionItem label={t("Added")}>{formatUnixDateTime(evalCase.createdAt)}</DescriptionItem>
    </DescriptionList>
  );
}

type CaseEditorProps = {
  evalCase: ExtractionEvalCaseDetail;
  canUpdate: boolean;
  saving: boolean;
  onSave: (title: string, notes: string) => void;
  formId: string;
};

function CaseEditor({ evalCase, canUpdate, saving, onSave, formId }: CaseEditorProps) {
  const t = useT();
  const titleId = useId();
  const notesId = useId();
  const [title, setTitle] = useState(evalCase.title);
  const [notes, setNotes] = useState(evalCase.notes);

  const titleValid = title.trim().length > 0 && title.length <= MAX_TITLE;
  const dirty = title.trim() !== evalCase.title || notes.trim() !== evalCase.notes;

  return (
    <form
      id={formId}
      className="grid gap-3"
      onSubmit={(event) => {
        event.preventDefault();
        if (dirty && titleValid && !saving) {
          onSave(title.trim(), notes.trim());
        }
      }}
    >
      <div className="grid gap-1.5">
        <Label htmlFor={titleId}>{t("Title")}</Label>
        <Input
          id={titleId}
          value={title}
          maxLength={MAX_TITLE}
          disabled={!canUpdate}
          aria-invalid={!titleValid}
          onChange={(event) => setTitle(event.target.value)}
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor={notesId}>{t("Notes")}</Label>
        <Textarea
          id={notesId}
          value={notes}
          maxLength={MAX_NOTES}
          disabled={!canUpdate}
          placeholder={t("Why this case is in the set, or what makes it hard")}
          onChange={(event) => setNotes(event.target.value)}
        />
      </div>
    </form>
  );
}

const FORM_ID = "extraction-eval-case-form";

/**
 * One evaluation case: its frozen document, the values it is scored against,
 * and its place in the lifecycle. The title and notes are the only things a
 * person edits; the input and the answer key stay as they were captured.
 */
export function CasePanel({ open, onOpenChange, row }: DataTablePanelProps<ExtractionEvalCaseRow>) {
  const t = useT();
  const id = row?.id;
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { update, remove } = useCaseMutations();
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  const detail = useQuery({
    queryKey: [EXTRACTION_EVAL_CASE_DETAIL_KEY, id],
    queryFn: ({ signal }) => fetchExtractionEvalCase(id ?? "", { signal }),
    enabled: open && id !== undefined,
  });
  const evalCase = detail.data;
  const savingEdits = update.isPending && update.variables?.input.status === undefined;

  return (
    <>
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={evalCase?.title ?? t("Evaluation case")}
        size="xl"
        headerActions={
          evalCase && canUpdate ? (
            <div className="flex items-center gap-2">
              {nextCaseStatuses(evalCase.status).map((status) => (
                <Button
                  key={status}
                  type="button"
                  size="sm"
                  variant={status === "Retired" ? "outline" : "secondary"}
                  isLoading={update.isPending && update.variables?.input.status === status}
                  onClick={() =>
                    update.mutate({ id: evalCase.id, input: { version: evalCase.version, status } })
                  }
                >
                  {t(CASE_STATUS_ACTION[status])}
                </Button>
              ))}
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setConfirmingDelete(true)}
              >
                <Trash2Icon />
                {t("Delete")}
              </Button>
            </div>
          ) : undefined
        }
        footer={
          canUpdate ? (
            <>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Close")}
              </Button>
              <Button type="submit" form={FORM_ID} isLoading={savingEdits}>
                {t("Save changes")}
              </Button>
            </>
          ) : undefined
        }
      >
        {detail.isLoading || !evalCase ? (
          <ComponentLoader />
        ) : (
          <div className="flex flex-col gap-4">
            <CaseDetails evalCase={evalCase} />
            <CaseEditor
              key={`${evalCase.id}:${evalCase.version}`}
              evalCase={evalCase}
              canUpdate={canUpdate}
              saving={savingEdits}
              formId={FORM_ID}
              onSave={(title, notes) =>
                update.mutate({
                  id: evalCase.id,
                  input: { version: evalCase.version, title, notes },
                })
              }
            />
            <SnapshotView snapshot={evalCase.expected} title={t("Confirmed values")} />
          </div>
        )}
      </DataTablePanelContainer>

      <AlertDialog open={confirmingDelete} onOpenChange={setConfirmingDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <AlertTriangleIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Delete evaluation case")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Delete {0}, its frozen document text, and every run result that scored it. Retire it instead to keep its history.",
                evalCase?.title ?? t("this case"),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={remove.isPending}
              onClick={() => {
                if (!evalCase) return;
                remove.mutate(evalCase.id, {
                  onSuccess: () => {
                    setConfirmingDelete(false);
                    onOpenChange(false);
                  },
                });
              }}
            >
              {t("Delete case")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
