import { describe, expect, expectTypeOf, it } from "vitest";
import {
  searchBoolean,
  searchCodec,
  searchInteger,
  searchJson,
  searchList,
  searchLiteral,
  searchNumberLiteral,
  searchString,
  type SearchValues,
} from "../search-codec";

describe("searchString", () => {
  it("reads and writes the value as it is", () => {
    expect(searchString.parse("Acme Foods")).toBe("Acme Foods");
    expect(searchString.serialize("a,b:c/d")).toBe("a,b:c/d");
  });
});

describe("searchInteger", () => {
  it("reads a whole number, and the number a value starts with", () => {
    expect(searchInteger.parse("42")).toBe(42);
    expect(searchInteger.parse("-3")).toBe(-3);
    expect(searchInteger.parse("12abc")).toBe(12);
  });

  it("reads a value that is not a number as nothing", () => {
    expect(searchInteger.parse("abc")).toBeNull();
    expect(searchInteger.parse("")).toBeNull();
  });

  it("writes a whole number", () => {
    expect(searchInteger.serialize(7)).toBe("7");
    expect(searchInteger.serialize(2.6)).toBe("3");
  });
});

describe("searchBoolean", () => {
  it("reads true in any case and anything else as false", () => {
    expect(searchBoolean.parse("true")).toBe(true);
    expect(searchBoolean.parse("TRUE")).toBe(true);
    expect(searchBoolean.parse("false")).toBe(false);
    expect(searchBoolean.parse("yes")).toBe(false);
    expect(searchBoolean.serialize(true)).toBe("true");
  });
});

describe("searchLiteral", () => {
  const tab = searchLiteral(["general", "security"]);

  it("reads one of its values and nothing else", () => {
    expect(tab.parse("security")).toBe("security");
    expect(tab.parse("billing")).toBeNull();
  });
});

describe("searchNumberLiteral", () => {
  const zoom = searchNumberLiteral([0.5, 1, 2]);

  it("reads one of its numbers and nothing else", () => {
    expect(zoom.parse("0.5")).toBe(0.5);
    expect(zoom.parse("2")).toBe(2);
    expect(zoom.parse("3")).toBeNull();
    expect(zoom.parse("wide")).toBeNull();
    expect(zoom.serialize(0.5)).toBe("0.5");
  });
});

describe("searchList", () => {
  const fleets = searchList(searchString);

  it("reads a list joined by commas, and an empty value as no items", () => {
    expect(fleets.parse("north,south")).toEqual(["north", "south"]);
    expect(fleets.parse("")).toEqual([]);
  });

  it("keeps a separator inside an item encoded, so it reads back as one item", () => {
    const written = fleets.serialize(["north, east", "south"]);

    expect(written).toBe("north%2C east,south");
    expect(fleets.parse(written)).toEqual(["north, east", "south"]);
  });

  it("drops the items it cannot read", () => {
    const severities = searchList(searchLiteral(["low", "high"]));

    expect(severities.parse("low,extreme,high")).toEqual(["low", "high"]);
  });

  it("joins by the separator it is given", () => {
    const kinds = searchList(searchString, "|");

    expect(kinds.serialize(["a", "b"])).toBe("a|b");
    expect(kinds.parse("a|b")).toEqual(["a", "b"]);
  });

  it("calls two lists the same when their items are, in order", () => {
    expect(fleets.eq(["a", "b"], ["a", "b"])).toBe(true);
    expect(fleets.eq(["a", "b"], ["b", "a"])).toBe(false);
    expect(fleets.eq(["a"], ["a", "b"])).toBe(false);
  });
});

describe("searchJson", () => {
  const scope = searchJson((value) =>
    typeof value === "object" && value !== null && "days" in value && typeof value.days === "number"
      ? { days: value.days }
      : null,
  );

  it("reads JSON its check accepts", () => {
    expect(scope.parse('{"days":7}')).toEqual({ days: 7 });
    expect(scope.serialize({ days: 7 })).toBe('{"days":7}');
  });

  it("reads JSON its check refuses, or text that is not JSON, as nothing", () => {
    expect(scope.parse('{"weeks":1}')).toBeNull();
    expect(scope.parse("{days:7")).toBeNull();
  });

  it("calls two values the same when they write the same", () => {
    expect(scope.eq({ days: 7 }, { days: 7 })).toBe(true);
    expect(scope.eq({ days: 7 }, { days: 8 })).toBe(false);
  });
});

describe("a codec's default and options", () => {
  it("carries its default, and keeps it through options", () => {
    const page = searchInteger.withDefault(1).withOptions({ history: "push" });

    expect(page.defaultValue).toBe(1);
    expect(page.history).toBe("push");
    expect(page.parse("3")).toBe(3);
  });

  it("is replaced in the history unless it asks to push", () => {
    expect(searchString.history).toBeUndefined();
    expect(searchString.withOptions({ history: "push" }).history).toBe("push");
  });

  it("compares with ===, unless the codec says how", () => {
    const tags = searchCodec<string[]>({
      parse: (value) => value.split("|"),
      serialize: (value) => value.join("|"),
      eq: (a, b) => a.join("|") === b.join("|"),
    });

    expect(searchString.eq("a", "a")).toBe(true);
    expect(tags.eq(["a"], ["a"])).toBe(true);
  });
});

describe("the values a map of codecs reads", () => {
  it("are each codec's own type, never null with a default and possibly null without", () => {
    const map = {
      page: searchInteger.withDefault(1),
      tab: searchLiteral(["general", "security"]),
      fleets: searchList(searchString).withDefault([]),
      open: searchBoolean.withDefault(false).withOptions({ history: "push" }),
    };

    expectTypeOf<SearchValues<typeof map>>().toEqualTypeOf<{
      page: number;
      tab: "general" | "security" | null;
      fleets: string[];
      open: boolean;
    }>();
  });
});
