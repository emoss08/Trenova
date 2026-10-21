import { usePermission } from "@/hooks/use-permission";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { CardActionButton } from "./card-action";
import { CardBar, CardBarHint, CardIcon } from "./record-card";
import { Operation } from "@trenova/shared/types/permission";
import {
  SETTLEMENT_ACTION_OPERATION,
  type SettlementAction,
  type Signoff,
} from "./settlement-values";

/**
 * A money record as a stub, its perforated tear-off down one side: PAY for a
 * worker's settlement, AP for a carrier's, DET for a detention charge.
 */
export function SettlementStub({ mark }: { mark: string }) {
  return (
    <span className="dk-rec-stub" aria-hidden>
      <b>{mark}</b>
      <i />
      <i />
    </span>
  );
}

/** Who moved the settlement through each step it has passed, under its track. */
export function SettlementSignoffs({ rows }: { rows: readonly Signoff[] }) {
  if (rows.length === 0) return null;
  return (
    <dl className="dk-rec-signs">
      {rows.map((row) => (
        <div key={row.key}>
          <dt>{row.step}</dt>
          <dd>
            <b>{row.who}</b>
            <span>{row.when}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * Why a settlement was voided, kept as written, with who voided it, when, and
 * how far it had got: before posting nothing reached the ledger, after it the
 * entry was reversed.
 */
export function SettlementVoidSlip({
  reason,
  by,
  when,
  reached,
}: {
  reason: string;
  by: string;
  when: string;
  reached: string;
}) {
  const t = useT();
  return (
    <div className="dk-rec-voidslip">
      {reason.trim() && <blockquote>{reason.trim()}</blockquote>}
      <dl>
        <div>
          <dt>{t("Voided by")}</dt>
          <dd>{by}</dd>
        </div>
        <div>
          <dt>{t("When")}</dt>
          <dd>{when}</dd>
        </div>
        <div>
          <dt>{t("Had reached")}</dt>
          <dd>{reached}</dd>
        </div>
      </dl>
    </div>
  );
}

/** One row of a statement: a way of paying or charging, how it was worked out, and what it came to. */
export type StatementRow = {
  key: string;
  name: string;
  code?: string;
  detail: string;
  /**
   * The kind whose read the row's detail needs, when it was withheld: the detail is drawn as
   * withheld rather than left out (docs/engineering/reference-views.md).
   */
  withheld?: ReferenceKind;
  basis: string;
  amount: string;
  manual?: boolean;
};

/** A run of statement rows under the heading of what they do to the total. */
export type StatementGroup = {
  key: string;
  title: string;
  total: string;
  rows: readonly StatementRow[];
};

/**
 * A settlement's statement as an overview: one row per way of paying,
 * however many lines and loads share it, grouped under what each does to the
 * total, the rows added by hand tagged so. The statement itself, line by line,
 * stays on the settlement's page.
 */
export function SettlementStatement({
  groups,
  foot,
  note,
}: {
  groups: readonly StatementGroup[];
  foot?: { label: string; amount: string };
  note?: string;
}) {
  const t = useT();
  return (
    <>
      <table className="dk-rec-stmt">
        <thead>
          <tr>
            <th>{t("Line")}</th>
            <th>{t("Worked out")}</th>
            <th>{t("Amount")}</th>
          </tr>
        </thead>
        {groups.map((group) => (
          <tbody key={group.key}>
            <tr className="dk-grp">
              <th colSpan={2} scope="rowgroup">
                {group.title}
              </th>
              <th className="dk-amt">{group.total}</th>
            </tr>
            {group.rows.map((row) => (
              <tr key={row.key}>
                <td>
                  <div className="dk-desc">
                    <span>
                      {row.name}
                      {row.code && <span className="dk-rec-stmt-code">{row.code}</span>}
                      {row.manual && <span className="dk-rec-stmt-tag">{t("By hand")}</span>}
                    </span>
                    {row.detail ? (
                      <em>{row.detail}</em>
                    ) : (
                      row.withheld && <WithheldValue kind={row.withheld} />
                    )}
                  </div>
                </td>
                <td className="dk-qty">{row.basis}</td>
                <td className="dk-amt">{row.amount}</td>
              </tr>
            ))}
          </tbody>
        ))}
      </table>
      {foot && (
        <dl className="dk-rec-stmt-foot">
          <dt>{foot.label}</dt>
          <dd>{foot.amount}</dd>
        </dl>
      )}
      {note && <p className="dk-rec-note">{note}</p>}
    </>
  );
}

/**
 * The settlement's bar: the steps beside the way on, quiet at its start, a
 * line on where the rest of the work is done, a way to the settlement's page,
 * and the one step that moves it on. Each step is shown only to someone who
 * may take it, and none is pressed twice while one is running.
 */
export function SettlementBar({
  primary,
  more,
  labelOf,
  may,
  busy,
  onAction,
  hint,
  pageTo,
  pageLabel,
}: {
  primary: SettlementAction | null;
  more: readonly SettlementAction[];
  labelOf: (action: SettlementAction) => string;
  may: (action: SettlementAction) => boolean;
  busy: boolean;
  onAction: (action: SettlementAction) => void;
  hint: string;
  pageTo: string;
  pageLabel: string;
}) {
  const leading = primary !== null && may(primary) ? primary : null;
  return (
    <CardBar>
      {more.filter(may).map((action) => (
        <CardActionButton
          key={action}
          variant="quiet"
          className="dk-rec-step"
          blocker={null}
          busy={busy}
          onClick={() => onAction(action)}
        >
          {action === "void" && <CardIcon name="x" size={12} stroke={2.4} />}
          {labelOf(action)}
        </CardActionButton>
      ))}
      <CardBarHint>{hint}</CardBarHint>
      <Link className={cn("dk-ax-btn", leading === null && "dk-ink")} to={pageTo}>
        <CardIcon name="ext" size={13} />
        {pageLabel}
      </Link>
      {leading !== null && (
        <CardActionButton blocker={null} busy={busy} onClick={() => onAction(leading)}>
          {labelOf(leading)}
        </CardActionButton>
      )}
    </CardBar>
  );
}

/**
 * Whether the person may take each settlement step, under the permission the
 * server checks for it on this kind of settlement, read once for the card.
 */
export function useSettlementSteps(resource: string): (action: SettlementAction) => boolean {
  const { allowed: submit } = usePermission(resource, Operation.Submit);
  const { allowed: approve } = usePermission(resource, Operation.Approve);
  const { allowed: reject } = usePermission(resource, Operation.Reject);
  const { allowed: update } = usePermission(resource, Operation.Update);
  const { allowed: cancel } = usePermission(resource, Operation.Cancel);
  const granted: Record<number, boolean> = {
    [Operation.Submit]: submit,
    [Operation.Approve]: approve,
    [Operation.Reject]: reject,
    [Operation.Update]: update,
    [Operation.Cancel]: cancel,
  };
  return (action) => granted[SETTLEMENT_ACTION_OPERATION[action]] ?? false;
}
