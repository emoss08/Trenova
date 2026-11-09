import { recordPath } from "@/config/record-links";
import { statusChoices } from "@/lib/choices";
import { factoringNoaDisplayStatus } from "@/lib/factoring";
import type { FactoringNoa } from "@/lib/graphql/factoring";
import type { FactoringCompanyCustomer } from "@/lib/graphql/factoring-company-page";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

type Step = "done" | "now" | "good" | "todo";

const STEP_DOT: Record<Step, string> = {
  done: "bg-foreground border-foreground",
  now: "bg-warning border-warning",
  good: "bg-success border-success",
  todo: "bg-card border-border-strong",
};

const STEP_TEXT: Record<Step, string> = {
  done: "text-muted-foreground",
  now: "text-foreground font-medium",
  good: "text-success-foreground font-medium",
  todo: "text-foreground-subtle",
};

/** Drafted, sent, acknowledged: where a customer's notice is on its way to letting it be factored. */
function NoticeTrack({ noa, nowUnix }: { noa: FactoringNoa | null; nowUnix: number }) {
  const t = useT();
  const shown = noa ? factoringNoaDisplayStatus(noa, nowUnix) : null;
  const reached = shown === "Draft" ? 1 : shown === "Sent" ? 2 : shown === "Acknowledged" ? 3 : 0;
  const names = [t("Drafted"), t("Sent"), t("Acknowledged")];
  const step = (index: number): Step => {
    const position = index + 1;
    if (position < reached) return "done";
    if (position === reached) return shown === "Acknowledged" ? "good" : "now";
    return "todo";
  };

  if (shown === "Expired") {
    return (
      <span className="text-danger-foreground text-xs font-medium">
        {t("Acknowledged, signed copy expired")}
      </span>
    );
  }

  return (
    <span className="flex items-center gap-1.5">
      {names.map((name, index) => (
        <span key={name} className="flex items-center gap-1.5">
          <span
            className={cn(
              "flex items-center gap-1.5 text-xs whitespace-nowrap",
              STEP_TEXT[step(index)],
            )}
          >
            <span
              aria-hidden
              className={cn("size-2 rounded-full border-[1.5px]", STEP_DOT[step(index)])}
            />
            {name}
          </span>
          {index < names.length - 1 ? (
            <span
              aria-hidden
              className={cn("h-px w-4", index + 1 < reached ? "bg-foreground" : "bg-border")}
            />
          ) : null}
        </span>
      ))}
    </span>
  );
}

function noteFor(t: TranslateFn, customer: FactoringCompanyCustomer, nowUnix: number) {
  const noa = customer.notice;
  if (!noa) {
    return customer.lastReleasedAt
      ? { text: t("Last notice released {0}", formatUnixDate(customer.lastReleasedAt)), warn: true }
      : { text: t("No notice yet; invoices already print the factor's remit-to"), warn: true };
  }
  switch (factoringNoaDisplayStatus(noa, nowUnix)) {
    case "Draft":
      return { text: t("Drafted {0}", formatUnixDate(noa.createdAt)), warn: false };
    case "Sent": {
      const days = noa.sentAt ? Math.floor((nowUnix - noa.sentAt) / 86_400) : 0;
      return {
        text: t(
          "Sent {0} · {1, plural, one {# day} other {# days}} without an answer",
          noa.sentAt ? formatUnixDate(noa.sentAt) : "",
          days,
        ),
        warn: days >= 14,
      };
    }
    case "Acknowledged":
      return {
        text: noa.expiresAt
          ? t(
              "Acknowledged {0} · signed copy expires {1}",
              noa.acknowledgedAt ? formatUnixDate(noa.acknowledgedAt) : "",
              formatUnixDate(noa.expiresAt),
            )
          : t("Acknowledged {0}", noa.acknowledgedAt ? formatUnixDate(noa.acknowledgedAt) : ""),
        warn: false,
      };
    case "Expired":
      return {
        text: t("Expired {0}", noa.expiresAt ? formatUnixDate(noa.expiresAt) : ""),
        warn: true,
      };
    case "Released":
      return { text: t("Released"), warn: false };
  }
}

export type FactoringCustomerActions = {
  canCreate: boolean;
  canUpdate: boolean;
  onDraft: (customerId: string) => void;
  onSend: (noa: FactoringNoa) => void;
  onAcknowledge: (noa: FactoringNoa) => void;
  onRelease: (noa: FactoringNoa) => void;
};

/** The one step that moves a customer's notice on, and the others it allows. */
function NoticeActions({
  customer,
  nowUnix,
  actions,
}: {
  customer: FactoringCompanyCustomer;
  nowUnix: number;
  actions: FactoringCustomerActions;
}) {
  const t = useT();
  const noa = customer.notice ?? null;
  const shown = noa ? factoringNoaDisplayStatus(noa, nowUnix) : null;

  return (
    <span className="flex justify-end gap-1">
      {!noa && actions.canCreate ? (
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => actions.onDraft(customer.id)}
        >
          {t("Draft notice")}
        </Button>
      ) : null}
      {noa && shown === "Draft" && actions.canUpdate ? (
        <Button type="button" size="xs" variant="outline" onClick={() => actions.onSend(noa)}>
          {t("Send")}
        </Button>
      ) : null}
      {noa && shown === "Sent" && actions.canUpdate ? (
        <>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => actions.onAcknowledge(noa)}
          >
            {t("Record acknowledgement")}
          </Button>
          <Button type="button" size="xs" variant="ghost" onClick={() => actions.onSend(noa)}>
            {t("Send again")}
          </Button>
        </>
      ) : null}
      {noa && (shown === "Acknowledged" || shown === "Expired") && actions.canUpdate ? (
        <Button type="button" size="xs" variant="ghost" onClick={() => actions.onRelease(noa)}>
          {t("Release")}
        </Button>
      ) : null}
    </span>
  );
}

/**
 * The customers table's columns. Name, code and status are the customer's own fields, so the
 * server searches, filters and sorts on them; the notice and funded figures are read per page and
 * only shown.
 */
export function getFactoringCustomerColumns(
  t: TranslateFn,
  {
    nowUnix,
    canReadMoney,
    actions,
  }: { nowUnix: number; canReadMoney: boolean; actions: FactoringCustomerActions },
): ColumnDef<FactoringCompanyCustomer>[] {
  return [
    {
      accessorKey: "name",
      header: t("Customer"),
      cell: ({ row }) => (
        <span className="flex min-w-0 flex-col">
          <Link
            to={recordPath("customer", row.original.id)}
            className="ui-focus-ring truncate rounded-sm text-sm font-medium hover:underline"
          >
            {row.original.name}
          </Link>
          <span className="text-foreground-subtle font-mono text-xs">{row.original.code}</span>
        </span>
      ),
      size: 240,
      minSize: 180,
      meta: {
        label: t("Customer"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.code}</span>,
      size: 110,
      minSize: 90,
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 110,
      minSize: 100,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "funded",
      header: t("Funded with factor"),
      cell: ({ row }) => (
        <span className="flex flex-col items-end">
          <span className="text-sm tabular-nums">
            {canReadMoney && row.original.fundedCount > 0
              ? formatMinor(row.original.fundedMinor)
              : "—"}
          </span>
          <span className="text-muted-foreground text-xs tabular-nums">
            {row.original.fundedCount > 0
              ? t("{0, plural, one {# invoice} other {# invoices}}", row.original.fundedCount)
              : t("None funded")}
          </span>
        </span>
      ),
      size: 150,
      minSize: 130,
      meta: { label: t("Funded with factor"), sortable: false, filterable: false },
    },
    {
      id: "notice",
      header: t("Notice of assignment"),
      cell: ({ row }) => {
        const note = noteFor(t, row.original, nowUnix);
        return (
          <span className="flex min-w-0 flex-col gap-1">
            <NoticeTrack noa={row.original.notice ?? null} nowUnix={nowUnix} />
            <span
              className={cn(
                "truncate text-xs",
                note.warn ? "text-warning-foreground" : "text-muted-foreground",
              )}
            >
              {note.text}
            </span>
          </span>
        );
      },
      size: 320,
      minSize: 260,
      meta: { label: t("Notice of assignment"), sortable: false, filterable: false },
    },
    {
      id: "sentTo",
      header: t("Sent to"),
      cell: ({ row }) => (
        <span className="text-muted-foreground truncate text-sm">
          {row.original.notice?.sentTo ?? "—"}
        </span>
      ),
      size: 200,
      minSize: 140,
      meta: { label: t("Sent to"), sortable: false, filterable: false },
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <NoticeActions customer={row.original} nowUnix={nowUnix} actions={actions} />
      ),
      size: 270,
      minSize: 240,
      enableHiding: false,
      meta: { label: t("Actions"), sortable: false, filterable: false },
    },
  ];
}
