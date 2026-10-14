import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { Controller, FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { FormChangesFooter, FormPanelSection } from "@/components/form-changes";
import { FormSaveDock } from "@/components/form-save-dock";
import { useAnyFieldDirty, useChangedFieldNames } from "@/hooks/use-dirty-fields";

// Runs under the React Compiler, as the app is built. react-hook-form changes its
// dirtyFields object in place, so anything memoized on that object's identity goes stale
// and the save dock never opens; under the plain unit project this cannot be seen.

type Settings = {
  name: string;
  address: { city: string; state: string };
  version: number;
};

const SAVED: Settings = { name: "Main", address: { city: "Austin", state: "TX" }, version: 3 };

// Bound through Controller, as the app's fields are: a register() spread in compiled JSX
// is memoized with the form and is not registered again after a reset.
function TextField({ name, label }: { name: "name" | "address.city"; label: string }) {
  return (
    <Controller<Settings>
      name={name}
      render={({ field }) => (
        <input
          aria-label={label}
          name={field.name}
          value={String(field.value)}
          onChange={field.onChange}
          onBlur={field.onBlur}
          ref={field.ref}
        />
      )}
    />
  );
}

function SettingsForm({ onForm }: { onForm?: (form: UseFormReturn<Settings>) => void }) {
  const form = useForm<Settings>({ defaultValues: SAVED });
  onForm?.(form);

  return (
    <FormProvider {...form}>
      <form>
        <TextField name="name" label="Name" />
        <TextField name="address.city" label="City" />
        <FormSaveDock />
      </form>
    </FormProvider>
  );
}

function saveButton() {
  return screen.queryByRole("button", { name: "Save changes" });
}

describe("FormSaveDock under the React Compiler", () => {
  it("stays closed while the form holds what it opened with", () => {
    render(<SettingsForm />);

    expect(saveButton()).toBeNull();
  });

  it("opens with save and reset once a field is changed", async () => {
    render(<SettingsForm />);

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "North" } });

    expect(await screen.findByRole("button", { name: "Save changes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset" })).toBeInTheDocument();
  });

  it("opens for a change to a nested field", async () => {
    render(<SettingsForm />);

    fireEvent.change(screen.getByLabelText("City"), { target: { value: "Dallas" } });

    expect(await screen.findByRole("button", { name: "Save changes" })).toBeInTheDocument();
  });

  it("closes again when a change is typed back to what the form opened with", async () => {
    render(<SettingsForm />);
    const name = screen.getByLabelText("Name");

    fireEvent.change(name, { target: { value: "North" } });
    await screen.findByRole("button", { name: "Save changes" });
    fireEvent.change(name, { target: { value: "Main" } });

    await waitFor(() => expect(saveButton()).toBeNull());
  });

  it("opens for a second form's change after the first is reset", async () => {
    let form: UseFormReturn<Settings> | undefined;
    render(<SettingsForm onForm={(f) => (form = f)} />);

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "North" } });
    await screen.findByRole("button", { name: "Save changes" });
    act(() => form!.reset(SAVED));
    await waitFor(() => expect(saveButton()).toBeNull());
    fireEvent.change(screen.getByLabelText("City"), { target: { value: "Dallas" } });

    expect(await screen.findByRole("button", { name: "Save changes" })).toBeInTheDocument();
  });

  it("stays closed when only the version changes", () => {
    let form: UseFormReturn<Settings> | undefined;
    render(<SettingsForm onForm={(f) => (form = f)} />);

    act(() => form!.setValue("version", 4, { shouldDirty: true }));

    expect(saveButton()).toBeNull();
  });
});

function PanelForm() {
  const form = useForm<Settings>({ defaultValues: SAVED });

  return (
    <FormProvider {...form}>
      <form>
        <FormPanelSection id="where" title="Where" fields={["address.city"]}>
          <TextField name="address.city" label="City" />
        </FormPanelSection>
        <TextField name="name" label="Name" />
        <FormChangesFooter form={form} fields={{}} create={false} />
      </form>
    </FormProvider>
  );
}

describe("form panel change tracking under the React Compiler", () => {
  it("counts each change in the panel footer", async () => {
    render(<PanelForm />);
    expect(screen.getByText("No changes")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "North" } });
    expect(await screen.findByRole("button", { name: "1 unsaved change" })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("City"), { target: { value: "Dallas" } });
    expect(await screen.findByRole("button", { name: "2 unsaved changes" })).toBeInTheDocument();
    expect(screen.queryByText("No changes")).toBeNull();
  });

  it("marks only the section holding a change", async () => {
    render(<PanelForm />);

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "North" } });
    await screen.findByRole("button", { name: "1 unsaved change" });
    expect(screen.queryByText("Unsaved changes")).toBeNull();

    fireEvent.change(screen.getByLabelText("City"), { target: { value: "Dallas" } });
    expect(await screen.findByText("Unsaved changes")).toBeInTheDocument();
  });
});

describe("dirty field hooks under the React Compiler", () => {
  it("useChangedFieldNames lists each changed top-level field, less the version", () => {
    const { result } = renderHook(() => {
      const form = useForm<Settings>({ defaultValues: SAVED });
      return { form, changed: useChangedFieldNames(form.control) };
    });

    expect(result.current.changed).toEqual([]);
    act(() => result.current.form.setValue("name", "North", { shouldDirty: true }));
    expect(result.current.changed).toEqual(["name"]);
    act(() => result.current.form.setValue("address.city", "Dallas", { shouldDirty: true }));
    act(() => result.current.form.setValue("version", 4, { shouldDirty: true }));
    expect(result.current.changed).toEqual(["name", "address"]);
  });

  it("useChangedFieldNames leaves out the field it is told holds the version", () => {
    const { result } = renderHook(() => {
      const form = useForm<Settings>({ defaultValues: SAVED });
      return { form, changed: useChangedFieldNames(form.control, "name") };
    });

    act(() => result.current.form.setValue("name", "North", { shouldDirty: true }));
    act(() => result.current.form.setValue("version", 4, { shouldDirty: true }));

    expect(result.current.changed).toEqual(["version"]);
  });

  it("useAnyFieldDirty follows only the fields it is given, at any depth", () => {
    const { result } = renderHook(() => {
      const form = useForm<Settings>({ defaultValues: SAVED });
      return {
        form,
        address: useAnyFieldDirty(form.control, ["address.city", "address.state"]),
        name: useAnyFieldDirty(form.control, ["name"]),
      };
    });

    act(() => result.current.form.setValue("address.city", "Dallas", { shouldDirty: true }));
    expect(result.current.address).toBe(true);
    expect(result.current.name).toBe(false);

    act(() => result.current.form.setValue("address.city", "Austin", { shouldDirty: true }));
    expect(result.current.address).toBe(false);
  });

  it("useAnyFieldDirty is false for an empty list of fields", () => {
    const { result } = renderHook(() => {
      const form = useForm<Settings>({ defaultValues: SAVED });
      return { form, dirty: useAnyFieldDirty(form.control, []) };
    });

    act(() => result.current.form.setValue("name", "North", { shouldDirty: true }));

    expect(result.current.dirty).toBe(false);
  });
});
