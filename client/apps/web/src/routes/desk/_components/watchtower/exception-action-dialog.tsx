import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  assignWatchtowerItem,
  reopenWatchtowerItem,
  resolveWatchtowerItem,
  setWatchtowerItemDue,
  snoozeWatchtowerItem,
  type WatchtowerItem,
  type WatchtowerResolutionCode,
  type WatchtowerResolutionOption,
} from "@/lib/graphql/watchtower";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import {
  EMPTY_EXCEPTION_ACTION,
  MAX_EXCEPTION_NOTE,
  exceptionActionSchema,
  type ExceptionAction,
  type ExceptionActionValues,
} from "./exception-forms";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

function actionCopy(t: TranslateFn, action: ExceptionAction) {
  switch (action) {
    case "assign":
      return {
        title: t("Assign exception"),
        description: t("Give it to someone who can see the record behind it. They are told."),
        submit: t("Assign"),
        pending: t("Assigning..."),
      };
    case "snooze":
      return {
        title: t("Snooze exception"),
        description: t(
          "Set it aside until a time. It comes back on its own and its owner is told.",
        ),
        submit: t("Snooze"),
        pending: t("Snoozing..."),
      };
    case "due":
      return {
        title: t("Change due time"),
        description: t("Past its due time an exception escalates. Leave it empty to clear it."),
        submit: t("Save due time"),
        pending: t("Saving..."),
      };
    case "resolve":
      return {
        title: t("Resolve exception"),
        description: t(
          "Close it with a reason. It stays closed while the record still shows the problem.",
        ),
        submit: t("Resolve"),
        pending: t("Resolving..."),
      };
    default:
      return {
        title: t("Reopen exception"),
        description: t("Put it back on the open queue with a fresh due time."),
        submit: t("Reopen"),
        pending: t("Reopening..."),
      };
  }
}

function initialValues(item: WatchtowerItem): ExceptionActionValues {
  return {
    ...EMPTY_EXCEPTION_ACTION,
    assigneeId: item.assigneeId ?? "",
    dueAt: item.dueAt ?? null,
  };
}

/**
 * Every change to an exception that needs more than a click: who owns it,
 * when it comes back, when it is due, why it closed, why it reopened. One
 * dialog, so the version travels the same way for each and a stale screen
 * is refused the same way.
 */
export function ExceptionActionDialog({
  item,
  action,
  codes,
  onClose,
  onSaved,
}: {
  item: WatchtowerItem;
  action: ExceptionAction | null;
  codes: readonly WatchtowerResolutionOption[];
  onClose: () => void;
  onSaved: (item: WatchtowerItem) => Promise<void> | void;
}) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const open = action !== null;
  const current = action ?? "reopen";
  const copy = actionCopy(t, current);
  const schema = useMemo(
    () => exceptionActionSchema(t, current, codes, now),
    [t, current, codes, now],
  );
  const form = useForm<ExceptionActionValues>({
    resolver: zodResolver(schema),
    values: initialValues(item),
  });
  const { control, handleSubmit, reset } = form;
  const selectedCode = useWatch({ control, name: "code" });
  const noteRequired =
    current === "resolve" &&
    codes.some((option) => option.code === selectedCode && option.requiresNote);

  const mutation = useApiMutation({
    form,
    resourceName: "Exception",
    mutationFn: (values: ExceptionActionValues) => {
      const change = { id: item.id, version: item.version, note: values.note };
      switch (current) {
        case "assign":
          return assignWatchtowerItem({
            ...change,
            assigneeId: values.assigneeId === "" ? null : values.assigneeId,
          });
        case "snooze":
          return snoozeWatchtowerItem({ ...change, until: values.until ?? 0 });
        case "due":
          return setWatchtowerItemDue({ ...change, dueAt: values.dueAt });
        case "resolve":
          return resolveWatchtowerItem({
            ...change,
            code: values.code as WatchtowerResolutionCode,
          });
        default:
          return reopenWatchtowerItem(change);
      }
    },
    onSuccess: async (saved) => {
      await onSaved(saved);
      reset(EMPTY_EXCEPTION_ACTION);
      onClose();
    },
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>{copy.description}</DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => mutation.mutateAsync(values))(event);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            {current === "assign" && (
              <FormControl>
                <UserAutocompleteField
                  control={control}
                  name="assigneeId"
                  label={t("Owner")}
                  placeholder={t("Nobody")}
                />
              </FormControl>
            )}
            {current === "snooze" && (
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name="until"
                  label={t("Bring it back at")}
                  rules={{ required: true }}
                />
              </FormControl>
            )}
            {current === "due" && (
              <FormControl>
                <AutoCompleteDateTimeField control={control} name="dueAt" label={t("Due at")} />
              </FormControl>
            )}
            {current === "resolve" && (
              <FormControl>
                <SelectField
                  control={control}
                  name="code"
                  label={t("Resolution")}
                  rules={{ required: true }}
                  options={codes.map((option) => ({ value: option.code, label: option.label }))}
                />
              </FormControl>
            )}
            <FormControl>
              <TextareaField
                control={control}
                name="note"
                label={current === "resolve" ? t("How it was resolved") : t("Note")}
                rules={{ required: noteRequired }}
                maxLength={MAX_EXCEPTION_NOTE}
                placeholder={
                  noteRequired ? t("Required for this resolution") : t("Kept in its history")
                }
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={mutation.isPending} loadingText={copy.pending}>
              {copy.submit}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
