---
path: /admin/graphql-explorer
aliases: [GraphQL playground, API explorer, persisted operations, query catalog, run a query, developer tools]
related:
  - /admin/api-keys
  - /admin/audit-logs
---

## What it's for
GraphQL explorer is a developer tool that lists every persisted GraphQL operation the Trenova web
client uses, with the fragments they share. The header shows how many operations and fragments
exist. The left side is a searchable list grouped into **Operations** and **Fragments**; the right
side shows the selected item.

For an operation, the **Definition** tab shows its **Variables**, **Input types**, **Root fields**,
the fragments it uses and the full definition; the **Run** tab lets you execute it with your own
variables and see the **Response**; the usages tab lists the source files that reference it.
Developers and technical administrators use it to understand what the app asks the server for and
to try an operation directly.

## Tasks

### Find an operation or fragment
Keywords: search queries, find mutation, look up GraphQL
1. Open [GraphQL explorer](/admin/graphql-explorer).
2. Type in **Search operations…** to search by name, field or domain.
3. Narrow the list with **All**, **Queries**, **Mutations** or **Fragments**, then select an item.
4. Use **Copy name**, **Copy hash** or **Copy definition** to copy what you need.

### Run an operation
Keywords: execute query, test query, try mutation, send request
1. Open [GraphQL explorer](/admin/graphql-explorer) and select an operation.
2. Select the **Run** tab.
3. Edit the JSON under **Variables** (**Reset** puts back the starting template).
4. Select **Run** (or press Cmd/Ctrl+Enter). The **Response** appears with its status; use **Copy
   response** to copy it.
5. Select **History** to see the last runs of this operation and replay their variables, or
   **Clear** to empty the list.

### See where an operation is used
Keywords: usages, which screen uses this query, fragment used by
1. Open [GraphQL explorer](/admin/graphql-explorer) and select an operation or fragment.
2. Select the usages tab to see the source files that reference it. For a fragment, the
   **Definition** tab also lists the operations that use it and any **Nested fragments**.

## Notes
Opening the page needs read access to the organization. Runs go to the live server under your own
account, so they only return what your permissions allow. A mutation shows "Mutation — executes
against live data": running it really changes records, with no extra confirmation. An operation
with no persisted hash cannot be run.
