import { cn } from "@/lib/utils";
import { useMemo, useState } from "react";
import { ChevronDownIcon } from "lucide-react";
import { FieldRow } from "./field-row";
import type { ReconciliationField } from "./types";

type FieldReconciliationListProps = {
  fields: Record<string, ReconciliationField>;
  showIssuesOnly: boolean;
  onAccept: (key: string) => void;
  onEdit: (key: string, value: unknown) => void;
  onReset: (key: string) => void;
};

type FieldGroup = {
  label: string;
  keys: string[];
};

const FIELD_GROUPS: FieldGroup[] = [
  {
    label: "Reference Numbers",
    keys: ["bol", "proNumber", "loadNumber", "referenceNumber", "poNumber", "appointmentNumber"],
  },
  {
    label: "Rates & Charges",
    keys: ["rate", "fuelSurcharge", "paymentTerms"],
  },
  {
    label: "Shipment Details",
    keys: ["equipmentType", "commodity", "weight", "pieces", "serviceType"],
  },
  {
    label: "Parties",
    keys: ["shipper", "consignee", "carrierName", "scac", "billTo", "carrierContact"],
  },
];

const ALL_GROUPED_KEYS = new Set(FIELD_GROUPS.flatMap((g) => g.keys));

function SectionHeader({
  label,
  issueCount,
  collapsed,
  onToggle,
}: {
  label: string;
  issueCount: number;
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      className="flex w-full items-center gap-2 px-2 py-1.5 text-2xs font-medium tracking-wider text-muted-foreground/60 uppercase transition-colors hover:text-muted-foreground"
      onClick={onToggle}
    >
      <ChevronDownIcon
        className={cn("size-3 transition-transform", collapsed && "-rotate-90")}
      />
      <span>{label}</span>
      {issueCount > 0 && (
        <span className="flex size-4 items-center justify-center rounded-full bg-amber-500/15 text-2xs font-medium text-amber-500">
          {issueCount}
        </span>
      )}
    </button>
  );
}

export function FieldReconciliationList({
  fields,
  showIssuesOnly,
  onAccept,
  onEdit,
  onReset,
}: FieldReconciliationListProps) {
  const [collapsedSections, setCollapsedSections] = useState<Set<string>>(new Set());

  const toggleSection = (label: string) => {
    setCollapsedSections((prev) => {
      const next = new Set(prev);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
  };

  const groupedSections = useMemo(() => {
    return FIELD_GROUPS.map((group) => {
      const groupFields = group.keys
        .map((key) => fields[key])
        .filter((f): f is ReconciliationField => !!f);

      const filtered = showIssuesOnly
        ? groupFields.filter(
            (f) => f.status === "needs-review" || f.status === "missing" || f.status === "conflicting",
          )
        : groupFields;

      const issueCount = groupFields.filter(
        (f) => f.status === "needs-review" || f.status === "missing" || f.status === "conflicting",
      ).length;

      return { ...group, fields: filtered, issueCount };
    }).filter((g) => g.fields.length > 0);
  }, [fields, showIssuesOnly]);

  // Ungrouped fields (any fields not in the predefined groups)
  const ungroupedFields = useMemo(() => {
    const extras = Object.values(fields).filter((f) => !ALL_GROUPED_KEYS.has(f.key));
    if (showIssuesOnly) {
      return extras.filter(
        (f) => f.status === "needs-review" || f.status === "missing" || f.status === "conflicting",
      );
    }
    return extras;
  }, [fields, showIssuesOnly]);

  if (groupedSections.length === 0 && ungroupedFields.length === 0) {
    return (
      <div className="px-4 py-6 text-center text-xs text-muted-foreground/50">
        {showIssuesOnly ? "All fields accepted. No issues to review." : "No fields extracted."}
      </div>
    );
  }

  return (
    <div className="px-3 py-2">
      {groupedSections.map((section) => {
        const collapsed = collapsedSections.has(section.label);
        return (
          <div key={section.label} className="mb-1">
            <SectionHeader
              label={section.label}
              issueCount={section.issueCount}
              collapsed={collapsed}
              onToggle={() => toggleSection(section.label)}
            />
            {!collapsed && (
              <div className="ml-1">
                {section.fields.map((field) => (
                  <FieldRow
                    key={field.key}
                    field={field}
                    onAccept={onAccept}
                    onEdit={onEdit}
                    onReset={onReset}
                    onSelectAlternative={onEdit}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}

      {ungroupedFields.length > 0 && (
        <div className="mb-1">
          <SectionHeader
            label="Other"
            issueCount={ungroupedFields.filter(
              (f) => f.status === "needs-review" || f.status === "missing" || f.status === "conflicting",
            ).length}
            collapsed={collapsedSections.has("Other")}
            onToggle={() => toggleSection("Other")}
          />
          {!collapsedSections.has("Other") && (
            <div className="ml-1">
              {ungroupedFields.map((field) => (
                <FieldRow
                  key={field.key}
                  field={field}
                  onAccept={onAccept}
                  onEdit={onEdit}
                  onReset={onReset}
                  onSelectAlternative={onEdit}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
