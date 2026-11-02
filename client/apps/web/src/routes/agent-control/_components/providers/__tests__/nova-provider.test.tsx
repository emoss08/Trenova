import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { NovaProvider } from "../nova-provider";

vi.mock("../../kit/marks", () => ({ Mark: () => null }));

const NOVA_PLATFORMS = [
  "Direct",
  "ClaudePlatformAWS",
  "ClaudePlatformGoogleCloud",
  "VertexAI",
] as const;

function row(
  overrides: Partial<AIProviderRow> & Pick<AIProviderRow, "id" | "name">,
): AIProviderRow {
  return {
    kind: "AnthropicMessages",
    platform: "Direct",
    enabled: true,
    nova: false,
    ...overrides,
  } as AIProviderRow;
}

function renderSection(
  providers: AIProviderRow[],
  overrides: Partial<Parameters<typeof NovaProvider>[0]> = {},
) {
  const onChoose = vi.fn();
  const onAdd = vi.fn();
  render(
    <NovaProvider
      providers={providers}
      model="claude-haiku-5-5"
      novaPlatforms={NOVA_PLATFORMS}
      canChoose
      canAdd
      choosing={false}
      onChoose={onChoose}
      onAdd={onAdd}
      {...overrides}
    />,
  );
  return { onChoose, onAdd };
}

afterEach(cleanup);

describe("NovaProvider", () => {
  it("offers only Anthropic providers on a platform Nova runs through", () => {
    renderSection([
      row({ id: "aiprv_direct", name: "Anthropic" }),
      row({ id: "aiprv_vertex", name: "Vertex", platform: "VertexAI" }),
      row({ id: "aiprv_bedrock", name: "Bedrock", platform: "Bedrock" }),
      row({ id: "aiprv_vllm", name: "Workstation", kind: "OpenAIChat" }),
    ]);

    const group = screen.getByRole("group", { name: "Nova's provider" });
    expect(
      within(group)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Anthropic", "Vertex"]);
  });

  it("names the provider Nova runs on and does not offer it again", () => {
    renderSection([
      row({ id: "aiprv_a", name: "Anthropic", nova: true }),
      row({ id: "aiprv_b", name: "Backup" }),
    ]);

    expect(screen.getByText("Runs on claude-haiku-5-5 through Anthropic")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Anthropic" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByRole("button", { name: "Anthropic" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Backup" })).toBeEnabled();
  });

  it("chooses the provider a person picks", async () => {
    const backup = row({ id: "aiprv_b", name: "Backup" });
    const { onChoose } = renderSection([
      row({ id: "aiprv_a", name: "Anthropic", nova: true }),
      backup,
    ]);

    await userEvent.click(screen.getByRole("button", { name: "Backup" }));

    expect(onChoose).toHaveBeenCalledExactlyOnceWith(backup);
  });

  it("asks for an Anthropic provider when the organization has none", async () => {
    const { onAdd } = renderSection([
      row({ id: "aiprv_vllm", name: "Workstation", kind: "OpenAIChat" }),
    ]);

    expect(screen.getByText("Not set up")).toBeInTheDocument();
    expect(
      screen.getByText("Nova needs an Anthropic provider. Add one and Nova runs on its key."),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Add an Anthropic provider" }));
    expect(onAdd).toHaveBeenCalledOnce();
  });

  it("says Nova is paused while the provider it runs on is off", () => {
    renderSection([row({ id: "aiprv_a", name: "Anthropic", nova: true, enabled: false })]);

    expect(
      screen.getByText("Anthropic runs Nova but is off, so Nova is paused until it is turned on."),
    ).toBeInTheDocument();
  });
});
