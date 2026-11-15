import { beforeEach, describe, expect, it, vi } from "vitest";
import { formulaTemplateService } from "../formula-template";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", () => ({ api: { get: mocks.get } }));

describe("formulaTemplateService.getDeclaredInputs", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  // The shipment, lane and re-rate forms read a rating method's inputs on every plan, so they go
  // through the pricing read the Go handler registers outside the template studio's routes
  // (GET /pricing-templates/:templateID/inputs), never through GET /formula-templates/:id.
  it("reads the inputs from the pricing route every plan may call", async () => {
    mocks.get.mockResolvedValue({
      inputs: [
        {
          name: "rate",
          doc: "The rate per mile.",
          type: "Money/mi",
          descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
          required: true,
        },
        {
          name: "minimumCharge",
          type: "Money",
          descriptor: { kind: "money" },
          required: false,
        },
      ],
    });

    const inputs = await formulaTemplateService.getDeclaredInputs("ft_per_mile");

    expect(mocks.get).toHaveBeenCalledTimes(1);
    expect(mocks.get).toHaveBeenCalledWith("/pricing-templates/ft_per_mile/inputs", {
      signal: undefined,
    });
    expect(inputs.map((input) => input.name)).toEqual(["rate", "minimumCharge"]);
    expect(inputs[0]?.required).toBe(true);
    expect(inputs[1]?.doc).toBeNull();
  });

  it("answers no inputs when the server sends none", async () => {
    mocks.get.mockResolvedValue({ inputs: null });

    await expect(formulaTemplateService.getDeclaredInputs("ft_flat")).resolves.toEqual([]);
  });
});
