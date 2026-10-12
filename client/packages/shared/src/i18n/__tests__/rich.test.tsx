import type * as Locales from "@trenova/shared/i18n/generated/locales";
import { parseRich, renderRich, useRichT } from "@trenova/shared/i18n/rich";
import { requireCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const SPANISH: Record<string, string> = {
  // The translation moves the bold amount to the end of the sentence.
  "You've used <b>{0}</b> of this month's AI allowance":
    "Ha consumido del límite de IA de este mes <b>{0}</b>",
  "Press <kbd/> to approve": "Pulse <kbd/> para aprobar",
  "Renamed to {0}: <b>saved</b>": "Renombrado a {0}: <b>guardado</b>",
  // Broken: the closing tag is missing.
  "Invitation sent to <b>{0}</b>": "Invitación enviada a <b>{0}",
  "{0, plural, one {# shipment} other {# shipments}} need <b>review</b>":
    "{0, plural, one {# envío necesita} other {# envíos necesitan}} <b>revisión</b>",
};

vi.mock("@trenova/shared/i18n/generated/locales", async (importOriginal) => {
  const actual = await importOriginal<typeof Locales>();
  return {
    ...actual,
    CATALOG_LOADERS: { ...actual.CATALOG_LOADERS, es: { core: async () => SPANISH } },
  };
});

const bold = { b: (children: React.ReactNode) => <strong>{children}</strong> };

function Rich({
  message,
  tags,
  args,
}: {
  message: string;
  tags: Parameters<typeof renderRich>[2];
  args: unknown[];
}) {
  const rt = useRichT();
  return <p data-testid="out">{rt(message, tags, ...args)}</p>;
}

function html(): string {
  return screen.getByTestId("out").innerHTML;
}

beforeAll(() => {
  requireCatalog("core");
});

afterEach(async () => {
  cleanup();
  await act(() => setLocale("en"));
});

describe("parseRich", () => {
  it("splits a sentence at the declared tags only", () => {
    expect(parseRich("Use <kbd/> or <b>both</b> <i>x</i>", ["kbd", "b"])).toEqual([
      { tag: null, text: "Use " },
      { tag: "kbd", text: null },
      { tag: null, text: " or " },
      { tag: "b", text: "both" },
      { tag: null, text: " <i>x</i>" },
    ]);
  });

  it("refuses an unclosed, stray or nested tag", () => {
    expect(parseRich("Sent to <b>{0}", ["b"])).toBeNull();
    expect(parseRich("Sent </b> to", ["b"])).toBeNull();
    expect(parseRich("<b>a <b>b</b></b>", ["b"])).toBeNull();
  });
});

describe("rt", () => {
  it("renders the whole sentence with its markup in English", () => {
    render(
      <Rich
        message="You've used <b>{0}</b> of this month's AI allowance"
        tags={bold}
        args={["82%"]}
      />,
    );
    expect(html()).toBe("You've used <strong>82%</strong> of this month's AI allowance");
  });

  it("lets a translation move the marked-up words", async () => {
    await act(() => setLocale("es"));
    render(
      <Rich
        message="You've used <b>{0}</b> of this month's AI allowance"
        tags={bold}
        args={["82%"]}
      />,
    );
    expect(html()).toBe("Ha consumido del límite de IA de este mes <strong>82%</strong>");
  });

  it("renders a self-closing tag with no text", async () => {
    await act(() => setLocale("es"));
    render(
      <Rich message="Press <kbd/> to approve" tags={{ kbd: () => <kbd>⌘↵</kbd> }} args={[]} />,
    );
    expect(html()).toBe("Pulse <kbd>⌘↵</kbd> para aprobar");
  });

  it("never reads an argument as markup", async () => {
    await act(() => setLocale("es"));
    render(<Rich message="Renamed to {0}: <b>saved</b>" tags={bold} args={["<b>Q3</b>"]} />);
    expect(html()).toBe("Renombrado a &lt;b&gt;Q3&lt;/b&gt;: <strong>guardado</strong>");
  });

  it("falls back to the English sentence when a translation breaks its tags", async () => {
    await act(() => setLocale("es"));
    render(<Rich message="Invitation sent to <b>{0}</b>" tags={bold} args={["ana@x.com"]} />);
    expect(html()).toBe("Invitation sent to <strong>ana@x.com</strong>");
  });

  it("keeps the words when even the source does not parse", () => {
    render(<Rich message="Sent to <b>{0}" tags={bold} args={["ana@x.com"]} />);
    expect(html()).toBe("Sent to ana@x.com");
  });

  it("formats a plural outside the tags in the reader's language", async () => {
    await act(() => setLocale("es"));
    render(
      <Rich
        message="{0, plural, one {# shipment} other {# shipments}} need <b>review</b>"
        tags={bold}
        args={[3]}
      />,
    );
    expect(html()).toBe("3 envíos necesitan <strong>revisión</strong>");
  });
});
