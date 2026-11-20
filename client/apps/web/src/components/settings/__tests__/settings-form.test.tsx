import { InputField } from "@/components/fields/input-field";
import { TestRouter, type UrlUpdateEvent } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { useFormContext } from "react-hook-form";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { SettingsForm, type SettingsTabNav } from "../settings-form";
import { SettingsSection } from "../settings-section";

const served = vi.hoisted(() => ({ refusal: null as unknown }));

// The real mutation calls `save` with what it is given and hands a refusal to the form; this
// does the same without a server.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({
    mutationFn,
    form,
  }: {
    mutationFn: (values: unknown) => Promise<unknown>;
    form: unknown;
  }) => ({
    mutateAsync: async (values: unknown) => {
      if (served.refusal) {
        const { handleMutationError } = await vi.importActual<
          typeof import("@/hooks/use-api-mutation")
        >("@/hooks/use-api-mutation");
        handleMutationError({ error: served.refusal, form: form as never });
        throw served.refusal;
      }
      return mutationFn(values);
    },
  }),
}));

const schema = z.object({ name: z.string(), code: z.string() });
type Values = z.infer<typeof schema>;
type Tab = "general" | "codes";

const TABS = [
  { value: "general", label: "General", fields: ["name"] },
  { value: "codes", label: "Codes", fields: ["code"] },
] as const satisfies readonly { value: Tab; label: string; fields: readonly (keyof Values)[] }[];

function GeneralTab({ nav }: { nav: SettingsTabNav<Tab> }) {
  const { control } = useFormContext<Values>();
  return (
    <SettingsSection title="General">
      <InputField control={control} name="name" label="Name" />
      <p>Showing {nav.openTab}</p>
      <button type="button" onClick={() => nav.selectTab("codes")}>
        Go to codes
      </button>
    </SettingsSection>
  );
}

function CodesTab() {
  const { control } = useFormContext<Values>();
  return (
    <SettingsSection title="Codes">
      <InputField control={control} name="code" label="Code" />
    </SettingsSection>
  );
}

function renderForm({
  search = "",
  save = vi.fn(async (values: Values) => values),
  prepare,
  onUrlUpdate,
}: {
  search?: string;
  save?: (values: Values) => Promise<unknown>;
  prepare?: (values: Values) => Values;
  onUrlUpdate?: (event: UrlUpdateEvent) => void;
} = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <TestRouter search={search} onUrlUpdate={onUrlUpdate}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <SettingsForm<Values, Tab>
            values={{ name: "Acme", code: "ac" }}
            schema={schema}
            queryKey={["settings-form-test"]}
            save={save}
            resourceName="Test Settings"
            prepare={prepare}
            tabs={TABS}
            content={(tab, nav) => (tab === "general" ? <GeneralTab nav={nav} /> : <CodesTab />)}
          />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
  return { save };
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});
afterEach(() => {
  served.refusal = null;
  cleanup();
});

describe("SettingsForm", () => {
  it("saves the record as prepare sends it, not as the form holds it", async () => {
    const user = userEvent.setup();
    const save = vi.fn(async (values: Values) => values);
    renderForm({ save, prepare: (values) => ({ ...values, code: values.code.toUpperCase() }) });

    const name = screen.getByRole("textbox", { name: "Name" });
    await user.clear(name);
    await user.type(name, "Acme Freight");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    expect(save).toHaveBeenCalledWith({ name: "Acme Freight", code: "AC" });
  });

  it("saves the form's values as they stand when the page has no prepare", async () => {
    const user = userEvent.setup();
    const save = vi.fn(async (values: Values) => values);
    renderForm({ save });

    await user.type(screen.getByRole("textbox", { name: "Name" }), " Co");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledWith({ name: "Acme Co", code: "ac" }));
  });

  it("opens and marks the tab holding a field the server refused", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Code is already in use",
      instance: "/api/v1/test-settings/",
      code: "INVALID",
      errors: [
        { path: ["code"], message: "Code is already in use", code: "DUPLICATE", location: "body" },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByRole("textbox", { name: "Name" }), " Co");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const codes = await screen.findByRole("tab", { name: /Codes/, selected: true });
    expect(within(codes).getByText("Has fields to fix")).toBeInTheDocument();
    expect(within(screen.getByRole("tabpanel")).getByText("Code is already in use")).toBeVisible();
  });

  it("opens the tab the address names", () => {
    renderForm({ search: "?tab=codes" });

    expect(screen.getByRole("tab", { name: "Codes" })).toHaveAttribute("aria-selected", "true");
  });

  // The first tab keeps a clean address; any other is written to ?tab= so it can be linked.
  it("keeps the open tab in the address", async () => {
    const user = userEvent.setup();
    const updates: string[] = [];
    renderForm({ onUrlUpdate: (event) => updates.push(event.queryString) });

    await user.click(screen.getByRole("tab", { name: "Codes" }));
    await waitFor(() => expect(updates.at(-1)).toBe("?tab=codes"));

    await user.click(screen.getByRole("tab", { name: "General" }));
    await waitFor(() => expect(updates.at(-1)).toBe(""));
  });

  it("tells a tab's content which tab is open and lets it open another", async () => {
    const user = userEvent.setup();
    renderForm();

    expect(screen.getByText("Showing general")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Go to codes" }));

    expect(await screen.findByRole("tab", { name: "Codes", selected: true })).toBeInTheDocument();
    expect(screen.getByText("Showing codes")).toBeInTheDocument();
  });
});
