// assert-codegen-clean.mjs fails when generated output does not match its source.
//
// Two checks, because neither alone is sufficient:
//
//   git diff        catches a tracked artifact that regeneration just changed. This is
//                   the CI case — a clean checkout, codegen runs, drift shows up as a
//                   working-tree modification.
//   ls-files -o     catches an artifact git does not track at all. `git diff` reports
//                   nothing for those, so a newly added generated file would sail
//                   through the gate on the very commit that introduces it.
//
// Deliberately compares the working tree against the index rather than against HEAD:
// output that was regenerated and staged is correct output, and must not fail the gate
// just because the commit has not happened yet.
import { execFileSync } from "node:child_process";

const PATHS = [
  "src/generated",
  "src/schema.graphql",
  "../shared/src/types/generated",
  "../../../services/tms/internal/api/graphql/persisted-documents.json",
];

function git(args) {
  return execFileSync("git", args, { encoding: "utf8" });
}

const problems = [];

try {
  git(["diff", "--exit-code", "--", ...PATHS]);
} catch {
  problems.push(git(["diff", "--stat", "--", ...PATHS]).trim());
}

const untracked = git(["ls-files", "--others", "--exclude-standard", "--", ...PATHS]).trim();
if (untracked !== "") {
  problems.push(`untracked generated output:\n${untracked}`);
}

if (problems.length === 0) {
  process.exit(0);
}

process.stderr.write(
  `Generated artifacts are out of date with their source.\n\n${problems.join("\n\n")}\n\n` +
    "Run `pnpm --filter @trenova/graphql codegen` and commit the result.\n" +
    "GraphQL types come from services/tms/internal/api/graphql/schema/*.graphqls;\n" +
    "error enums come from services/tms/pkg/errortypes/enums.go and\n" +
    "services/tms/internal/api/helpers/problemtypes.go.\n",
);
process.exit(1);
