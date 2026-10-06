# Retiring the legacy seed accounts

`trenova cloud retire-seed-accounts` removes the accounts the `AdminAccount` seed created
with the published password `admin123!` while that seed still ran outside development. Trenova
Cloud production ran it, so production may still hold:

| Username | Email | Organization the seed put it in |
|---|---|---|
| `admin` | `admin@trenova.app` | Trenova Logistics (`TRNV`), member of both |
| `admin-logistics` | `admin.logistics@trenova.app` | Trenova Logistics (`TRNV`) |
| `admin-transport` | `admin.transport@trenova.app` | Trenova Transportation (`TTNV`) |

The seed now runs only in `development` and `test` and creates only `admin`; this command
cleans up what it left behind in Cloud. It ships with the Cloud edition only.

## Usage

```bash
trenova cloud retire-seed-accounts                 # dry run: read-only, prints the plan
trenova cloud retire-seed-accounts --output json   # the same plan as JSON
trenova cloud retire-seed-accounts --apply         # perform it
```

`--dry-run` is the default and runs every read inside one read-only transaction; it needs only
the database. `--apply` also connects to Redis (sessions, cached permissions and the audit
buffer). Passing `--apply` together with an explicit `--dry-run` is refused. The command exits
non-zero when a step after the commit failed and lists it under "Failed steps". Running it
again ends the accounts' sessions again; a cached permission that could not be cleared expires
with its TTL, and an audit entry that could not be recorded was logged at error in full and must
be filed by hand.

Run the dry run first, read it, then apply.

## How an account is identified

Every condition must hold, so a customer's account is never matched:

1. The username is exactly one of the three above **and** the email address (compared without
   case) is the one the seed gave that username. A customer who picked the username
   `admin-logistics` has a different email and is not considered at all.
2. The seed's own record proves it created the row, by at least one of:
   - a `seed_created_entities` row with `seed_name = 'AdminAccount'`,
     `table_name = 'users'` and the user's ID (written by the seeder since that table
     existed), or
   - a password hash that still verifies against `admin123!`, which only the seed set.

An account that matches (1) but not (2), for example one whose password someone changed on a
database without seed tracking, is reported as `unproven` and left untouched; review it by
hand.

An account that is already `Inactive`, locked and holds no memberships, role assignments,
active API keys, MFA factors or open reset links is reported as `already retired`. Its
sessions are ended again and nothing else is changed, which is what makes the command safe
to re-run.

## What retiring does

In one transaction under the system scope, with the matched user and organization rows
locked:

1. Deletes the account's organization memberships, role assignments and MFA factors,
   invalidates its unused password reset links, and revokes every active API key it created
   (`revoked_by_id` is the instance system user). A key created by an account with a public
   password must be assumed compromised, whatever organization it belongs to; the dry run
   counts them and `--output json` lists each one.
2. Counts every row that references the user through any foreign key in the catalog
   (`pg_constraint`, so a new table is covered without a code change; each count stops at
   1000). With no references the user row is deleted, inside a savepoint. Otherwise, or if
   the database still refuses the delete, the row is kept as a tombstone: `Inactive`, locked,
   `must_change_password`, and a bcrypt hash of 32 random bytes nobody knows. The username and
   email stay, so nobody can sign up or be invited as `admin@trenova.app`. In practice
   `admin` is kept: `audit_entries.user_id` is `ON DELETE RESTRICT` and append-only.
3. Inspects the seed's demo organizations (exact name and SCAC above) and deletes one only
   when all of these hold:
   - `seed_created_entities` records the `AdminAccount` seed creating it;
   - it does not host the instance system user;
   - no user belongs to it any more, through `current_organization_id` or a membership,
     tombstones included (`users.current_organization_id` cascades, so deleting an
     organization a user still points at would delete that user);
   - it holds no append-only history (`audit_entries`, `auth_events`, the AI audit tables,
     `ai_logs`), which can never be deleted and whose cascade the database refuses;
   - every other table referencing it holds no row for it, except the defaults
     `tenantbootstrap` and the base seeds create for every organization (controls,
     sequences, roles and resource permissions, GL accounts and account types, document
     types and templates, service failure reason codes, TCA allowlist, agent definitions,
     jurisdiction rules).

   The delete runs in a savepoint; a foreign key, a trigger or a permission refusal keeps the
   organization, and it is reported with the database's reason. Everything else is reported
   for manual review with the tables that hold its data.

After the commit the account's sessions are ended in Redis, its cached permissions are
cleared for every organization it left, and the security audit entries are written.

## Audit trail

Every change is a critical `audit_entries` row written through `services.SecurityAuditor`,
attributed to the system principal, with `metadata.source = "cloud_retire_seed_accounts"`:

| Change | Resource | Operation |
|---|---|---|
| Membership removed | `user` | `update` |
| Role assignment removed | `role` | `unassign` |
| API key revoked | `api_key` | `update` |
| Account deleted | `user` | `delete` |
| Account disabled and locked (MFA factors removed and reset links invalidated are counted in its metadata) | `user` | `lock` |
| Demo organization deleted | `organization` | `delete` |

Each entry is recorded in the organization it concerns. An organization deleted in the same
run cannot hold audit rows, so its entries go to the instance system user's organization with
`metadata.organizationId` naming the deleted one and `metadata.recordedInOrganizationId` the
one it was recorded in. With no system user, those entries are logged at error with every
field and listed under the command's failed steps.

## Row-level security

The repository (`seedaccountrepository`) declares `dbscope.WithSystem` in each of its methods,
each listed with its justification in `internal/cloud/rlsscope_test.go`: the accounts, their
memberships and the demo organizations span tenants, and an operator command holds no tenant
scope.

## Testing

`seedaccountservice/service_test.go` covers the decisions with in-memory fakes;
`seedaccountservice/service_integration_test.go` (build tag `integration`, needs Docker) runs
a dry run, an apply and a second apply against Postgres with a customer account that shares a
seeded username.
