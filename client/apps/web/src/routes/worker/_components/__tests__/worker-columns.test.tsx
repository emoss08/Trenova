import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { WorkerRow } from "@/lib/graphql/worker-table";
import { describe, expect, it } from "vitest";
import { getColumns } from "../worker-columns";

type Meta = {
  label?: string;
  apiField?: string;
  filterable?: boolean;
  sortable?: boolean;
  filterType?: string;
  filterOptions?: ReadonlyArray<{ value: string }>;
};

function columnByApiField(field: string): ColumnDef<WorkerRow> {
  const match = getColumns().find(
    (column) => (column.meta as Meta | undefined)?.apiField === field,
  );
  if (!match) {
    throw new Error(
      `no column for ${field}; have ${getColumns()
        .map((column) => (column.meta as Meta | undefined)?.apiField)
        .join(", ")}`,
    );
  }
  return match;
}

describe("worker roster HR columns", () => {
  // Six phases of HR work are invisible if the roster cannot show them. Each
  // of these reads a roll-up the server denormalises onto the profile.
  it.each([
    ["profile.complianceStatus", "Compliance"],
    ["profile.trainingHealth", "Training"],
    ["profile.safetyRating", "Safety"],
    ["profile.nextCredentialExpiry", "Expires Next"],
  ])("offers a %s column", (apiField, label) => {
    const column = columnByApiField(apiField);
    expect((column.meta as Meta).label).toBe(label);
  });

  // Displaying them is only half the job: HR works from a filtered list, so
  // every roll-up column has to be filterable and sortable. These resolve
  // through the profile relation, which the worker entity must declare
  // queryable server-side for the field to reach the joined table at all.
  it.each([
    "profile.complianceStatus",
    "profile.trainingHealth",
    "profile.safetyRating",
    "profile.nextCredentialExpiry",
  ])("lets HR filter and sort by %s", (apiField) => {
    const meta = columnByApiField(apiField).meta as Meta;
    expect(meta.filterable).toBe(true);
    expect(meta.sortable).toBe(true);
  });

  it("filters training and safety from a fixed list of states", () => {
    const training = columnByApiField("profile.trainingHealth").meta as Meta;
    expect(training.filterType).toBe("select");
    expect(training.filterOptions?.map((option) => option.value)).toEqual(
      expect.arrayContaining(["Current", "Overdue", "Expired", "Missing"]),
    );

    const safety = columnByApiField("profile.safetyRating").meta as Meta;
    expect(safety.filterType).toBe("select");
    expect(safety.filterOptions?.map((option) => option.value)).toEqual(
      expect.arrayContaining(["Excellent", "Good", "Watch", "AtRisk"]),
    );
  });

  // "Everything expiring before a date" is the question the column exists to
  // answer, so the default operator has to be a bound, not an equality.
  it("defaults the expiry filter to a cutoff rather than an exact date", () => {
    const meta = columnByApiField("profile.nextCredentialExpiry").meta as Meta & {
      defaultFilterOperator?: string;
    };
    expect(meta.filterType).toBe("date");
    expect(meta.defaultFilterOperator).toBe("lte");
  });
});
