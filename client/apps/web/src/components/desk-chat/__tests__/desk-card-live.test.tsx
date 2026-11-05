import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { DeskCardsLiveProvider } from "../desk-card";
import { DeskErrorCard } from "../desk-error-card";

afterEach(cleanup);

/**
 * A conversation's failures used to be alerts, so opening a conversation read out
 * every reply that had ever failed in it. Only a failure that happens while the
 * person is there is announced.
 */
describe("DeskErrorCard announcements", () => {
  it("does not announce a failure from history read as the conversation opens", () => {
    render(
      <DeskCardsLiveProvider value={false}>
        <DeskErrorCard title="The reply didn't come through" />
      </DeskCardsLiveProvider>,
    );

    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.getByText("The reply didn't come through")).toBeInTheDocument();
  });

  it("announces a failure that happens while the person is there", () => {
    render(
      <DeskCardsLiveProvider value>
        <DeskErrorCard title="The reply didn't come through" />
      </DeskCardsLiveProvider>,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("The reply didn't come through");
  });

  it("keeps an old failure quiet after the history has settled", () => {
    const view = render(
      <DeskCardsLiveProvider value={false}>
        <DeskErrorCard title="The reply didn't come through" />
      </DeskCardsLiveProvider>,
    );
    view.rerender(
      <DeskCardsLiveProvider value>
        <DeskErrorCard title="The reply didn't come through" />
      </DeskCardsLiveProvider>,
    );

    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("announces failures outside a conversation as it always has", () => {
    render(<DeskErrorCard title="Couldn't load the agents" />);

    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't load the agents");
  });
});
