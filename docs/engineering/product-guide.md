# The product guide

Agents answer "where is…" and "how do I…" from the product guide. Every agent
holds two core tools that read it: `find_in_trenova` (pages and step-by-step
tasks, filtered to what the person can open) and `open_page` (moves the app to
a page or record). The prompt also names the page a person is on from it.

The guide is `services/tms/pkg/productguide/catalog_gen.json`. It is generated;
never edit it by hand.

## Where it comes from

`client/apps/web/scripts/product-guide/generate.mjs` builds it from the app's own
code, read statically (nothing is imported or run):

| Source | What it gives |
|---|---|
| `client/apps/web/src/router.tsx` | every page a signed-in person can open, and the permissions and capabilities its loaders (and its parents') demand |
| `client/apps/web/src/config/navigation.config.ts` | module, group and label for each page; the create quick actions |
| each page's `pageHeaderProps` | the page's own one-line description |
| `client/apps/web/src/config/record-links.ts` | where each kind of record opens |
| `docs/product-guide/**.md` | what each page is for and how to do things on it |

Because the structure is read from the router and navigation, moving a page,
renaming it or changing what guards it updates the catalog on the next
generate. The words are the only hand-written part.

## Commands

```bash
cd client/apps/web
pnpm guide:generate   # write the catalog
pnpm guide:check      # fail if it is stale or a guide is wrong (CI runs this)
```

The **GraphQL Codegen** job in `.github/workflows/test-client.yml` runs
`guide:check`. It also runs when only a guide, the catalog or the English i18n
catalog changes, since each of those can make the other stale.

## What fails the build

- A page the router serves with no guide (and not listed in another guide's
  `covers`).
- A guide for a path the router does not serve, or two guides for one path.
- A navigation entry pointing at a path no route serves.
- A **bold** label that is not text the app shows: an i18n string used by the
  client, a navigation label, a quick action, a page title, or a table's
  generated "New {thing}" button.
- A markdown link or a `related` entry that is not a page in the app.
- A record link whose page no route serves.
- A catalog that differs from what the generator would write.

The label check is what keeps "every page has steps" honest. Rename a button and
every guide that tells people to press it fails until it is updated.

## Adding or changing a page

1. Add the route and navigation entry as usual.
2. Write `docs/product-guide/<module>/<page>.md` (format below).
3. `pnpm guide:generate` and commit the catalog with the change.

## Guide format

```markdown
---
path: /billing/configuration-files/rate-matrices
aliases: [rate table, lane rates, pricing grid]
related:
  - /billing/configuration-files/customers
covers:
  - /billing/configuration-files/rate-matrices/new
---

## What it's for
What the page is for and who uses it, in one or two short paragraphs.

## Tasks

### Add a rate matrix
Keywords: new rate, lane pricing
1. Open [Rate matrices](/billing/configuration-files/rate-matrices).
2. Select **New rate matrix**.
3. Fill in **Name** and **Effective date**, then **Save**.

## Notes
Optional: permissions a task needs, what happens after, common mistakes.
```

**Frontmatter**
- `path` (required): the page's route.
- `aliases`: the words people use for this page that aren't in its name.
- `related`: other pages with guides.
- `covers`: other routes this guide documents (a `/new` page, a `:id` detail
  route) that don't get their own guide.
- `title`: only for a page outside the navigation whose header has no title.

**Body**
- `## What it's for` and `## Tasks` are required. `## Notes` is optional.
- Each task is a `### Title`, an optional `Keywords:` line, then numbered steps.
- Continuation lines are indented.

**Writing it**
- Write from the page's code, not from memory: its header, table actions, panel
  fields, buttons, tabs and permissions.
- **Bold** only on-screen text, spelled exactly as the app shows it.
- Link pages with their route.
- Leave out anything you aren't sure the page does. The agents repeat these
  guides to people as fact.
