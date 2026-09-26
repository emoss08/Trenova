import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ExchangeRateLine, formatExchangeRate } from "../exchange-rate-line";

const APRIL_TENTH = Date.UTC(2026, 3, 10, 12) / 1000;
const APRIL_EIGHTEENTH = Date.UTC(2026, 3, 18, 12) / 1000;

describe("ExchangeRateLine", () => {
  it("renders nothing for a document in the functional currency", () => {
    const { container } = render(<ExchangeRateLine currency="USD" rate={null} quotedOn={null} />);

    expect(container).toBeEmptyDOMElement();
  });

  it("names the rate and the day it was quoted", () => {
    render(<ExchangeRateLine currency="CAD" rate="0.731200000000" quotedOn={APRIL_TENTH} />);

    expect(screen.getByText(/Exchange rate/)).toHaveTextContent(
      "Exchange rate · 1 CAD = 0.7312, quoted Apr 10, 2026",
    );
  });

  it("adds the rate a settlement was paid at", () => {
    render(
      <ExchangeRateLine
        currency="CAD"
        rate="0.7312"
        quotedOn={APRIL_TENTH}
        paidRate="0.7288"
        paidQuotedOn={APRIL_EIGHTEENTH}
      />,
    );

    expect(screen.getByText(/Exchange rate/)).toHaveTextContent(
      "1 CAD = 0.7312, quoted Apr 10, 2026 · paid at 0.7288, quoted Apr 18, 2026",
    );
  });

  it("shows only the paid rate when posting needed none", () => {
    render(
      <ExchangeRateLine
        currency="CAD"
        rate={null}
        quotedOn={null}
        paidRate="0.7288"
        paidQuotedOn={APRIL_EIGHTEENTH}
      />,
    );

    expect(screen.getByText(/Exchange rate/)).toHaveTextContent(
      "Exchange rate · paid at 0.7288, quoted Apr 18, 2026",
    );
  });
});

describe("formatExchangeRate", () => {
  it("trims the stored precision to what a person reads", () => {
    expect(formatExchangeRate("0.731200000000")).toBe("0.7312");
    expect(formatExchangeRate("1.123456789")).toBe("1.123457");
    expect(formatExchangeRate("not a number")).toBe("not a number");
  });
});
