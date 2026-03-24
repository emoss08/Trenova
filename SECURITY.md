# Security Policy

## Supported Versions

Trenova is currently in active development and the repository README states that it is not yet suitable for production use. Because of that, the usual long-lived support matrix for multiple release lines does not apply yet.

At this stage, security fixes are only expected to land on the latest development state of the project.

| Version | Supported |
| ------- | --------- |
| Latest development branch / newest tagged release | :white_check_mark: |
| Older tags, images, or snapshots | :x: |
| Modified forks and private downstream patches | :x: |

When Trenova begins maintaining multiple supported release lines, this table will be updated to reflect the actual support window for each line.

## Reporting a Vulnerability

If you believe you have found a security vulnerability in Trenova, do not open a public GitHub issue, discussion, or pull request.

Please report it privately by email to `eric.moss08@gmail.com`.

Include as much of the following as you can:

- A clear summary of the issue and the affected component.
- The repository path, service, route, workflow, or UI surface involved.
- Steps to reproduce the issue.
- Any required configuration, feature flags, roles, or permissions.
- A proof of concept, sample request, or sanitized payload if one is available.
- The security impact you expect, such as authentication bypass, privilege escalation, data exposure, remote code execution, SSRF, or denial of service.
- Any known mitigations or workarounds.
- Whether you believe the issue is already being exploited in the wild.

Please do not include real secrets, customer data, or other sensitive production information unless it is strictly necessary to explain the issue. If sensitive material is necessary, minimize it and clearly label it.

## What To Expect

We will try to follow this process for private vulnerability reports:

1. Acknowledge receipt within 3 business days.
2. Review and validate the report.
3. Request clarification or reproduction details if needed.
4. Classify severity and determine the remediation plan.
5. Prepare and ship a fix in the latest supported code line.
6. Coordinate disclosure once a fix or mitigation is available.

While timelines can vary based on severity, complexity, and reproduction quality, we aim to provide a status update at least every 5 business days while the report is active.

If the report is accepted, we will normally:

- Confirm that the issue is valid.
- Work on a remediation or mitigation.
- Credit the reporter if they want public acknowledgment.
- Share disclosure timing once it is safe to do so.

If the report is not accepted, we will explain why when we reasonably can. Common reasons include:

- The behavior is working as intended and does not create a security boundary failure.
- The issue cannot be reproduced with the information provided.
- The report depends on an unsupported deployment, outdated build, or modified fork.
- The concern is a best-practice recommendation without a demonstrable exploit path.

## Coordinated Disclosure

We ask reporters to avoid public disclosure until:

- We have confirmed and fixed the issue, or
- We have shipped a mitigation and agreed on a disclosure date together.

If a report affects third-party dependencies, infrastructure, or another upstream project, we may need to coordinate disclosure with those maintainers before publishing details.

## Scope

This policy applies to security issues in this repository, including the main TMS backend, shared packages, deployment assets, and the frontend client.

The following are generally out of scope unless there is a clear, demonstrable security impact:

- Requests for general hardening advice without a specific vulnerability.
- Issues that require unrealistic attacker assumptions.
- Problems limited to local development defaults in isolated environments.
- Reports against custom forks, local modifications, or unsupported old builds.

## Safe Harbor

We support good-faith security research intended to improve Trenova's security. If you act in good faith, avoid privacy violations, avoid destructive testing, and give us a reasonable opportunity to remediate the issue before disclosure, we will treat your research as authorized under this policy.
