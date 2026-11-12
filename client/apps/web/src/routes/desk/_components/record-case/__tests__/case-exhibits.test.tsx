import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaseExhibits } from "../case-exhibits";
import { deskCase } from "./case-fixture";

vi.mock("../case-route", () => ({ CaseRoute: () => <div data-testid="route" /> }));

const NOW = 1_800_000_000;

afterEach(cleanup);

describe("a case's evidence column", () => {
  it("draws each exhibit the case has", () => {
    const data = deskCase();
    render(<CaseExhibits data={data} lit={null} now={NOW} timezone="UTC" />);

    expect(screen.getByRole("complementary", { name: "Evidence" })).toBeTruthy();
    expect(document.querySelectorAll(".dk-cz-exh")).toHaveLength(data.exhibits.length);
  });

  it("draws nothing at all when the case has no evidence", () => {
    const { container } = render(
      <CaseExhibits data={deskCase({ exhibits: [] })} lit={null} now={NOW} timezone="UTC" />,
    );

    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByRole("complementary", { name: "Evidence" })).toBeNull();
  });
});
