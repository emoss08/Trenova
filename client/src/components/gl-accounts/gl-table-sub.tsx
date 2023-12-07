/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { USDollarFormat } from "@/lib/utils";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";

const glAccountHistory = [
  {
    period: "1",
    end: "2023-01-31",
    beginBalance: 25318772.77,
    debit: 1849004.24,
    credit: 89485.24,
    endBalance: 27078291.77,
  },
  {
    period: "2",
    end: "2023-02-28",
    beginBalance: 84442.19,
    debit: 37897.72,
    credit: 21028.58,
    endBalance: 101311.33,
  },
  {
    period: "3",
    end: "2023-03-28",
    beginBalance: 25891.68,
    debit: 25563.74,
    credit: 20246.71,
    endBalance: 31208.71,
  },
  {
    period: "4",
    end: "2023-04-28",
    beginBalance: 78379.86,
    debit: 15165.64,
    credit: 23829.85,
    endBalance: 69715.65,
  },
  {
    period: "5",
    end: "2023-05-28",
    beginBalance: 58338.2,
    debit: 45405.64,
    credit: 25234.34,
    endBalance: 78509.5,
  },
  {
    period: "6",
    end: "2023-06-28",
    beginBalance: 28183.78,
    debit: 37790.21,
    credit: 30918.45,
    endBalance: 35055.54,
  },
  {
    period: "7",
    end: "2023-07-28",
    beginBalance: 25050.63,
    debit: 45487.31,
    credit: 49139.27,
    endBalance: 21398.67,
  },
  {
    period: "8",
    end: "2023-08-28",
    beginBalance: 81021.72,
    debit: 45108.3,
    credit: 15507.38,
    endBalance: 110622.64,
  },
  {
    period: "9",
    end: "2023-09-28",
    beginBalance: 72983.17,
    debit: 44941.91,
    credit: 34199.2,
    endBalance: 83725.88,
  },
  {
    period: "10",
    end: "2023-10-28",
    beginBalance: 47214.27,
    debit: 5035.06,
    credit: 21708.59,
    endBalance: 30540.74,
  },
  {
    period: "11",
    end: "2023-11-28",
    beginBalance: 61088.7,
    debit: 45650.55,
    credit: 48330.32,
    endBalance: 58408.93,
  },
  {
    period: "12",
    end: "2023-10-31",
    beginBalance: 47700.98,
    debit: 43265.5,
    credit: 13024.62,
    endBalance: 77941.86,
  },
];

const totalAmounts = {
  totalBeginBalance: 25929067.95,
  totalDebit: 2240315.82,
  totalCredit: 392652.55,
  totalEndBalance: 27776731.22,
};

export function GLAccountTableSub() {
  return (
    <div>
      <div className="flex flex-col py-5 pl-3 ">
        <h2 className="scroll-m-20 text-2xl font-semibold tracking-tight">
          General Ledger Account history
        </h2>
        <p className="text-muted-foreground">
          The history of the General Ledger Account. It shows the beginning
          balance, debit, credit, and ending balance for each period.
        </p>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-1/12">Period</TableHead>
            <TableHead className="w-2/12">End</TableHead>
            <TableHead className="w-3/12">Begin Balance</TableHead>
            <TableHead className="w-2/12">Debit</TableHead>
            <TableHead className="w-2/12">Credit</TableHead>
            <TableHead className="w-2/12">End Balance</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {glAccountHistory.map((history) => (
            <TableRow key={history.period} className="border-none">
              <TableCell className="w-1/12 text-xs">{history.period}</TableCell>
              <TableCell className="w-2/12 text-xs">{history.end}</TableCell>
              <TableCell className="w-3/12 text-xs">
                {USDollarFormat(history.beginBalance)}
              </TableCell>
              <TableCell className="w-2/12 text-xs">
                {USDollarFormat(history.debit)}
              </TableCell>
              <TableCell className="w-2/12 text-xs">
                {USDollarFormat(history.credit)}
              </TableCell>
              <TableCell className="w-2/12 text-xs">
                {USDollarFormat(history.endBalance)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
        <TableFooter>
          <TableRow>
            <TableCell colSpan={2}>Total</TableCell>
            {/* Make sure these cells match the width of your table body columns */}
            <TableCell className="w-3/12 text-xs">
              {USDollarFormat(totalAmounts.totalBeginBalance)}
            </TableCell>
            <TableCell className="w-2/12 text-xs">
              {USDollarFormat(totalAmounts.totalDebit)}
            </TableCell>
            <TableCell className="w-2/12 text-xs">
              {USDollarFormat(totalAmounts.totalCredit)}
            </TableCell>
            <TableCell className="w-2/12 text-xs">
              {USDollarFormat(totalAmounts.totalEndBalance)}
            </TableCell>
          </TableRow>
        </TableFooter>
      </Table>
    </div>
  );
}
