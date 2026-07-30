import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { safeParse } from "@trenova/shared/lib/parse";

const toastError = vi.hoisted(() => vi.fn());
vi.mock("sonner", () => ({ toast: { error: toastError, success: vi.fn() } }));

const schema = z.object({ id: z.string(), count: z.number() });

describe("safeParse", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("returns the parsed value on success", async () => {
    await expect(safeParse(schema, { id: "row_1", count: 3 })).resolves.toEqual({
      id: "row_1",
      count: 3,
    });
    expect(toastError).not.toHaveBeenCalled();
  });

  it("applies schema coercion and defaults rather than passing the input through", async () => {
    const withDefault = z.object({ id: z.string(), status: z.string().default("Active") });

    await expect(safeParse(withDefault, { id: "row_1" })).resolves.toEqual({
      id: "row_1",
      status: "Active",
    });
  });

  // Every service response crosses this boundary, so a schema that has drifted from the
  // server has to surface as something the user can report — not an unhandled rejection
  // swallowed by a query.
  it("rejects with the ZodError and tells the user when the shape is wrong", async () => {
    await expect(safeParse(schema, { id: "row_1", count: "three" })).rejects.toBeInstanceOf(
      z.ZodError,
    );
    expect(toastError).toHaveBeenCalledTimes(1);
  });

  it("names the payload in the message when a label is given", async () => {
    await expect(safeParse(schema, null, "User")).rejects.toBeInstanceOf(z.ZodError);

    expect(toastError).toHaveBeenCalledWith(
      "Failed to parse User",
      expect.objectContaining({ description: expect.any(String) }),
    );
  });

  it("falls back to a generic label when none is given", async () => {
    await expect(safeParse(schema, undefined)).rejects.toBeInstanceOf(z.ZodError);

    expect(toastError).toHaveBeenCalledWith("Failed to parse response", expect.anything());
  });

  it("treats a missing field, an explicit null and a wrong type as distinct failures", async () => {
    const missing = await safeParse(schema, { id: "row_1" }).catch((e: z.ZodError) => e);
    const nulled = await safeParse(schema, { id: "row_1", count: null }).catch(
      (e: z.ZodError) => e,
    );
    const wrongType = await safeParse(schema, { id: "row_1", count: [] }).catch(
      (e: z.ZodError) => e,
    );

    for (const error of [missing, nulled, wrongType]) {
      expect(error).toBeInstanceOf(z.ZodError);
      expect((error as z.ZodError).issues[0].path).toEqual(["count"]);
    }
  });
});
