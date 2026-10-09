import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { checklistDifferences } from "@/lib/case-checklist-steps";
import { queries } from "@/lib/queries";
import type { ChecklistKind, ChecklistTemplate } from "@/types/case-checklist";
import { useQuery } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useCallback, useState, type ReactNode } from "react";
import { onRadioArrows } from "../../use-modal-focus";
import { ChecklistEditor } from "./checklist-editor";
import { CustomerFinder } from "./customer-finder";

const KINDS: readonly ChecklistKind[] = ["ReadyToBill", "ReadyToClose"];

/** Whose checklist is open: the organization's, or one customer's, saved or not yet. */
type Selection = { customerId: string | null; name: string };

const ORGANIZATION: Selection = { customerId: null, name: "" };

function kindTitle(kind: ChecklistKind, t: TranslateFn): string {
  return kind === "ReadyToBill" ? t("Ready to bill") : t("Ready to close");
}

function kindCaption(kind: ChecklistKind, t: TranslateFn): string {
  return kind === "ReadyToBill" ? t("Shipments") : t("Invoices");
}

/**
 * Case checklists, in the Desk's settings: whose checklist on the left (the
 * organization's, and each customer that bills differently), the one being
 * edited on the right. Switching to another while the open one has unsaved
 * changes is refused, and the save bar says why.
 */
export function CaseChecklistsSection({ readOnly }: { readOnly: boolean }) {
  const t = useT();
  const [kind, setKind] = useState<ChecklistKind>("ReadyToBill");
  const [selected, setSelected] = useState<Selection>(ORGANIZATION);
  const [dirty, setDirty] = useState(false);
  const [nudge, setNudge] = useState(0);
  const [adding, setAdding] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const clearNotice = useCallback(() => setNotice(null), []);
  const query = useQuery(queries.caseChecklist.list(kind));
  const data = query.data;

  const guard = (change: () => void) => {
    if (dirty) {
      setNudge((count) => count + 1);
      return;
    }
    setNotice(null);
    change();
  };
  const pickKind = (next: ChecklistKind) =>
    guard(() => {
      setKind(next);
      setSelected(ORGANIZATION);
    });
  const pick = (next: Selection) => guard(() => setSelected(next));

  if (!data) {
    return (
      <div className="dk-ck dk-ck-loading" aria-busy>
        {query.isError ? (
          <p className="dk-ck-empty">{t("The checklists could not be loaded.")}</p>
        ) : (
          <span className="dk-ck-skel" />
        )}
      </div>
    );
  }

  const saved =
    selected.customerId === null
      ? data.organization
      : data.customers.find((template) => template.customerId === selected.customerId);
  const isNew = selected.customerId !== null && !saved;
  const template: ChecklistTemplate = saved ?? {
    ...data.organization,
    id: "",
    version: 0,
    customerId: selected.customerId ?? "",
    customerName: selected.name,
  };

  return (
    <div className="dk-ck">
      <aside className="dk-ck-side">
        <div
          className="dk-ck-kind"
          role="radiogroup"
          aria-label={t("Checklist")}
        >
          {KINDS.map((option) => (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={kind === option}
              tabIndex={kind === option ? 0 : -1}
              className={cn(kind === option && "dk-on")}
              onClick={() => pickKind(option)}
              onKeyDown={(event) => onRadioArrows(event, KINDS, kind, pickKind)}
            >
              <b>{kindTitle(option, t)}</b>
              <span>{kindCaption(option, t)}</span>
            </button>
          ))}
        </div>

        <div className="dk-ck-gh">{t("Your organization")}</div>
        <WhoRow
          active={selected.customerId === null}
          organization
          tile={<DeskIcon name="home" size={13} />}
          title={t("Every customer")}
          note={data.organization.id ? t("Customized") : t("Trenova's default")}
          onClick={() => pick(ORGANIZATION)}
        />

        <div className="dk-ck-gh">
          {t("Customers with their own")}
          {data.customers.length > 0 && <i>{data.customers.length}</i>}
        </div>
        {data.customers.map((customer) => {
          const differences = checklistDifferences(customer.items, data.organization.items);
          return (
            <WhoRow
              key={customer.id}
              active={selected.customerId === customer.customerId}
              tile={getNameInitials(customer.customerName, "?")}
              title={customer.customerName || t("Customer")}
              note={
                differences === 0
                  ? t("Same as yours")
                  : t(
                      "{0, plural, one {# difference from yours} other {# differences from yours}}",
                      differences,
                    )
              }
              onClick={() =>
                pick({ customerId: customer.customerId, name: customer.customerName })
              }
            />
          );
        })}
        {isNew && (
          <div className="dk-ck-who dk-on dk-new" aria-current="true">
            <span className="dk-ck-tile">{getNameInitials(selected.name, "?")}</span>
            <span className="dk-ck-wt">
              <b>{selected.name}</b>
              <span>{t("Not saved yet")}</span>
            </span>
          </div>
        )}
        {data.customers.length === 0 && !isNew && (
          <p className="dk-ck-empty">
            {t("A customer that bills differently can have its own checklist in place of yours.")}
          </p>
        )}
        {!readOnly &&
          (adding ? (
            <CustomerFinder
              existing={data.customers}
              onClose={() => setAdding(false)}
              onPick={(customer) => {
                setAdding(false);
                pick({ customerId: customer.id, name: customer.name });
              }}
            />
          ) : (
            <button type="button" className="dk-ck-add" onClick={() => setAdding(true)}>
              <DeskIcon name="plus" size={13} />
              {t("Add a customer")}
            </button>
          ))}
      </aside>

      <ChecklistEditor
        key={`${template.customerId || "organization"}:${template.id || "new"}:${template.version}`}
        kind={kind}
        template={template}
        locked={data.locked}
        isNew={isNew}
        readOnly={readOnly}
        nudge={nudge}
        notice={notice}
        onNoticeDone={clearNotice}
        onDirtyChange={setDirty}
        onSaved={(next, message) => {
          setNotice(message);
          setSelected({ customerId: next.customerId || null, name: next.customerName });
        }}
        onLeave={() => {
          setDirty(false);
          setSelected(ORGANIZATION);
        }}
      />
    </div>
  );
}

function WhoRow({
  active,
  organization = false,
  tile,
  title,
  note,
  onClick,
}: {
  active: boolean;
  organization?: boolean;
  tile: ReactNode;
  title: string;
  note: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn("dk-ck-who", active && "dk-on")}
      aria-current={active ? "true" : undefined}
      onClick={onClick}
    >
      <span className={cn("dk-ck-tile", organization && "dk-org")}>{tile}</span>
      <span className="dk-ck-wt">
        <b>{title}</b>
        <span>{note}</span>
      </span>
    </button>
  );
}
