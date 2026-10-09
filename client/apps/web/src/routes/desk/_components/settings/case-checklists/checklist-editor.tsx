import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { builtinStepLabel, builtinStepRule } from "@/lib/case-checklist-labels";
import {
  casePositions,
  checklistDifferences,
  formatCasePosition,
  newCustomStepKey,
} from "@/lib/case-checklist-steps";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  checklistTemplateSchema,
  type ChecklistKind,
  type ChecklistTemplate,
  type TemplateItem,
} from "@/types/case-checklist";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useState } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type FieldErrors } from "react-hook-form";
import { toast } from "sonner";
import { ChecklistReorder } from "./checklist-reorder";
import { ChecklistStepRow } from "./checklist-step-row";
import { checklistEditorSchema, MAX_ADDED_STEPS, type ChecklistEditorValues } from "./editor-schema";

const SAVED_NOTICE_MS = 2400;

/** Where a locked step's rule is kept, and the page that keeps it. */
const LOCKED_SOURCES: Record<string, { page: string; place: "profiles" | "control" }> = {
  pod: { page: "/billing/configuration-files/customers", place: "profiles" },
  paperwork: { page: "/billing/configuration-files/customers", place: "profiles" },
  rateConfirmation: { page: "/admin/billing-controls", place: "control" },
};

function editorDescription(
  kind: ChecklistKind,
  customerName: string,
  isCustomer: boolean,
  isNew: boolean,
  t: TranslateFn,
): string {
  const bill = kind === "ReadyToBill";
  if (!isCustomer) {
    return bill
      ? t("Every shipment case shows these steps between the record and the bill, unless the customer has its own.")
      : t("Every invoice case shows these steps between the record and closing, unless the customer has its own.");
  }
  if (isNew) {
    return bill
      ? t("Starts as a copy of your organization's. Save it to use it for {0}'s shipments.", customerName)
      : t("Starts as a copy of your organization's. Save it to use it for {0}'s invoices.", customerName);
  }
  return bill
    ? t("Used for {0}'s shipment cases in place of your organization's.", customerName)
    : t("Used for {0}'s invoice cases in place of your organization's.", customerName);
}

/**
 * One checklist being edited: the steps a person sets, the steps that follow
 * rules kept elsewhere, a Reorder mode for the order the case shows them in,
 * and a save bar that counts what is unsaved. It is remounted for each
 * template and each saved version, so what it holds is only ever one
 * checklist's draft.
 */
export function ChecklistEditor({
  kind,
  template,
  locked,
  isNew,
  readOnly,
  nudge,
  notice,
  onNoticeDone,
  onDirtyChange,
  onSaved,
  onLeave,
}: {
  kind: ChecklistKind;
  template: ChecklistTemplate;
  locked: readonly string[];
  isNew: boolean;
  readOnly: boolean;
  /** Counts refused switches; each new one shakes the save bar. */
  nudge: number;
  /** What the save bar says for a moment after a save, carried over the remount. */
  notice: string | null;
  onNoticeDone: () => void;
  onDirtyChange: (dirty: boolean) => void;
  onSaved: (template: ChecklistTemplate, notice: string) => void;
  /** Back to the organization's checklist: a customer's removed, or a new one discarded. */
  onLeave: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const isCustomer = Boolean(template.customerId);
  const customerName = template.customerName || t("Customer");
  const [openKey, setOpenKey] = useState<string | null>(null);
  const [reorder, setReorder] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [pickedNames, setPickedNames] = useState<Record<string, string>>({});
  const [nudgeAtMount] = useState(nudge);
  const nudged = nudge > nudgeAtMount;

  const form = useForm<ChecklistEditorValues>({
    resolver: zodResolver(checklistEditorSchema),
    defaultValues: { items: template.items },
  });
  const { control, handleSubmit, reset, setValue, formState } = form;
  const { fields, append, remove, move } = useFieldArray({ control, name: "items" });
  const watched = useWatch({ control, name: "items" });
  const items = watched ?? template.items;

  const differences = checklistDifferences(items, template.items);
  const dirty = !readOnly && (isNew || differences > 0);
  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);
  useEffect(() => () => onDirtyChange(false), [onDirtyChange]);

  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(onNoticeDone, SAVED_NOTICE_MS);
    return () => window.clearTimeout(timer);
  }, [notice, onNoticeDone]);

  const documentTypeIds = useMemo(
    () =>
      [
        ...new Set(
          items.flatMap((item) => (item.custom?.documentTypeId ? [item.custom.documentTypeId] : [])),
        ),
      ].filter((id) => !pickedNames[id]),
    [items, pickedNames],
  );
  const namesQuery = useQuery({
    ...queries.caseChecklist.documentTypes(documentTypeIds),
    enabled: documentTypeIds.length > 0,
  });
  const documentNames = useMemo(() => {
    const names: Record<string, string> = { ...pickedNames };
    for (const option of namesQuery.data?.results ?? []) {
      names[option.id] = option.label;
    }
    return names;
  }, [namesQuery.data, pickedNames]);

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: queries.caseChecklist.list(kind).queryKey });

  const save = useApiMutation({
    mutationFn: (values: ChecklistEditorValues) =>
      apiService.caseChecklistService.save({
        id: template.id || undefined,
        version: template.version,
        kind,
        customerId: template.customerId || undefined,
        items: values.items,
      }),
    form,
    resourceName: "Case Checklist",
    onSuccess: async (saved) => {
      const next = checklistTemplateSchema.parse({
        ...saved,
        customerName: saved.customerName || template.customerName,
      });
      reset({ items: next.items });
      await refresh();
      onSaved(next, t("Checklist saved"));
    },
  });

  const removeTemplate = useApiMutation({
    mutationFn: () => apiService.caseChecklistService.remove(template.id),
    resourceName: "Case Checklist",
    onSuccess: async () => {
      toast.success(
        isCustomer
          ? t("{0} now uses your organization's checklist", customerName)
          : t("Your organization uses Trenova's default checklist again"),
      );
      await refresh();
      if (isCustomer) {
        onLeave();
      }
    },
  });

  const onInvalid = (errors: FieldErrors<ChecklistEditorValues>) => {
    const index = (errors.items as unknown[] | undefined)?.findIndex(Boolean) ?? -1;
    setReorder(false);
    if (index >= 0) {
      setOpenKey(items[index]?.key ?? null);
    }
  };
  const submit = handleSubmit((values) => save.mutateAsync(values), onInvalid);

  const discard = () => {
    if (isNew) {
      onLeave();
      return;
    }
    reset({ items: template.items });
    setOpenKey(null);
  };

  const addStep = () => {
    const step: TemplateItem = {
      key: newCustomStepKey(),
      mode: "Required",
      custom: { label: "", check: "Manual", documentTypeId: "", stepLabel: "", prompt: "" },
    };
    append(step);
    setReorder(false);
    setOpenKey(step.key);
  };

  const positions = casePositions(items);
  const added = items.filter((item) => item.custom).length;
  const counts = {
    required: items.filter((item) => item.mode === "Required").length,
    optional: items.filter((item) => item.mode === "Optional").length,
    off: items.filter((item) => item.mode === "Off").length,
  };
  const itemErrors = formState.isSubmitted
    ? ((formState.errors.items as Array<unknown> | undefined) ?? [])
    : [];
  const resettable = isCustomer ? !isNew : Boolean(template.id);
  const bill = kind === "ReadyToBill";

  return (
    <FormProvider {...form}>
      <section className="dk-ck-main">
        <header className="dk-ck-h">
          <div className="dk-ck-ht">
            <h3>
              {isCustomer ? t("{0}'s checklist", customerName) : t("Your organization's checklist")}
            </h3>
            <p>{editorDescription(kind, customerName, isCustomer, isNew, t)}</p>
          </div>
          {!readOnly && (
            <div className="dk-ck-ha">
              <button
                type="button"
                className={cn("dk-ck-btn", reorder && "dk-on")}
                aria-pressed={reorder}
                onClick={() => {
                  setReorder((value) => !value);
                  setOpenKey(null);
                }}
              >
                <DeskIcon name={reorder ? "check" : "rail"} size={13} />
                {reorder ? t("Done") : t("Reorder")}
              </button>
              {resettable && (
                <button
                  type="button"
                  className={cn("dk-ck-btn", confirming && "dk-danger")}
                  disabled={removeTemplate.isPending}
                  onBlur={() => setConfirming(false)}
                  onClick={() => {
                    if (!confirming) {
                      setConfirming(true);
                      return;
                    }
                    setConfirming(false);
                    removeTemplate.mutate();
                  }}
                >
                  {confirming
                    ? t("Click again to confirm")
                    : isCustomer
                      ? t("Remove this checklist")
                      : t("Go back to the default")}
                </button>
              )}
            </div>
          )}
        </header>

        <div className="dk-ck-sum">
          <span>
            <i className="dk-ck-dot dk-m-req" />
            {t("{0} required", counts.required)}
          </span>
          <span>
            <i className="dk-ck-dot dk-m-opt" />
            {t("{0} optional", counts.optional)}
          </span>
          {counts.off > 0 && (
            <span>
              <i className="dk-ck-dot dk-m-off" />
              {t("{0} off", counts.off)}
            </span>
          )}
          <span className="dk-sp" />
          {!reorder && <span className="dk-ck-sumh">{t("Numbers are the order the case shows.")}</span>}
        </div>

        <div className="dk-ck-scroll" key={reorder ? "reorder" : "steps"}>
          {reorder ? (
            <>
              <p className="dk-ck-lead">
                {t(
                  "Drag steps into the order your team works them, or use the arrow keys on a handle. Steps that are off aren't shown on the case.",
                )}
              </p>
              <ChecklistReorder
                items={items}
                sortIds={fields.map((field) => field.id)}
                positions={positions}
                locked={locked}
                onMove={move}
              />
            </>
          ) : (
            <>
              <div className="dk-ck-sec">
                <div className="dk-ck-st">
                  <b>{t("Steps you set")}</b>
                  <span>{t("Choose how each one counts on the case.")}</span>
                </div>
                <ol className="dk-ck-list">
                  {fields.map((field, index) => {
                    if (locked.includes(field.key)) {
                      return null;
                    }
                    const item = items[index] ?? field;
                    return (
                      <ChecklistStepRow
                        key={field.id}
                        index={index}
                        item={item}
                        kind={kind}
                        position={positions.get(item.key)}
                        open={openKey === item.key}
                        readOnly={readOnly}
                        invalid={Boolean(itemErrors[index])}
                        documentName={
                          item.custom?.documentTypeId
                            ? documentNames[item.custom.documentTypeId]
                            : undefined
                        }
                        onDocumentPicked={(id, name) =>
                          setPickedNames((current) => ({ ...current, [id]: name }))
                        }
                        onModeChange={(mode) =>
                          setValue(`items.${index}.mode`, mode, { shouldDirty: true })
                        }
                        onToggle={() =>
                          setOpenKey((current) => (current === item.key ? null : item.key))
                        }
                        onRemove={() => {
                          remove(index);
                          if (openKey === item.key) setOpenKey(null);
                        }}
                      />
                    );
                  })}
                </ol>
                {!readOnly && (
                  <div className="dk-ck-addrow">
                    <button
                      type="button"
                      className="dk-ck-btn"
                      disabled={added >= MAX_ADDED_STEPS}
                      onClick={addStep}
                    >
                      <DeskIcon name="plus" size={13} />
                      {t("Add a step")}
                    </button>
                    <span>
                      {added >= MAX_ADDED_STEPS
                        ? t("A checklist can have at most 12 added steps.")
                        : bill
                          ? t(
                              "Ticked by a person on the case, or by a document on file. {0} of 12 added.",
                              added,
                            )
                          : t("Ticked by a person on the case. {0} of 12 added.", added)}
                    </span>
                  </div>
                )}
              </div>

              <div className="dk-ck-sec dk-lk">
                <div className="dk-ck-st">
                  <b>
                    <DeskIcon name="lock" size={12} />
                    {t("Always required")}
                  </b>
                  <span>
                    {t(
                      "These follow rules kept elsewhere. You can move them with Reorder, but not turn them off.",
                    )}
                  </span>
                </div>
                <ol className="dk-ck-list">
                  {items
                    .filter((item) => locked.includes(item.key))
                    .map((item) => {
                      const source = LOCKED_SOURCES[item.key];
                      return (
                        <li key={item.key} className="dk-ck-r dk-lk">
                          <div className="dk-ck-rl">
                            <span className="dk-ck-n">
                              {formatCasePosition(positions.get(item.key))}
                            </span>
                            <div className="dk-ck-t">
                              <b>{builtinStepLabel(item.key, t)}</b>
                              <span>{builtinStepRule(item.key, t)}</span>
                            </div>
                            {source ? (
                              <a
                                className="dk-ck-src"
                                href={source.page}
                                target="_blank"
                                rel="noreferrer"
                              >
                                {source.place === "profiles"
                                  ? t("Set in Billing profiles")
                                  : t("Set in Billing control")}
                                <DeskIcon name="ext" size={11} />
                              </a>
                            ) : (
                              <span className="dk-ck-src dk-plain">
                                {bill ? t("Needed for the bill") : t("Needed for closing")}
                              </span>
                            )}
                          </div>
                        </li>
                      );
                    })}
                </ol>
              </div>
            </>
          )}
        </div>

        {!readOnly && (dirty || notice) && (
          <div className={cn("dk-ck-bar", !dirty && "dk-ok")} key={dirty ? `d${nudge}` : "notice"}>
            {dirty ? (
              <>
                <span className={cn("dk-ck-bm", nudged && "dk-nudge")}>
                  <i />
                  {isNew
                    ? t("{0}'s checklist isn't saved yet", customerName)
                    : t(
                        "{0, plural, one {# unsaved change} other {# unsaved changes}}",
                        differences,
                      )}
                  {nudged && <em>{t("Save or discard before switching")}</em>}
                </span>
                <button type="button" className="dk-ck-btn" onClick={discard}>
                  {t("Discard")}
                </button>
                <button
                  type="button"
                  className="dk-ck-btn dk-pri"
                  disabled={save.isPending}
                  onClick={() => void submit()}
                >
                  {isNew ? t("Save checklist") : t("Save changes")}
                </button>
              </>
            ) : (
              <span className="dk-ck-bm dk-ok" role="status">
                <DeskIcon name="check" size={13} />
                {notice}
              </span>
            )}
          </div>
        )}
      </section>
    </FormProvider>
  );
}
