import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { builtinStepLabel, builtinStepRule } from "@/lib/case-checklist-labels";
import { formatCasePosition } from "@/lib/case-checklist-steps";
import type { ChecklistItemMode, ChecklistKind, TemplateItem } from "@/types/case-checklist";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Seg } from "../desk-seg";
import { CustomStepForm } from "./custom-step-form";

/**
 * One step a person sets: its place on the case, its name and what it
 * checks, and whether it is required, optional or off. A step the
 * organization added is marked as theirs, opens to its form, and can be
 * removed; one that failed a save says it needs a fix.
 */
export function ChecklistStepRow({
  index,
  item,
  kind,
  position,
  open,
  readOnly,
  invalid,
  documentName,
  onDocumentPicked,
  onModeChange,
  onToggle,
  onRemove,
}: {
  index: number;
  item: TemplateItem;
  kind: ChecklistKind;
  position: number | undefined;
  open: boolean;
  readOnly: boolean;
  invalid: boolean;
  documentName: string | undefined;
  onDocumentPicked: (id: string, name: string) => void;
  onModeChange: (mode: ChecklistItemMode) => void;
  onToggle: () => void;
  onRemove: () => void;
}) {
  const t = useT();
  const custom = item.custom;
  const title = custom ? custom.label.trim() || t("New step") : builtinStepLabel(item.key, t);
  const note = custom
    ? custom.check === "Document"
      ? documentName
        ? t("Ticked once a {0} is on file", documentName)
        : t("Ticked once the document is on file")
      : t("Ticked by a person on the case")
    : builtinStepRule(item.key, t);

  return (
    <li
      className={cn(
        "dk-ck-r",
        item.mode === "Off" && "dk-off",
        open && "dk-open",
        invalid && "dk-bad",
      )}
    >
      <div className="dk-ck-rl">
        <span className="dk-ck-n">{formatCasePosition(position)}</span>
        <button
          type="button"
          className="dk-ck-t"
          disabled={!custom}
          aria-expanded={custom ? open : undefined}
          onClick={custom ? onToggle : undefined}
        >
          <b>
            {title}
            {custom && <em>{t("Yours")}</em>}
            {invalid && <em className="dk-err">{t("Needs a fix")}</em>}
          </b>
          <span>{note}</span>
        </button>
        {custom && !readOnly && (
          <span className="dk-ck-ra">
            <button
              type="button"
              className="dk-ib"
              aria-label={open ? t("Close {0}", title) : t("Edit {0}", title)}
              aria-pressed={open}
              onClick={onToggle}
            >
              <DeskIcon name="edit" size={14} />
            </button>
            <button
              type="button"
              className="dk-ib"
              aria-label={t("Remove {0}", title)}
              onClick={onRemove}
            >
              <DeskIcon name="trash" size={14} />
            </button>
          </span>
        )}
        <Seg<ChecklistItemMode>
          small
          label={t("How {0} counts", title)}
          value={item.mode}
          disabled={readOnly}
          onChange={onModeChange}
          options={[
            ["Required", t("Required")],
            ["Optional", t("Optional")],
            ["Off", t("Off")],
          ]}
        />
      </div>
      {open && custom && !readOnly && (
        <CustomStepForm
          index={index}
          kind={kind}
          onDocumentPicked={onDocumentPicked}
          onDone={onToggle}
          onRemove={onRemove}
        />
      )}
    </li>
  );
}
