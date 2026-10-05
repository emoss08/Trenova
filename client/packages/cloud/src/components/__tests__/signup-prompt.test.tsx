import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SignupPrompt } from "../signup-prompt";

const publicConfig = vi.hoisted(() => ({ signupAvailable: false }));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: {},
    isLoading: false,
    isCloud: publicConfig.signupAvailable,
    signupAvailable: publicConfig.signupAvailable,
  }),
}));

function renderPrompt() {
  return render(
    <MemoryRouter>
      <SignupPrompt fallback="Sign in with the account your organization set up for you." />
    </MemoryRouter>,
  );
}

describe("SignupPrompt", () => {
  beforeEach(() => {
    publicConfig.signupAvailable = false;
  });

  it("links to signup when Trenova Cloud has signup open", () => {
    publicConfig.signupAvailable = true;
    renderPrompt();

    expect(screen.getByRole("link", { name: "Create an account" })).toHaveAttribute(
      "href",
      "/signup",
    );
    expect(
      screen.queryByText("Sign in with the account your organization set up for you."),
    ).not.toBeInTheDocument();
  });

  it("keeps the host's line where accounts are made by an administrator", () => {
    renderPrompt();

    expect(screen.queryByRole("link", { name: "Create an account" })).not.toBeInTheDocument();
    expect(
      screen.getByText("Sign in with the account your organization set up for you."),
    ).toBeInTheDocument();
  });
});
