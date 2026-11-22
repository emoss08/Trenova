# Emails

Every email the TMS sends follows one standard: a paper ground, a 600px column, the
company's own mark and name above a white card with an ink border, a mono kicker, one heading
with one highlighted phrase, flat label rows, at most one amber action band with one ink
button, and a footer under the card that says why the reader got it. The design is the
standard written for Managed Trenova's broadcast email; this page is how the TMS keeps every
email on it.

## Where things live

| What | Where |
|---|---|
| The shell, the blocks, the structure check | `pkg/emaildesign` |
| The block functions a template calls | `templatefuncs.EmailFuncMapFor`, used by the engine for `EmailHTML` only |
| Built-in emails (HTML, text, subject per kind) | `internal/modules/documents/domain/documenttemplate/starters/assets/<kind>.email.*` |
| Rendering a message for a send | `documenttemplateservice.RenderMessage` (and `PreviewVersion` for the editor and test sends) |
| The company header (name, icon or logo) | `documenttemplateservice/brand.go`, cached per organization for ten minutes |
| Emails composed for `workspace` (notification, digest, profile test) | `emailports.Composer`, implemented by `documents/app/workspaceemail` |
| The preview and the structure test | `internal/modules/documents/app/emailpreview` |

## How an email is built

A template holds content and nothing of the shell. It is a flow of block calls:

```
{{ preheader (t "One sentence that says what the email is.") }}
{{ kicker (t "Invoice") }}
{{ heading "Invoice {0} <hl>is attached</hl>." .InvoiceNumber }}
{{ para (t "Hello {0}," .CustomerName) }}
{{ facts (fact (t "Due") .DueDate) (fact (t "Total") .Total) }}
{{ action (t "Open tracking page") .TrackingURL (t "The link works until {0}." .ExpiresAt) }}
{{ footer (t "You received this email because ...") }}
```

`emaildesign.Compose` then sanitizes the rendered content, lifts out the preheader and the
footer, splits the card around the action band, and wraps it in the document: doctype, meta
tags, the one style block, the hidden preheader, the header, the card and the footer. The
same call runs for a built-in and for an organization's own version, so a customized template
is wrapped exactly like a starter.

The blocks, each defined once in `pkg/emaildesign`:

| Block | Call |
|---|---|
| Preheader | `preheader text` |
| Kicker, section kicker | `kicker text`, `section text` |
| Heading | `heading "message with <hl>one phrase</hl>" args...` (translates itself) |
| Paragraph, sign-off | `para lines...`, `signoff lines...` |
| Fact rows | `facts (fact label value [detail])...` or `facts .CollectionOfLabelValue` |
| Numbered rows | `steps items...` or, in a range, `row (step $i) (cell ...)` |
| Data table | `tableOpen (col ...) (numCol ...)`, `row (cell ...) (num ...)`, `totalRow (wide n ...) (num ...)`, `tableClose`; `cell`, `num`, `label`, `item title detail [url]` |
| Quote, code box, notice | `quote text`, `code text`, `notice text` |
| A second action | `link url label` (an underlined link in the body) |
| Action band | `action label url [sentence] [kicker]`, renders nothing without a safe URL |
| Footer | `footer text [manageLabel manageURL]` |

Every block escapes what it is given and drops a link that is not `https`, `http` or `mailto`.
A text channel has `heading` too, which writes the same sentence without the highlight, so a
plain-text part says the same heading. The plain-text part ends with `--` and the same footer
line as the HTML.

## Rules

- **No markup in a starter.** `TestEmailStartersHoldNoMarkupOfTheirOwn` fails on any tag in an
  email starter other than `<hl>` inside a heading. A search for `#ffa31a` or
  `bgcolor="#121210"` finds `pkg/emaildesign` and nothing else.
- **The header is the tenant's.** The brand is the organization's name with its icon (or, with
  none, its logo) sized to 30px tall. Trenova's mark appears in no email. An email to the
  company's own people may say "Trenova" in its words; an email to customers, carriers,
  drivers or a factor never does.
- **One heading, one highlight, one band.** `emaildesign.Check` reports a missing preheader,
  a missing footer, more than one band or not exactly one heading. The publish gate runs it
  against the kind's sample, so an organization cannot publish an email off the standard;
  `Compose` reports the same findings as warnings on a live render.
- **Translations.** The extractor (`i18n/tools/extract-templates.mjs`) reads `t "…"` and
  `heading "…"`. A heading's `<hl>` tag is part of the message, and `i18n.mjs merge` refuses a
  translation that drops it.
- **A new email kind** gets its HTML, text and subject starter, a sample context and a case in
  `emailpreview/catalog.go`; `TestEveryEmailKindHasAPreview` fails without one.

## The preview

```
cd services/tms
task email-preview -- -out <dir> -locales en,es
```

It renders every email, every notable variant (with and without a logo, with and without an
action, long tables), from the kinds' fixed sample data into `<dir>/<locale>/<kind>-<n>.html`
and `.txt`, with an `index.html` grouped by area. It goes through the same engine, starters and
`Compose` as a send; only image inlining and the database lookups are skipped. The sample
company is fictional (`templatecatalog.SampleCompanyName`).

`TestEveryEmailFollowsTheStandard` renders the same cases in every supported language and
asserts the standard's structure: the document shell, one preheader that is not the subject,
the company header, one heading with one highlighted phrase, at most one band with its one
button inside it, the footer under the card, and a plain-text part with the same footer line.
