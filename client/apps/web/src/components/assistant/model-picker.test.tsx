import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ModelPicker } from "./model-picker";

const twoProviders = [
  {
    id: "aiprv_01M2ZX9KNWNT4TN19C45ZT63WG",
    name: "OpenRouter",
    kind: "OpenAIChat",
    model: "nvidia/nemotron-3-super-120b-a12b",
    trusted: true,
  },
  {
    id: "aiprv_01M301NZM073J2744DF4VXNZ6Z",
    name: "Minimax",
    kind: "OpenAIChat",
    model: "minimax/minimax-m3",
    trusted: true,
  },
] as const;

describe("ModelPicker", () => {
  it("renders the trigger when the organization has providers", () => {
    render(<ModelPicker options={twoProviders} value="" onChange={() => {}} />);
    expect(screen.getByLabelText("Choose which model answers")).toBeInTheDocument();
  });

  it("names the picked model on the trigger", () => {
    render(<ModelPicker options={twoProviders} value={twoProviders[1].id} onChange={() => {}} />);
    expect(screen.getByText("minimax/minimax-m3")).toBeInTheDocument();
  });

  it("renders nothing only when there is no provider at all", () => {
    const { container } = render(<ModelPicker options={[]} value="" onChange={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });
});
