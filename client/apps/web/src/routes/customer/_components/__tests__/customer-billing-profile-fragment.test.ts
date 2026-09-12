import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * The customer edit panel is seeded straight from the GraphQL table row —
 * `FormEditPanel` calls `reset(row)` — so any field the form binds but the
 * fragment does not select arrives undefined and falls back to its default.
 *
 * That is not merely a display bug. The biller sees the default, and saving the
 * form writes it back, silently overwriting whatever was stored. This is exactly
 * how the consolidated-invoicing fields looked like they "never saved": they
 * were saved, the fragment just never asked for them.
 */
const FORM = join(
  import.meta.dirname,
  "..",
  "customer-billing-profile-form.tsx",
);
const FRAGMENT = join(
  import.meta.dirname,
  "..","..","..","..","..","..","..",
  "packages",
  "graphql",
  "src",
  "operations",
  "customer",
  "table.graphql",
);

function selectedFragmentFields(): Set<string> {
  const source = readFileSync(FRAGMENT, "utf8");
  const start = source.indexOf("fragment CustomerBillingProfileFields");
  expect(start).toBeGreaterThanOrEqual(0);

  const block = source.slice(start, source.indexOf("\n}", start));
  return new Set([...block.matchAll(/^ {2}(\w+)/gm)].map((match) => match[1]));
}

function boundFormFields(): string[] {
  const source = readFileSync(FORM, "utf8");
  return [
    ...new Set([...source.matchAll(/"billingProfile\.(\w+)"/g)].map((m) => m[1])),
  ].sort();
}

describe("CustomerBillingProfileFields fragment", () => {
  it("selects every field the billing profile form binds", () => {
    const selected = selectedFragmentFields();
    const missing = boundFormFields().filter((field) => !selected.has(field));

    expect(missing).toEqual([]);
  });

  // Named explicitly so a future edit to the fragment cannot quietly drop them
  // again: these are the fields that replaced the six dead consolidation dials,
  // and every one of them was missing when the form first shipped.
  it("selects the billing schedule fields the statement engine reads", () => {
    const selected = selectedFragmentFields();

    for (const field of [
      "invoiceDelivery",
      "billingCycle",
      "billingCycleAnchorDay",
      "billingCycleTimezone",
      "splitBy",
      "sectionBy",
      "invoiceDetail",
      "maxShipmentsPerInvoice",
      "autoApprove",
    ]) {
      expect(selected, `fragment must select ${field}`).toContain(field);
    }
  });
});
