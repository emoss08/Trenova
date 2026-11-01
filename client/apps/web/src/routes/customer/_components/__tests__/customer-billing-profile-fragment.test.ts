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
const FORM = join(import.meta.dirname, "..", "customer-billing-profile-form.tsx");
const FRAGMENT = join(
  import.meta.dirname,
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
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
  return [...new Set([...source.matchAll(/"billingProfile\.(\w+)"/g)].map((m) => m[1]))].sort();
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
      "emailInvoiceEnabled",
      "ediInvoiceEnabled",
    ]) {
      expect(selected, `fragment must select ${field}`).toContain(field);
    }
  });
});

const CUSTOMER_FORM = join(import.meta.dirname, "..", "customer-form.tsx");

function selectedCustomerRowFields(): Set<string> {
  const source = readFileSync(FRAGMENT, "utf8");
  const start = source.indexOf("fragment CustomerTableRowFields");
  expect(start).toBeGreaterThanOrEqual(0);

  const block = source.slice(start, source.indexOf("\n}", start));
  return new Set([...block.matchAll(/^ {2}(\w+)/gm)].map((match) => match[1]));
}

function boundCustomerFields(): string[] {
  const source = readFileSync(CUSTOMER_FORM, "utf8");
  return [...new Set([...source.matchAll(/name="(\w+)"/g)].map((m) => m[1]))].sort();
}

// The update mutation writes every field the form holds, so a field the row does not
// select would be saved back empty. Broker vetting, DOT and MC numbers and the
// tracking thresholds were cleared on every edit this way.
describe("CustomerTableRowFields fragment", () => {
  it("selects every top-level field the customer form binds", () => {
    const selected = selectedCustomerRowFields();
    const missing = boundCustomerFields().filter((field) => !selected.has(field));

    expect(missing).toEqual([]);
  });
});

const TRACKING_FORM = join(import.meta.dirname, "..", "customer-tracking-settings.tsx");
const TRACKING_FRAGMENT = join(FRAGMENT, "..", "customer-tracking-profile.graphql");

function selectedTrackingFields(): Set<string> {
  const source = readFileSync(TRACKING_FRAGMENT, "utf8");
  const start = source.indexOf("fragment CustomerTrackingProfileFields");
  expect(start).toBeGreaterThanOrEqual(0);

  const block = source.slice(start, source.indexOf("\n}", start));
  return new Set([...block.matchAll(/^ {2}(\w+)/gm)].map((match) => match[1]));
}

function boundTrackingFields(): string[] {
  const source = readFileSync(TRACKING_FORM, "utf8");
  return [...new Set([...source.matchAll(/"trackingProfile\.(\w+)"/g)].map((m) => m[1]))].sort();
}

// The tracking tab is part of the customer form, seeded from the same row, so a
// setting the row does not select would be saved back as its default.
describe("CustomerTrackingProfileFields fragment", () => {
  it("is selected by the customer row", () => {
    expect(selectedCustomerRowFields()).toContain("trackingProfile");
  });

  it("selects every field the tracking tab binds", () => {
    const selected = selectedTrackingFields();
    const bound = boundTrackingFields();

    expect(bound.length).toBeGreaterThan(0);
    expect(bound.filter((field) => !selected.has(field))).toEqual([]);
  });
});
