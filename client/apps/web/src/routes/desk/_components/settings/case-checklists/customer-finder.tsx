import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { queries } from "@/lib/queries";
import type { ChecklistTemplate } from "@/types/case-checklist";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { useState } from "react";

const SEARCH_DEBOUNCE_MS = 160;

/**
 * Finds a customer to give a checklist of its own, from the same source the
 * customer field searches. A customer that already has one says so, and
 * picking it opens it. Enter picks the first result; Escape closes.
 */
export function CustomerFinder({
  existing,
  onPick,
  onClose,
}: {
  existing: readonly ChecklistTemplate[];
  onPick: (customer: { id: string; name: string }) => void;
  onClose: () => void;
}) {
  const t = useT();
  const [text, setText] = useState("");
  const query = useDebounce(text.trim(), SEARCH_DEBOUNCE_MS);
  const results = useQuery({
    ...queries.caseChecklist.customers(query),
    placeholderData: keepPreviousData,
  });
  const found = results.data?.results ?? [];
  const has = new Set(existing.map((template) => template.customerId));

  return (
    <div className="dk-ck-find">
      <div className="dk-ck-fs">
        <DeskIcon name="search" size={13} />
        <input
          // oxlint-disable-next-line jsx-a11y/no-autofocus -- the finder opened to be typed into
          autoFocus
          value={text}
          placeholder={t("Find a customer")}
          aria-label={t("Find a customer")}
          onChange={(event) => setText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.stopPropagation();
              onClose();
            }
            if (event.key === "Enter" && found[0]) {
              event.preventDefault();
              onPick({ id: found[0].id, name: found[0].label });
            }
          }}
        />
        <button type="button" className="dk-ib" aria-label={t("Cancel")} onClick={onClose}>
          <DeskIcon name="x" size={12} />
        </button>
      </div>
      <div className="dk-ck-fr" role="listbox" aria-label={t("Customers")}>
        {found.map((customer) => (
          <button
            key={customer.id}
            type="button"
            role="option"
            aria-selected={false}
            onClick={() => onPick({ id: customer.id, name: customer.label })}
          >
            <span className="dk-ck-tile dk-sm">{getNameInitials(customer.label, "?")}</span>
            <b>{customer.label}</b>
            {has.has(customer.id) && <span>{t("Has its own")}</span>}
          </button>
        ))}
        {found.length === 0 && (
          <span className="dk-ck-none">
            {results.isFetching
              ? t("Searching…")
              : query
                ? t("No customer matches “{0}”.", query)
                : t("No customers yet.")}
          </span>
        )}
      </div>
    </div>
  );
}
