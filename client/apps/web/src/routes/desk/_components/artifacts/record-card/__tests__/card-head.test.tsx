import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CardHead } from "../record-card";

function head(subtitle?: string) {
  return render(
    <CardHead
      kicker={["ACME-2026", "Contract"]}
      title="Acme dry van contract"
      subtitle={subtitle}
      figure={
        <>
          <b>46</b>
          <span>lanes priced</span>
        </>
      }
    />,
  );
}

describe("CardHead", () => {
  it("sets the title over its subtitle, level with the figure's foot", () => {
    head("Southeast dry van lanes");

    const who = screen.getByRole("heading", { name: "Acme dry van contract", level: 2 })
      .parentElement as HTMLElement;
    expect(who).toHaveClass("dk-rec-who");
    expect(who).not.toHaveClass("dk-alone");
    expect(screen.getByText("Southeast dry van lanes")).toBeInTheDocument();
  });

  // A lone title would sink to the figure's caption and leave a gap under the
  // kicker; marked alone, it stands at the top of the row instead.
  it("marks a title with no subtitle alone, so it holds the top of the row", () => {
    head();

    const who = screen.getByRole("heading", { name: "Acme dry van contract", level: 2 })
      .parentElement as HTMLElement;
    expect(who).toHaveClass("dk-rec-who", "dk-alone");
    expect(who.children).toHaveLength(1);
  });
});
