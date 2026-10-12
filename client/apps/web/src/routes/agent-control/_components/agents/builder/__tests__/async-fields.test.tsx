import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { aicFieldTrigger, aicMultiFieldTrigger } from "../../../edit/field-trigger";
import { agentFormDefaults, type AgentFormValues } from "../../agent-form-schema";
import { LimitsBlock } from "../limits-block";
import { TriggerBlock } from "../trigger-block";

type ProviderProps = {
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  triggerClassName?: string;
};
type BoundProps = { name: string; triggerClassName?: string; placeholder?: string };

const seen = vi.hoisted(() => ({
  provider: null as ProviderProps | null,
  roles: null as BoundProps | null,
  timezone: null as BoundProps | null,
}));

vi.mock("@/components/autocomplete-fields", () => ({
  AIProviderAutocomplete: (props: ProviderProps) => {
    seen.provider = props;
    return (
      <div>
        <span data-testid="provider">{props.value || props.placeholder}</span>
        <button type="button" onClick={() => props.onValueChange("aiprv_2")}>
          pick
        </button>
        <button type="button" onClick={() => props.onValueChange("")}>
          clear
        </button>
      </div>
    );
  },
  RoleAutocompleteField: (props: BoundProps) => {
    seen.roles = props;
    return <div data-testid="roles" />;
  },
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: (props: BoundProps) => {
    seen.timezone = props;
    return <div data-testid="timezone" />;
  },
}));

function renderInForm(children: ReactNode, values: Partial<AgentFormValues>) {
  let form!: UseFormReturn<AgentFormValues>;
  function Host() {
    form = useForm<AgentFormValues>({ defaultValues: { ...agentFormDefaults, ...values } });
    return <FormProvider {...form}>{children}</FormProvider>;
  }
  render(<Host />);
  return () => form;
}

describe("LimitsBlock", () => {
  it("picks the preferred provider from the provider autocomplete, empty meaning automatic", async () => {
    const form = renderInForm(<LimitsBlock fresh={false} budget={undefined} />, {
      preferredProviderId: "aiprv_1",
    });

    expect(screen.getByTestId("provider").textContent).toBe("aiprv_1");
    expect(seen.provider?.placeholder).toBe("Automatic");
    expect(seen.provider?.triggerClassName).toBe(aicFieldTrigger);

    await userEvent.click(screen.getByText("pick"));
    expect(form().getValues("preferredProviderId")).toBe("aiprv_2");
    expect(form().getFieldState("preferredProviderId").isDirty).toBe(true);

    await userEvent.click(screen.getByText("clear"));
    expect(form().getValues("preferredProviderId")).toBe("");
  });
});

describe("TriggerBlock", () => {
  const base = {
    fresh: false,
    events: [],
    sensitiveTools: [],
    toolTitle: (name: string) => name,
  };

  it("binds the role picker to the agent's granted roles once access is limited", () => {
    renderInForm(<TriggerBlock {...base} />, { accessMode: "Roles", accessRoleIds: ["rol_1"] });

    expect(screen.getByTestId("roles")).toBeTruthy();
    expect(seen.roles?.name).toBe("accessRoleIds");
    expect(seen.roles?.triggerClassName).toBe(aicMultiFieldTrigger);
  });

  it("asks for no roles while everyone can use it", () => {
    seen.roles = null;
    renderInForm(<TriggerBlock {...base} />, { accessMode: "Everyone" });

    expect(screen.queryByTestId("roles")).toBeNull();
    expect(seen.roles).toBeNull();
  });

  it("binds the time zone select to the schedule's time zone", () => {
    renderInForm(<TriggerBlock {...base} />, {
      triggerMode: "Scheduled",
      cronExpression: "0 6 * * 1-5",
      cronTimezone: "America/Chicago",
    });

    expect(screen.getByTestId("timezone")).toBeTruthy();
    expect(seen.timezone?.name).toBe("cronTimezone");
    expect(seen.timezone?.triggerClassName).toBe(aicFieldTrigger);
  });
});
