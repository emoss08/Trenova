import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { queries } from "@/lib/queries";
import type { ChecklistKind, ChecklistTemplate } from "@/types/case-checklist";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Building07Icon, PlusIcon, User01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { ChecklistTemplateEditor } from "./checklist-template-editor";

type Selection = { customerId: string | null; customerName: string };

const ORGANIZATION: Selection = { customerId: null, customerName: "" };

/**
 * The organization's checklist and each customer's own, side by side with
 * the one being edited. A customer added here starts from the
 * organization's steps and is saved as its own the first time.
 */
export default function ChecklistWorkspace({ kind }: { kind: ChecklistKind }) {
  const t = useT();
  const { data } = useSuspenseQuery(queries.caseChecklist.list(kind));
  const [selected, setSelected] = useState<Selection>(ORGANIZATION);
  const [adding, setAdding] = useState(false);
  const picker = useForm<{ customerId: string }>({ defaultValues: { customerId: "" } });

  const saved = data.customers.find((template) => template.customerId === selected.customerId);
  const template: ChecklistTemplate =
    selected.customerId === null
      ? data.organization
      : (saved ?? {
          ...data.organization,
          id: "",
          version: 0,
          customerId: selected.customerId,
          customerName: selected.customerName,
        });

  return (
    <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
      <nav aria-label={t("Checklists")} className="flex flex-col gap-1">
        <TemplateRow
          active={selected.customerId === null}
          icon={<Building07Icon className="size-4" />}
          title={t("Your organization")}
          note={data.organization.id ? t("Customized") : t("Trenova's default")}
          onClick={() => setSelected(ORGANIZATION)}
        />
        {data.customers.length > 0 && (
          <p className="text-muted-foreground px-2 pt-3 pb-1 text-xs font-medium">
            {t("Customers with their own")}
          </p>
        )}
        {data.customers.map((customer) => (
          <TemplateRow
            key={customer.id}
            active={selected.customerId === customer.customerId}
            icon={<User01Icon className="size-4" />}
            title={customer.customerName || t("Customer")}
            note={t("Replaces your organization's")}
            onClick={() =>
              setSelected({
                customerId: customer.customerId,
                customerName: customer.customerName,
              })
            }
          />
        ))}
        {selected.customerId !== null && !saved && (
          <TemplateRow
            active
            icon={<User01Icon className="size-4" />}
            title={selected.customerName || t("Customer")}
            note={t("Not saved yet")}
            onClick={() => undefined}
          />
        )}
        {adding ? (
          <div className="flex flex-col gap-2 px-1 pt-2">
            <CustomerAutocompleteField
              control={picker.control}
              name="customerId"
              label={t("Customer")}
              placeholder={t("Find a customer")}
              onOptionChange={(option) => {
                if (!option?.id) {
                  return;
                }
                setSelected({ customerId: option.id, customerName: option.label });
                setAdding(false);
                picker.reset();
              }}
            />
            <Button size="xs" variant="ghost" className="self-start" onClick={() => setAdding(false)}>
              {t("Cancel")}
            </Button>
          </div>
        ) : (
          <Button
            size="sm"
            variant="ghost"
            className="mt-2 justify-start"
            onClick={() => setAdding(true)}
          >
            <PlusIcon className="size-4" />
            {t("Add a customer")}
          </Button>
        )}
      </nav>
      <ChecklistTemplateEditor
        key={`${template.customerId || "organization"}:${template.id || "new"}:${template.version}`}
        kind={kind}
        template={template}
        locked={data.locked}
        onRemoved={() => setSelected(ORGANIZATION)}
        onSaved={(next) =>
          setSelected({ customerId: next.customerId || null, customerName: next.customerName })
        }
      />
    </div>
  );
}

function TemplateRow({
  active,
  icon,
  title,
  note,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  title: string;
  note: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-current={active ? "true" : undefined}
      onClick={onClick}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover flex items-center gap-2.5 rounded-md px-2 py-1.5 text-left",
        active && "bg-surface-active",
      )}
    >
      <span className="text-muted-foreground">{icon}</span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-sm font-medium">{title}</span>
        <span className="text-muted-foreground truncate text-xs">{note}</span>
      </span>
    </button>
  );
}
