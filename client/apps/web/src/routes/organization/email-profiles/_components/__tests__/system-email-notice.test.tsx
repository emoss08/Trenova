import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SystemEmailNotice } from "../system-email-notice";

describe("SystemEmailNotice", () => {
  it("tells an administrator Trenova sends system email, and from which address", () => {
    render(<SystemEmailNotice fromAddress="no-reply@mail.trenova.app" />);

    expect(screen.getByText("System email is sent by Trenova on your behalf")).toBeTruthy();
    expect(
      screen.getByText(
        "Invitations, password resets and notifications go out from no-reply@mail.trenova.app until a profile is assigned to General. Email to anyone outside your organization needs your own profile.",
      ),
    ).toBeTruthy();
  });
});
