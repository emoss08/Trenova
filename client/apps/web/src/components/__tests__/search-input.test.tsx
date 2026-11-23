import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { SearchInput } from "../search-input";

function Search() {
  const [value, setValue] = useState("");
  return <SearchInput value={value} onValueChange={setValue} placeholder="Search changes" />;
}

describe("SearchInput", () => {
  it("offers a clear button only while it holds text, and clearing empties it", () => {
    render(<Search />);
    const field = screen.getByRole("textbox", { name: "Search changes" }) as HTMLInputElement;
    expect(screen.queryByRole("button", { name: "Clear the search" })).toBeNull();

    fireEvent.change(field, { target: { value: "hazmat" } });
    fireEvent.click(screen.getByRole("button", { name: "Clear the search" }));

    expect(field.value).toBe("");
    expect(screen.queryByRole("button", { name: "Clear the search" })).toBeNull();
  });

  it("clears on Escape", () => {
    render(<Search />);
    const field = screen.getByRole("textbox", { name: "Search changes" }) as HTMLInputElement;

    fireEvent.change(field, { target: { value: "hazmat" } });
    fireEvent.keyDown(field, { key: "Escape" });

    expect(field.value).toBe("");
  });
});
