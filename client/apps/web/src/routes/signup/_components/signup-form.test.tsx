import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useImperativeHandle, type Ref } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SignupForm } from "./signup-form";

const mocks = vi.hoisted(() => ({
  signup: vi.fn(),
  reset: vi.fn(),
  onSubmitted: vi.fn(),
}));

vi.mock("@/services/cloud-signup", () => ({
  cloudSignupService: { signup: mocks.signup },
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: {
      platformMode: "cloud",
      signupEnabled: true,
      turnstileSiteKey: "site-key",
      termsUrl: "https://example.com/terms",
      privacyUrl: "https://example.com/privacy",
      freePlan: { limits: {} },
    },
    isLoading: false,
    isCloud: true,
    signupAvailable: true,
  }),
}));

// Turnstile itself is Cloudflare's iframe; the form's contract with it is a token that
// arrives through onTokenChange and a reset() it must call once a token is spent.
vi.mock("@/components/turnstile-widget", () => ({
  TurnstileWidget: ({
    onTokenChange,
    ref,
  }: {
    onTokenChange: (token: string | null) => void;
    ref?: Ref<{ reset: () => void }>;
  }) => {
    useImperativeHandle(ref, () => ({
      reset: () => {
        mocks.reset();
        onTokenChange(null);
      },
    }));
    return (
      <button type="button" onClick={() => onTokenChange("turnstile-token")}>
        Pass security check
      </button>
    );
  },
}));

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <SignupForm turnstileSiteKey="site-key" onSubmitted={mocks.onSubmitted} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/your name/i), "Jordan Rivera");
  await user.type(screen.getByLabelText(/work email/i), "jordan@riverafreight.com");
  await user.type(screen.getByLabelText(/company name/i), "Rivera Freight LLC");
  await user.type(screen.getByLabelText(/^password/i), "correct horse battery");
  await user.click(screen.getByRole("checkbox"));
}

describe("SignupForm", { timeout: 20_000 }, () => {
  beforeEach(() => {
    mocks.signup.mockReset();
    mocks.reset.mockReset();
    mocks.onSubmitted.mockReset();
  });

  it("shows field errors and sends nothing when the form is empty", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("Enter your name")).toBeInTheDocument();
    expect(screen.getByText("Enter your work email")).toBeInTheDocument();
    expect(screen.getByText("Password must be at least 12 characters")).toBeInTheDocument();
    expect(
      screen.getByText("Accept the terms of service and privacy policy to continue"),
    ).toBeInTheDocument();
    expect(mocks.signup).not.toHaveBeenCalled();
  });

  it("asks for the security check before submitting", async () => {
    const user = userEvent.setup();
    renderForm();
    await fillValidForm(user);

    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(
      await screen.findByText("Complete the security check before creating your account."),
    ).toBeInTheDocument();
    expect(mocks.signup).not.toHaveBeenCalled();
  });

  it("submits the documented payload, spends the token and reports the submission", async () => {
    mocks.signup.mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderForm();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Pass security check" }));

    await user.click(screen.getByRole("button", { name: "Create account" }));

    await waitFor(() => expect(mocks.onSubmitted).toHaveBeenCalledTimes(1));
    expect(mocks.signup).toHaveBeenCalledWith({
      name: "Jordan Rivera",
      emailAddress: "jordan@riverafreight.com",
      companyName: "Rivera Freight LLC",
      password: "correct horse battery",
      acceptTerms: true,
      website: "",
      turnstileToken: "turnstile-token",
    });
    expect(mocks.reset).toHaveBeenCalledTimes(1);
    expect(mocks.onSubmitted).toHaveBeenCalledWith({
      emailAddress: "jordan@riverafreight.com",
      companyName: "Rivera Freight LLC",
    });
  });

  it("maps the server's field errors onto the form and resets the check", async () => {
    mocks.signup.mockRejectedValue(
      new ApiRequestError(422, {
        type: "https://trenova.app/problems/validation-error",
        title: "Validation Failed",
        status: 422,
        errors: [
          {
            field: "emailAddress",
            message: "Disposable email addresses cannot be used",
            code: "INVALID",
          },
          {
            field: "turnstileToken",
            message: "The security check failed. Try again.",
            code: "INVALID",
          },
        ],
      }),
    );
    const user = userEvent.setup();
    renderForm();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Pass security check" }));

    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(
      await screen.findByText("Disposable email addresses cannot be used"),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/work email/i)).toHaveAttribute("aria-invalid", "true");
    // turnstileToken has no input, so it lands on the form root rather than vanishing.
    expect(screen.getAllByText("The security check failed. Try again.").length).toBeGreaterThan(0);
    expect(mocks.reset).toHaveBeenCalledTimes(1);
    expect(mocks.onSubmitted).not.toHaveBeenCalled();
  });

  it("explains a rate limit with the wait the server asked for", async () => {
    mocks.signup.mockRejectedValue(
      new ApiRequestError(
        429,
        {
          type: "https://trenova.app/problems/rate-limit-exceeded",
          title: "Rate Limit Exceeded",
          status: 429,
        },
        1200,
      ),
    );
    const user = userEvent.setup();
    renderForm();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Pass security check" }));

    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(
      await screen.findByText(
        "Too many signup attempts from this network. Try again in 20 minutes.",
      ),
    ).toBeInTheDocument();
  });

  it("renders the honeypot out of reach of people and assistive technology", () => {
    const { container } = renderForm();
    const honeypot = container.querySelector<HTMLInputElement>('input[name="website"]');

    expect(honeypot).not.toBeNull();
    expect(honeypot).toHaveAttribute("tabindex", "-1");
    expect(honeypot).toHaveAttribute("autocomplete", "off");
    expect(honeypot?.closest('[aria-hidden="true"]')).not.toBeNull();
  });

  it("links the terms and privacy documents from the public config", () => {
    renderForm();

    expect(screen.getByRole("link", { name: "Terms of Service" })).toHaveAttribute(
      "href",
      "https://example.com/terms",
    );
    expect(screen.getByRole("link", { name: "Privacy policy" })).toHaveAttribute(
      "href",
      "https://example.com/privacy",
    );
  });
});
