import { act, renderHook, waitFor } from "@testing-library/react";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { describe, expect, it, vi } from "vitest";
import { useEditFlow } from "../use-edit-flow";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

type Values = { name: string; tools: string[]; version: number };

const loaded: Values = { name: "Billing desk", tools: ["get_invoice"], version: 3 };

function setup(
  onSave: (values: Values) => Promise<Values>,
  options: { create?: boolean; nameRequired?: boolean } = {},
) {
  const onClose = vi.fn();
  const loadLatest = vi.fn(async () => ({ ...loaded, name: "Their name", version: 4 }));
  const hook = renderHook(() => {
    const form = useForm<Values>({ defaultValues: loaded });
    form.register("name", options.nameRequired ? { required: "Name is required" } : undefined);
    form.register("tools");
    form.register("version");
    const flow = useEditFlow({ form, onSave, onClose, loadLatest, create: options.create });
    return { form, flow };
  });
  return { ...hook, onClose, loadLatest };
}

function conflictError() {
  return new ApiRequestError(409, {
    type: "resource-conflict",
    title: "Conflict",
    status: 409,
    conflict: {
      version: 4,
      updatedByName: "Sarah Alvarez",
      updatedAt: 1_800_000_000,
      changes: [{ field: "name", label: "Name" }],
    },
  });
}

describe("useEditFlow", () => {
  it("names what changed, leaving the version out", async () => {
    const { result } = setup(async (values) => values);

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));

    await waitFor(() => expect(result.current.flow.changed).toEqual(["name"]));
    expect(result.current.flow.dirty).toBe(true);
    expect(result.current.flow.canSave).toBe(true);
  });

  it("asks before closing with unsaved changes, and closes at once without", async () => {
    const { result, onClose } = setup(async (values) => values);

    act(() => result.current.flow.tryClose());
    expect(onClose).toHaveBeenCalledTimes(1);

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.dirty).toBe(true));
    act(() => result.current.flow.tryClose());

    expect(result.current.flow.confirmingClose).toBe(true);
    expect(onClose).toHaveBeenCalledTimes(1);
    act(() => result.current.flow.back());
    expect(result.current.flow.confirmingClose).toBe(false);
  });

  it("resets to what the save returned, so the next save carries the new version", async () => {
    const onSave = vi.fn(async (values: Values) => ({ ...values, version: values.version + 1 }));
    const { result } = setup(onSave);

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.canSave).toBe(true));
    act(() => result.current.flow.save());

    await waitFor(() => expect(result.current.flow.saved).toBe(true));
    expect(result.current.form.getValues("version")).toBe(4);
    expect(result.current.flow.dirty).toBe(false);
  });

  it("holds a lost race as a conflict, and keeping mine saves over theirs", async () => {
    const onSave = vi
      .fn<(values: Values) => Promise<Values>>()
      .mockRejectedValueOnce(conflictError())
      .mockImplementation(async (values) => ({ ...values, version: values.version + 1 }));
    const { result } = setup(onSave);

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.canSave).toBe(true));
    act(() => result.current.flow.save());

    await waitFor(() => expect(result.current.flow.conflict?.updatedByName).toBe("Sarah Alvarez"));
    act(() => result.current.flow.keepMine());

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(2));
    expect(onSave.mock.calls[1]?.[0]).toMatchObject({ name: "Payroll desk", version: 4 });
    await waitFor(() => expect(result.current.flow.conflict).toBeNull());
  });

  it("loads theirs over the draft", async () => {
    const { result, loadLatest } = setup(async () => {
      throw conflictError();
    });

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.canSave).toBe(true));
    act(() => result.current.flow.save());
    await waitFor(() => expect(result.current.flow.conflict).not.toBeNull());

    await act(() => result.current.flow.loadTheirs());

    expect(loadLatest).toHaveBeenCalled();
    expect(result.current.form.getValues()).toMatchObject({ name: "Their name", version: 4 });
    expect(result.current.flow.conflict).toBeNull();
    expect(result.current.flow.dirty).toBe(false);
  });

  it("undoes one change", async () => {
    const { result } = setup(async (values) => values);

    act(() => {
      result.current.form.setValue("name", "Payroll desk", { shouldDirty: true });
      result.current.form.setValue("tools", ["get_invoice", "hold_invoice"], { shouldDirty: true });
    });
    await waitFor(() => expect(result.current.flow.changed).toHaveLength(2));
    act(() => result.current.flow.undo("name"));

    await waitFor(() => expect(result.current.flow.changed).toEqual(["tools"]));
    expect(result.current.form.getValues("name")).toBe("Billing desk");
  });

  it("closes a new record once it is created", async () => {
    const { result, onClose } = setup(async (values) => ({ ...values, version: 1 }), {
      create: true,
    });

    expect(result.current.flow.canSave).toBe(true);
    act(() => result.current.flow.save());

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });
});

/**
 * An editor draws only some of its fields, so a save stopped by validation must say why
 * on its own: neither a rule the form checks before sending nor a field the server
 * refuses may fail without a word.
 */
describe("useEditFlow when a save is refused", () => {
  it("says why when the form's own rules stop the save, and sends nothing", async () => {
    vi.mocked(toast.error).mockClear();
    const onSave = vi.fn(async (values: Values) => values);
    const { result } = setup(onSave, { nameRequired: true });

    act(() => result.current.form.setValue("name", "", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.canSave).toBe(true));
    act(() => result.current.flow.save());

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        "This change could not be saved",
        expect.objectContaining({ description: expect.stringContaining("Name is required") }),
      ),
    );
    expect(onSave).not.toHaveBeenCalled();
  });

  it("says why when the server refuses a field the editor never shows", async () => {
    vi.mocked(toast.error).mockClear();
    const onSave = vi.fn(async () => {
      throw new ApiRequestError(422, {
        type: "validation-error",
        title: "Validation Error",
        status: 422,
        errors: [
          { field: "tools", message: "This tool needs a trusted provider", code: "INVALID" },
        ],
      });
    });
    const { result } = setup(onSave);

    act(() => result.current.form.setValue("name", "Payroll desk", { shouldDirty: true }));
    await waitFor(() => expect(result.current.flow.canSave).toBe(true));
    act(() => result.current.flow.save());

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        "This change could not be saved",
        expect.objectContaining({
          description: expect.stringContaining("This tool needs a trusted provider"),
        }),
      ),
    );
    expect(result.current.flow.saving).toBe(false);
  });
});
