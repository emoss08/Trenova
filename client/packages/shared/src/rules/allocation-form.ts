// allocation-form.ts validates the allocation rows of a form with the rules module, for the zod
// schemas the split editors' forms resolve with.
//
// A refinement needs the loaded module, so it is built from a RulesRuntime: a form cannot
// validate allocations before the module has loaded, and the shared data schemas (which also
// parse server responses, where the server has already validated every split) carry none.
// Every charge of the form is measured in one `allocation.remainder` call, and each charge's
// first problem lands on that charge's allocation list, where the split editor shows it.
//
// The refinement sits on the whole form, because a charge's total is read from fields beside
// its rows (the freight amount, an accessorial's method and units). Zod skips a refinement
// once another field has failed in a way that aborts, such as a required select left empty, so
// these run regardless (`when`): a split that does not add up is reported while the rest of
// the form is still being filled in, as the split dialogs need it to be.

import { translateMessage } from "@trenova/shared/i18n/message";
import type { z } from "zod";
import {
  type AllocationChargeInput,
  type AllocationRowInput,
  allocationRemainders,
} from "./allocation";
import type { RulesRuntime } from "./runtime";
import type { DecimalInput } from "./wire";

/** One charge of a form and where its allocation list sits in the form. */
export type AllocationFormCharge = {
  readonly path: readonly (string | number)[];
  readonly charge: AllocationChargeInput;
};

type AllocationRows = readonly (AllocationRowInput | null | undefined)[] | null | undefined;

type AllocatedCharge = {
  readonly method?: string | null;
  readonly amount?: DecimalInput;
  readonly unit?: number | null;
  readonly allocations?: AllocationRows;
};

type AllocatedShipment = {
  readonly freightChargeAmount?: DecimalInput;
  readonly freightAllocations?: AllocationRows;
  readonly additionalCharges?: readonly (AllocatedCharge | null | undefined)[] | null;
};

type SingleCharge = { readonly allocations?: AllocationRows };

/** Runs a refinement even when other fields failed, as long as there is an object to read. */
const EVEN_WITH_OTHER_ISSUES = {
  when: (payload: { value: unknown }) =>
    typeof payload.value === "object" && payload.value !== null,
};

/**
 * An additional charge as the split editor judges it. A percentage accessorial is priced
 * against the freight on the server, so its total is not known in the form and only a percent
 * split can be checked against it: it goes without a total or a method.
 */
export function additionalChargeForAllocation(charge: AllocatedCharge): AllocationChargeInput {
  if (charge.method === "Percentage") {
    return { kind: "Accessorial", allocations: charge.allocations };
  }
  return {
    kind: "Accessorial",
    method: charge.method,
    amount: charge.amount,
    unit: charge.unit,
    allocations: charge.allocations,
  };
}

/** Every allocated charge of a shipment form: the freight, then each additional charge. */
export function shipmentAllocationCharges(shipment: AllocatedShipment): AllocationFormCharge[] {
  const charges: AllocationFormCharge[] = [
    {
      path: ["freightAllocations"],
      charge: {
        kind: "Freight",
        total: shipment.freightChargeAmount ?? 0,
        allocations: shipment.freightAllocations,
      },
    },
  ];
  const additional = Array.isArray(shipment.additionalCharges) ? shipment.additionalCharges : [];
  additional.forEach((charge, index) => {
    if (!charge || typeof charge !== "object") return;
    charges.push({
      path: ["additionalCharges", index, "allocations"],
      charge: additionalChargeForAllocation(charge),
    });
  });
  return charges;
}

/** Adds an issue for every charge whose rows cannot be saved, from one module call. */
export function addAllocationIssues(
  rules: RulesRuntime,
  ctx: z.RefinementCtx,
  charges: readonly AllocationFormCharge[],
): void {
  const states = allocationRemainders(rules, { charges: charges.map((entry) => entry.charge) });
  states.forEach((state, index) => {
    if (state.problem === null) return;
    ctx.addIssue({
      code: "custom",
      message: translateMessage(state.problem),
      path: [...charges[index].path],
    });
  });
}

/** A shipment form's schema that also checks the freight's and every charge's split. */
export function withShipmentAllocations<S extends z.ZodType>(schema: S, rules: RulesRuntime): S {
  return schema.superRefine((shipment, ctx) => {
    addAllocationIssues(rules, ctx, shipmentAllocationCharges(shipment as AllocatedShipment));
  }, EVEN_WITH_OTHER_ISSUES);
}

/**
 * A schema for a form that splits one charge, which also checks that its `allocations` add up
 * to the total `totalOf` reads from the values (null when the total is not known here, when
 * only percent rows can be judged).
 */
export function withChargeAllocations<S extends z.ZodType>(
  schema: S,
  rules: RulesRuntime,
  totalOf: (values: z.output<S>) => DecimalInput,
): S {
  return schema.superRefine((values, ctx) => {
    const total = totalOf(values);
    const allocations = (values as SingleCharge).allocations;
    const charge: AllocationChargeInput =
      total === null || total === undefined ? { allocations } : { total, allocations };
    addAllocationIssues(rules, ctx, [{ path: ["allocations"], charge }]);
  }, EVEN_WITH_OTHER_ISSUES);
}
