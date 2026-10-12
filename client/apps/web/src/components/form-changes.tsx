import { isPlainList, listDelta, previewValue } from "@/lib/edit-diff";
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  ChevronUpIcon,
  FlipBackwardIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { InfoPopover } from "./info-popover";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment, toTitleCase } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
  type RefObject,
} from "react";
import {
  get,
  useFormContext,
  useFormState,
  useWatch,
  type FieldPath,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import {
  createFieldRegistry,
  SectionFieldsProvider,
  type FieldRegistry,
  type FieldValueFormat,
} from "@trenova/shared/lib/form-field-registry";

/** How a form names and shows one of its fields in a change review. */
export type ChangeField = {
  label: string;
  /** Writes a value; the default handles text, numbers, switches and lists. */
  format?: (value: unknown) => ReactNode;
  /** Names one item of a list field, such as a task by its label. */
  item?: (value: string | number) => string;
  /**
   * How the field itself writes a value, registered by the field (a record's name
   * for its ID, an option's label for its key); used where nothing above is given.
   */
  value?: FieldValueFormat;
};

export type ChangeFields = Record<string, ChangeField>;

const NO_CHANGE_FIELDS: ChangeFields = {};
const NO_FIELDS: readonly string[] = [];

/** A field named by its key, in sentence case, for a form that gave no labels. */
function fallbackLabel(key: string): string {
  const words = toSentenceFragment(toTitleCase(key));
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/**
 * The top-level fields that differ from what the form opened with. The form's
 * dirty fields are read on every render, because react-hook-form changes that
 * object in place.
 */
function useChangedFields<T extends FieldValues>(form: UseFormReturn<T, any, any>) {
  const { dirtyFields } = useFormState({ control: form.control });
  return Object.keys(dirtyFields).filter((key) => key !== "version");
}

type FormChangesFooterProps<T extends FieldValues> = {
  form: UseFormReturn<T, any, any>;
  fields: ChangeFields;
  /** The fields the panel drew, which name a change the way the form does. */
  registry?: FieldRegistry;
  create: boolean;
  /** Why the form cannot be saved yet, shown beside the changes. */
  problem?: string | null;
};

/**
 * The left of a form panel's footer: how many changes are unsaved, a review of
 * each one before and after with its own undo, and why the form cannot be saved.
 */
export function FormChangesFooter<T extends FieldValues>({
  form,
  fields,
  registry,
  create,
  problem,
}: FormChangesFooterProps<T>) {
  const t = useT();
  const changed = useChangedFields(form);
  const count = changed.length;

  return (
    <div className="flex min-w-0 items-center gap-3">
      {count > 0 ? (
        <UnsavedChanges form={form} changed={changed} fields={fields} registry={registry} />
      ) : (
        <span className="text-muted-foreground text-sm">
          {create ? t("Nothing is created until you save") : t("No changes")}
        </span>
      )}
      {problem && (
        <span className="text-warning-foreground inline-flex min-w-0 items-center gap-1.5 text-sm">
          <AlertCircleIcon className="size-3.5 shrink-0" />
          <span className="truncate">{problem}</span>
        </span>
      )}
    </div>
  );
}

const UNSAVED_TONE = {
  panel: {
    trigger: "text-foreground hover:bg-field data-popup-open:bg-field",
    chevron: "text-muted-foreground",
    offset: 8,
  },
  dock: {
    trigger:
      "text-background hover:bg-background/10 data-popup-open:bg-background/15",
    chevron: "text-background/70",
    // Anchored to the dock itself, so the gap is measured from the dock's edge.
    offset: 6,
  },
} as const;

type UnsavedChangesProps<T extends FieldValues> = {
  form: UseFormReturn<T, any, any>;
  changed: readonly string[];
  fields: ChangeFields;
  registry?: FieldRegistry | null;
  /** The surface it sits on: a panel's footer, or the dark save dock. */
  tone?: keyof typeof UNSAVED_TONE;
  /** What the review centres over, such as the whole dock; the button by default. */
  anchor?: RefObject<Element | null>;
};

/**
 * How many changes are unsaved, opening a review of each one before and after with
 * its own undo. The footer and the save dock both use it, so either way a person
 * reviews their changes the same way.
 */
export function UnsavedChanges<T extends FieldValues>({
  form,
  changed,
  fields,
  registry,
  tone = "panel",
  anchor,
}: UnsavedChangesProps<T>) {
  const t = useT();
  const rt = useRichT();
  const [open, setOpen] = useState(false);
  const [confirmingUndo, setConfirmingUndo] = useState(false);
  const count = changed.length;
  const bold = { b: (chunk: ReactNode) => <b className="font-semibold">{chunk}</b> };

  const askToUndoAll = () => {
    setOpen(false);
    setConfirmingUndo(true);
  };
  const undoAll = () => {
    form.reset();
    setConfirmingUndo(false);
  };

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <button
              type="button"
              className={cn(
                "group ui-focus-ring inline-flex h-7 items-center gap-2 rounded-md px-2.5 text-sm font-medium transition-colors",
                UNSAVED_TONE[tone].trigger,
              )}
            />
          }
        >
          <span className="bg-brand ring-brand/25 size-2 rounded-full ring-3" />
          {count === 1
            ? rt("<b>1</b> unsaved change", bold)
            : rt("<b>{0}</b> unsaved changes", bold, count)}
          <ChevronUpIcon
            className={cn(
              "size-3.5 rotate-180 transition-transform duration-200 group-data-popup-open:rotate-0",
              UNSAVED_TONE[tone].chevron,
            )}
          />
        </PopoverTrigger>
        <PopoverContent
          side="top"
          align={anchor ? "center" : "start"}
          anchor={anchor}
          sideOffset={UNSAVED_TONE[tone].offset}
          className="w-90 max-w-[calc(100vw-2rem)] gap-0 p-0"
        >
          <div className="border-border flex items-baseline justify-between gap-3 border-b px-2.5 py-2">
            <span className="text-xs">
              {count === 1
                ? rt("Review <b>1</b> change", bold)
                : rt("Review <b>{0}</b> changes", bold, count)}
            </span>
            <Button
              type="button"
              variant="link"
              size="xxs"
              className="text-2xs h-auto p-0"
              onClick={askToUndoAll}
            >
              {t("Undo all")}
            </Button>
          </div>
          <ChangeList form={form} changed={changed} fields={fields} registry={registry} />
        </PopoverContent>
      </Popover>
      <AlertDialog open={confirmingUndo} onOpenChange={setConfirmingUndo}>
        <AlertDialogContent className="gap-3">
          <AlertDialogCancel
            variant="ghost"
            size="icon"
            aria-label={t("Close")}
            className="text-muted-foreground hover:text-foreground -mt-1 -ml-1 size-7"
          >
            <XCloseIcon className="size-4" />
          </AlertDialogCancel>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-xl leading-tight font-semibold">
              {count === 1 ? t("Undo 1 change?") : t("Undo {0} changes?", count)}
            </AlertDialogTitle>
            <AlertDialogDescription className="text-foreground">
              {rt("Fields go back to how they opened. <b>This can't be undone.</b>", {
                b: (chunk) => <b className="font-semibold">{chunk}</b>,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel size="sm">{t("Keep changes")}</AlertDialogCancel>
            <AlertDialogAction size="sm" variant="destructive" onClick={undoAll}>
              {t("Undo all")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

type ChangeListProps<T extends FieldValues> = {
  form: UseFormReturn<T, any, any>;
  changed: readonly string[];
  fields: ChangeFields;
  registry?: FieldRegistry | null;
};

/**
 * The review's rows. A component of its own so its lookups and formatting run only
 * while the review is open, never on the footer's renders while it is closed.
 */
function ChangeList<T extends FieldValues>({ form, changed, fields, registry }: ChangeListProps<T>) {
  // Watched rather than read, so a row's "after" follows an edit made while the
  // review is open.
  const values: unknown[] = useWatch({
    control: form.control,
    name: changed as FieldPath<T>[],
  });
  return (
    <ScrollArea viewportClassName="max-h-64" maskVariant="popover">
      <div className="flex flex-col p-1">
        {changed.map((key, index) => {
          const given = fields[key];
          const field: ChangeField = {
            label: given?.label ?? registry?.label(key) ?? fallbackLabel(key),
            format: given?.format,
            item: given?.item,
            value: registry?.format(key),
          };
          return (
            <ChangeRow
              key={key}
              field={field}
              before={get(form.formState.defaultValues, key)}
              after={values[index]}
              onUndo={() => form.resetField(key as FieldPath<T>)}
            />
          );
        })}
      </div>
    </ScrollArea>
  );
}

type ChangeRowProps = {
  field: ChangeField;
  before: unknown;
  after: unknown;
  onUndo: () => void;
};

function ChangeRow({ field, before, after, onUndo }: ChangeRowProps) {
  const t = useT();

  return (
    <div className="hover:bg-muted-foreground/10 grid grid-cols-[minmax(0,7rem)_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md px-1.5 py-1 text-xs transition-colors">
      <span className="truncate opacity-70">{field.label}</span>
      <ChangeValue field={field} before={before} after={after} />
      <button
        type="button"
        className="ui-focus-ring inline-flex size-5 cursor-pointer items-center justify-center rounded-md opacity-70 transition-opacity hover:opacity-100"
        title={t("Undo this change")}
        aria-label={t("Undo the change to {0}", field.label)}
        onClick={onUndo}
      >
        <FlipBackwardIcon className="size-3" />
      </button>
    </div>
  );
}

function ChangeValue({ field, before, after }: Omit<ChangeRowProps, "onUndo">) {
  if (!field.format && isPlainList(before) && isPlainList(after)) {
    const { added, removed } = listDelta(before, after);
    const name =
      field.item ?? ((item: string | number) => field.value?.(item) ?? String(item));
    return (
      <span className="flex min-w-0 flex-wrap gap-x-2 gap-y-0.5">
        {added.map((value) => (
          <span key={`+${value}`} className="text-success">
            + {name(value)}
          </span>
        ))}
        {removed.map((value) => (
          <span key={`-${value}`} className="text-danger line-through">
            − {name(value)}
          </span>
        ))}
      </span>
    );
  }

  const write =
    field.format ?? ((value: unknown) => field.value?.(value) ?? previewValue(value));
  return (
    <span className="flex min-w-0 items-center gap-1.5">
      <span className="max-w-[45%] shrink-0 truncate opacity-60">{write(before)}</span>
      <ArrowRightIcon className="size-2.5 shrink-0 opacity-60" />
      <span className="min-w-0 flex-1 truncate font-medium">{write(after)}</span>
    </span>
  );
}

type FormPanelSectionProps = {
  id: string;
  title: string;
  note?: ReactNode;
  /**
   * Fields the section holds beyond those it draws through the shared fields,
   * which register themselves; a dot marks the section while any is unsaved.
   */
  fields?: readonly string[];
  /** Beside the title, such as a refresh button. */
  actions?: ReactNode;
  /** A longer explanation, opened from beside the title. */
  help?: ReactNode;
  children: ReactNode;
};

/** One titled block of a form panel, marked while it holds unsaved changes. */
export function FormPanelSection({
  id,
  title,
  note,
  fields = NO_FIELDS,
  actions,
  help,
  children,
}: FormPanelSectionProps) {
  const t = useT();
  const { control } = useFormContext();
  const [registry] = useState(createFieldRegistry);
  const drawn = useSyncExternalStore(registry.subscribe, registry.names, registry.names);
  const watched = useMemo(
    () => (fields.length === 0 ? drawn : Array.from(new Set([...fields, ...drawn]))),
    [drawn, fields],
  );
  const { dirtyFields } = useFormState({
    control,
    name: watched as string[],
    disabled: watched.length === 0,
  });
  const changed = watched.some((name) => Boolean(get(dirtyFields, name)));

  return (
    <FormSection
      id={id}
      title={title}
      description={note}
      action={actions}
      titleAside={help ? <InfoPopover title={title}>{help}</InfoPopover> : undefined}
      className="gap-3 py-5 first:pt-0"
      indicator={
        <>
          <span
            aria-hidden
            className={cn(
              "bg-brand -ml-2.5 size-1.5 shrink-0 rounded-full transition-opacity",
              changed ? "opacity-100" : "opacity-0",
            )}
          />
          {changed && <span className="sr-only">{t("Unsaved changes")}</span>}
        </>
      }
    >
      <SectionFieldsProvider registry={registry}>{children}</SectionFieldsProvider>
    </FormSection>
  );
}

/**
 * Closing a form panel with unsaved changes asks first, in the panel's own footer,
 * so an Escape, a click outside or the close button never loses work.
 *
 * The panel itself never subscribes to the form: a PanelDirtyTracker reports how
 * many fields are unsaved into a ref this reads when a close is asked for, so
 * typing re-renders neither the panel nor, through it, the fields in it.
 */
export function usePanelCloseGuard({
  open,
  onClose,
}: {
  open: boolean;
  /** Closes the panel for good, after any confirmation. */
  onClose: () => void;
}) {
  const dirtyCount = useRef(0);
  const [confirming, setConfirming] = useState<number | null>(null);
  const [wasOpen, setWasOpen] = useState(open);
  if (wasOpen !== open) {
    setWasOpen(open);
    if (!open) setConfirming(null);
  }

  const reportDirty = useCallback((count: number) => {
    dirtyCount.current = count;
  }, []);
  const requestClose = useCallback(() => {
    if (dirtyCount.current > 0) {
      setConfirming(dirtyCount.current);
      return;
    }
    onClose();
  }, [onClose]);
  const keepEditing = useCallback(() => setConfirming(null), []);
  const discard = useCallback(() => {
    setConfirming(null);
    onClose();
  }, [onClose]);

  return {
    confirmingClose: open && confirming !== null,
    changedCount: confirming ?? 0,
    reportDirty,
    requestClose,
    keepEditing,
    discard,
  };
}

export type PanelCloseGuard = ReturnType<typeof usePanelCloseGuard>;

/**
 * Renders nothing; keeps the close guard told how many fields are unsaved. It is
 * the only part of a panel that re-renders as fields turn dirty or clean.
 */
export function PanelDirtyTracker<T extends FieldValues>({
  form,
  guard,
}: {
  form: UseFormReturn<T, any, any>;
  guard: PanelCloseGuard;
}) {
  const count = useChangedFields(form).length;
  const { reportDirty } = guard;
  useEffect(() => {
    reportDirty(count);
  }, [count, reportDirty]);
  return null;
}

type FormPanelFooterProps<T extends FieldValues> = {
  form: UseFormReturn<T, any, any>;
  create: boolean;
  guard: PanelCloseGuard;
  /** The fields the panel drew, so the review names them as the form does. */
  registry?: FieldRegistry;
  changeFields?: ChangeFields;
  problem?: string | null;
  leading?: ReactNode;
  /** The save control, at the right. */
  save: ReactNode;
};

/**
 * A form panel's footer: what is unsaved and why it cannot be saved on the left,
 * the save on the right; while closing would lose changes, it asks instead.
 */
export function FormPanelFooter<T extends FieldValues>({
  form,
  create,
  guard,
  registry,
  changeFields,
  problem,
  leading,
  save,
}: FormPanelFooterProps<T>) {
  const t = useT();

  if (guard.confirmingClose) {
    return (
      <div
        className="animate-in fade-in-0 slide-in-from-bottom-1 ease-settle flex w-full items-center gap-2 duration-200 motion-reduce:animate-none"
        role="alert"
      >
        <AlertTriangleIcon className="text-warning size-4 shrink-0" />
        <span className="text-sm font-medium">
          {guard.changedCount === 1
            ? t("Discard 1 unsaved change?")
            : t("Discard {0} unsaved changes?", guard.changedCount)}
        </span>
        <span className="flex-1" />
        <Button type="button" size="sm" variant="outline" autoFocus onClick={guard.keepEditing}>
          {t("Keep editing")}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="border-danger-border text-danger-foreground hover:bg-danger-subtle"
          onClick={guard.discard}
        >
          {t("Discard and close")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex w-full items-center gap-2">
      <div className="mr-auto flex min-w-0 items-center gap-3">
        <FormChangesFooter
          form={form}
          fields={changeFields ?? NO_CHANGE_FIELDS}
          registry={registry}
          create={create}
          problem={problem}
        />
        {leading}
      </div>
      {save}
    </div>
  );
}
