---
path: /edi/test-cases
aliases: [EDI certification, EDI testing, test scenarios, partner certification, template test]
related:
  - /edi/designer
  - /edi/partners
---

## What it's for
EDI test cases are saved certification scenarios. Each one pairs a stored document payload with a partner document profile and records the warnings and errors the rendered X12 is expected to produce. Running a test case renders the payload through the partner's template and reports whether the result matches.

EDI implementation teams use them to certify a partner's documents and to re-check templates after changes.

## Tasks

### Create a test case
Keywords: new certification scenario, add EDI test
1. Open [Test cases](/edi/test-cases).
2. Select **New EDI test case**.
3. Under **Test case**, enter a **Name**, choose the **Document profile** it certifies, and add a **Description**.
4. Under **Expected outcome**, set **Expected warnings** and **Expected errors**, and optionally list **Expected warning codes** and **Expected error codes**.
5. Under **Document payload**, paste the source data into **Payload**.
6. Select **Create test case**.

### Run a test case and read the result
Keywords: run certification, check template output
1. Open [Test cases](/edi/test-cases) and select the test case's row.
2. Save any edits first, then select **Run preview**.
3. Read the verdict at the top: actual warning and error counts against the expected ones, and any missing or unexpected codes.
4. Select **Open inspector** to see the rendered X12 and its diagnostics. If the result is wrong, fix the payload or template, or update the expected counts and codes.

### Edit or delete a test case
Keywords: change expected errors, remove test
1. Open [Test cases](/edi/test-cases) and select the row.
2. Change the fields and select **Save test case**, or select **Delete** to remove it.

## Notes
Viewing test cases needs read access to EDI; creating needs create access, saving changes needs update access, and deleting needs delete access to EDI. A partner's **Readiness** tab on [Partners](/edi/partners) counts a test case as an onboarding step.
