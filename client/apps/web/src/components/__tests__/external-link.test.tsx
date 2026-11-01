import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExternalLink } from "../link";

afterEach(() => vi.restoreAllMocks());

describe("ExternalLink", () => {
  it("is a button inside a sentence, reached and pressed from the keyboard", async () => {
    const user = userEvent.setup();
    render(
      <p>
        Create an API key in <ExternalLink href="https://resend.com/api-keys">Resend.</ExternalLink>
      </p>,
    );

    const link = screen.getByRole("button", { name: "Resend." });
    expect(link.closest("p")).not.toBeNull();
    await user.tab();
    expect(link).toHaveFocus();
    await user.keyboard("{Enter}");

    expect(await screen.findByRole("alertdialog", { name: "External link" })).toBeInTheDocument();
  });

  it("opens the page in a new tab, without access back to Trenova, only after the person continues", async () => {
    const user = userEvent.setup();
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    render(<ExternalLink href="https://resend.com/api-keys">Resend.</ExternalLink>);

    await user.click(screen.getByRole("button", { name: "Resend." }));
    expect(open).not.toHaveBeenCalled();
    await user.click(await screen.findByRole("button", { name: "Continue" }));

    expect(open).toHaveBeenCalledExactlyOnceWith(
      "https://resend.com/api-keys",
      "_blank",
      "noopener,noreferrer",
    );
  });
});
