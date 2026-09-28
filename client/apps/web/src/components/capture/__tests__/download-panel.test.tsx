import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import { CaptureDownloadPanel } from "../download-panel";
import { json, renderWithClient, resetGraphQL, stubGraphQL } from "./capture-graphql-server";

const RELEASE = {
  version: "1.4.0",
  publishedAt: 1_760_000_000,
  minimumWindowsBuild: 19041,
  notes: "",
  installer: {
    fileName: "TrenovaCapture-1.4.0.msi",
    url: "https://downloads.example.test/TrenovaCapture-1.4.0.msi",
    sha256: "ab".repeat(32),
    size: 12_582_912,
  },
};

const MISSING = <p>No installer is published</p>;

describe("CaptureDownloadPanel", () => {
  afterEach(resetGraphQL);

  it("says the release could not be loaded, and offers a retry, when the request fails", async () => {
    const user = userEvent.setup();
    stubGraphQL({
      CaptureAgentRelease: [
        () => json({ errors: [{ message: "upstream unavailable" }] }, 503),
        () => json({ data: { captureAgentRelease: RELEASE } }),
      ],
    });

    renderWithClient(<CaptureDownloadPanel whenMissing={MISSING} />);

    expect(
      await screen.findByText("The Trenova Capture download could not be loaded."),
    ).toBeInTheDocument();
    expect(screen.queryByText("No installer is published")).toBeNull();

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByRole("button", { name: /Download Trenova Capture/ })).toHaveAttribute(
      "href",
      RELEASE.installer.url,
    );
  });

  it("shows the not-published note only when the server answers with no release", async () => {
    stubGraphQL({ CaptureAgentRelease: () => json({ data: { captureAgentRelease: null } }) });

    renderWithClient(<CaptureDownloadPanel whenMissing={MISSING} />);

    expect(await screen.findByText("No installer is published")).toBeInTheDocument();
    expect(screen.queryByText("The Trenova Capture download could not be loaded.")).toBeNull();
  });
});
