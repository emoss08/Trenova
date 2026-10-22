import { zodResolver } from "@hookform/resolvers/zod";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { withChargeAllocations } from "@trenova/shared/rules/allocation-form";
import { chargeAllocationSchema } from "@trenova/shared/types/shipment";
import type { Resolver } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

// The resolver must not validate until the module has loaded: a form's first validation can
// run before the idle preload has finished. A held load lets the test see that it waits.
const load = vi.hoisted(() => {
  let release: (() => void) | null = null;
  return {
    hold: () =>
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    release: () => release?.(),
    gate: null as Promise<void> | null,
  };
});

vi.mock("@trenova/shared/rules/load", async (importOriginal) => {
  const original = await importOriginal<typeof import("@trenova/shared/rules/load")>();
  return {
    ...original,
    loadRules: async () => {
      if (load.gate) await load.gate;
      return original.loadRules();
    },
  };
});

const { rulesResolver } = await import("@/lib/rules-resolver");

type Values = { amount: number; allocations: z.input<typeof chargeAllocationSchema>[] };

function chargeResolver(builds: RulesRuntime[]) {
  return rulesResolver<Values>((rules) => {
    builds.push(rules);
    return zodResolver(
      withChargeAllocations(
        z.object({ amount: z.number(), allocations: z.array(chargeAllocationSchema) }),
        rules,
        (values) => values.amount,
      ),
    ) as unknown as Resolver<Values>;
  });
}

const OVER = {
  amount: 100,
  allocations: [
    { billToCustomerId: "cus_a", method: "Amount" as const, amount: 60, sequence: 0 },
    { billToCustomerId: "cus_b", method: "Amount" as const, amount: 60, sequence: 1 },
  ],
};

const OPTIONS = { fields: {}, shouldUseNativeValidation: false } as Parameters<Resolver<Values>>[2];

afterEach(() => {
  load.gate = null;
});

describe("rulesResolver", () => {
  it("waits for the rules module before validating, then reports its message", async () => {
    load.gate = load.hold();
    const resolve = chargeResolver([]);
    let settled = false;
    const result = Promise.resolve(resolve(OVER, undefined, OPTIONS)).then((value) => {
      settled = true;
      return value;
    });

    await new Promise((resolveTick) => setTimeout(resolveTick, 10));
    expect(settled).toBe(false);

    load.release();
    const { errors } = await result;
    expect(errors.allocations?.message).toBe("Amounts must add up to the charge");
  });

  it("builds the schema once per loaded module", async () => {
    const builds: RulesRuntime[] = [];
    const resolve = chargeResolver(builds);
    await resolve(OVER, undefined, OPTIONS);
    const { errors, values } = await resolve(
      { ...OVER, allocations: [{ ...OVER.allocations[0], amount: 100 }] },
      undefined,
      OPTIONS,
    );
    expect(builds).toHaveLength(1);
    expect(errors).toEqual({});
    expect(values).toMatchObject({ amount: 100 });
  });
});
