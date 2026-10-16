import type { AuthEventRow } from "@/lib/graphql/security-activity-table";
import { render, screen } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { describe, expect, it } from "vitest";
import {
  authEventColumns,
  externalIdentityColumns,
  mfaAuthenticatorColumns,
  riskDecisionColumns,
} from "../_components/security-access/activity/activity-columns";

/** A row as the `authEvents` connection returns it (security_activity.graphqls). */
const signOut = {
  id: "aue_01",
  userId: "usr_01",
  identityProviderId: null,
  provider: "session.logout",
  outcome: "success",
  ipAddress: "",
  userAgent: "",
  authenticatorAal: 1,
  federationFal: 1,
  mfaState: "",
  riskOutcome: "allow",
  riskSignals: [],
  errorCode: "",
  occurredAt: 1_760_000_000,
  createdAt: 1_760_000_000,
  user: null,
} as unknown as AuthEventRow;

function renderCell(column: ColumnDef<AuthEventRow>, row: AuthEventRow) {
  const cell = column.cell;
  if (typeof cell !== "function") {
    throw new Error("column has no cell renderer");
  }
  render(<>{cell({ row: { original: row } } as never)}</>);
}

function filterableFields<T>(columns: ColumnDef<T>[]): string[] {
  return columns
    .filter((column) => column.meta?.filterable)
    .map((column) => column.meta?.apiField ?? "");
}

describe("authEventColumns", () => {
  const columns = authEventColumns(translate);

  it("names the event in words rather than by its provider key", () => {
    renderCell(columns[0]!, signOut);

    expect(screen.getByText("Signed out")).toBeInTheDocument();
  });

  it("says who it was when the event names nobody", () => {
    const person = columns.find((column) => column.id === "user")!;
    renderCell(person, signOut);

    expect(screen.getByText("Not identified")).toBeInTheDocument();
  });

  it("labels an outcome in words", () => {
    const outcome = columns.find((column) => "accessorKey" in column && column.accessorKey === "outcome")!;
    renderCell(outcome, { ...signOut, outcome: "denied" } as AuthEventRow);

    expect(screen.getByText("Refused")).toBeInTheDocument();
  });

  it("filters only on fields the server holds as text, never on the address", () => {
    expect(filterableFields(columns)).toEqual([
      "provider",
      "outcome",
      "user.name",
      "riskOutcome",
      "errorCode",
      "userAgent",
      "occurredAt",
    ]);
  });
});

describe("the other activity tables", () => {
  it("filter on the fields their connections expose", () => {
    expect(filterableFields(riskDecisionColumns(translate))).toEqual([
      "outcome",
      "user.name",
      "reason",
      "createdAt",
    ]);
    expect(filterableFields(externalIdentityColumns(translate))).toEqual([
      "externalEmail",
      "user.name",
      "externalSubject",
      "lastLoginAt",
      "createdAt",
    ]);
    expect(filterableFields(mfaAuthenticatorColumns(translate))).toEqual([
      "name",
      "user.name",
      "enabled",
      "verifiedAt",
      "lastUsedAt",
    ]);
  });
});
