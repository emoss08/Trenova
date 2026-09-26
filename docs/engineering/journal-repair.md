# Journal repair

Two write paths used to skip the general ledger:

- **Adjustment credit memos** (credit-only, credit-and-rebill, full-reversal and write-off
  adjustments) posted without a `CreditMemoPosted` journal and without the customer ledger line
  that takes the credit off the customer's balance.
- **Driver settlement payments** (`MarkPaid`) moved a settlement to Paid without the payment
  journal that clears the payable against cash.

Both now write their entries when the record posts or is paid. `trenova db repair-journals`
backfills the records written before that, once per record.

```bash
trenova db repair-journals --dry-run         # report what would be written; writes nothing
trenova db repair-journals                   # every organization
trenova db repair-journals --org org_01H...  # one organization
```

## What it writes

| Record | Found by | Writes |
|---|---|---|
| Adjustment credit memo, Posted, no customer ledger line | `journalrepairrepository.ListUnjournaledAdjustmentMemos` | `CreditMemoPosted` journal (Dr revenue, Cr AR) and the ledger line, through `invoiceledger.Poster` |
| Write-off memo, or memo whose `CreditMemoPosted` journal already exists | same | the ledger line only (the write-off journal is written by the write-off itself) |
| Paid driver settlement, net pay ≠ 0, no `paid_journal_batch_id` | `ListUnjournaledDriverPayments` | `DriverSettlementPaid` journal (Dr the snapshotted posted payable account, or the default settlements payable for a settlement posted before the snapshot existed; Cr the default cash account) on the paid date, then stamps `paid_journal_batch_id` |
| Paid settlement whose payment journal exists but was never stamped | same | stamps `paid_journal_batch_id` with the existing batch; no new journal |

The request builders are the ones the live paths use (`invoiceledger.CreditMemoRequest`,
`driversettlementservice.PaymentJournal`), so a repaired entry is indistinguishable from one
written at the time. Each record runs in its own transaction; the idempotency keys are the live
ones, so a second run finds nothing.

## Periods and review

Each journal is dated on its record's own date (the memo's invoice date, the settlement's paid
date) and goes through `journalposting.ResolvePeriod`: an open or locked period takes it; a closed
one follows the organization's closed-period policy, moving it to the next open period or skipping
the record. With journal posting set to **Manual** the entries land on **Journals to post** for
review; with **Automatic** they post. The author recorded is the system user.

## Skipped records

A record the repair cannot journal is listed with its reason and left unchanged: no accounting
accounts configured, no fiscal period covering its date, a closed period under the reject policy,
revenue not recognized on invoice post (no ledger entry is due), a credit memo whose invoice
adjustment cannot be found, or a settlement whose posting wrote no journal (no payable was booked,
so there is nothing to clear). Correct the cause and run again. Any other error stops
the run; the records already repaired stay repaired.
