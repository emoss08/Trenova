import { beforeEach, describe, expect, it, vi } from "vitest";
import { EDIService } from "../edi";
import { LadingService } from "../edi/lading";
import { EmailService } from "../email";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

function sent() {
  expect(requestGraphQL).toHaveBeenCalledTimes(1);
  return requestGraphQL.mock.calls[0]?.[0] as {
    operationName: string;
    variables: Record<string, unknown>;
  };
}

beforeEach(() => {
  requestGraphQL.mockReset();
});

describe("EDIService over GraphQL", () => {
  it("sends only the mapping fields the input takes when a tender is approved", async () => {
    requestGraphQL.mockResolvedValue({ approveEdiTransfer: { id: "edilt_1", status: "Approved" } });

    await new EDIService().approveTransfer("edilt_1", {
      mappings: [
        {
          id: "",
          businessUnitId: "bu_1",
          organizationId: "org_1",
          entityType: "Customer",
          sourceId: "HEB-SAT",
          sourceLabel: null,
          targetId: "cus_1",
          targetLabel: "H-E-B",
        },
      ],
    });

    expect(sent()).toEqual(
      expect.objectContaining({
        operationName: "ApproveEdiTransfer",
        variables: {
          id: "edilt_1",
          mappings: [
            {
              id: undefined,
              entityType: "Customer",
              sourceId: "HEB-SAT",
              sourceLabel: undefined,
              targetId: "cus_1",
              targetLabel: "H-E-B",
            },
          ],
        },
      }),
    );
  });

  it("reads one partner's readiness, with each step, or null when it is not found", async () => {
    requestGraphQL.mockResolvedValue({
      ediPartnerReadiness: [
        {
          partnerId: "edip_1",
          ready: false,
          completedCount: 1,
          totalCount: 2,
          items: [{ key: "transport", label: "Transport", complete: true }],
        },
      ],
    });
    const readiness = await new EDIService().getPartnerReadiness("edip_1");

    expect(sent().variables).toEqual({ partnerIds: ["edip_1"] });
    expect(readiness?.items).toHaveLength(1);

    requestGraphQL.mockReset().mockResolvedValue({ ediPartnerReadiness: [] });
    await expect(new EDIService().getPartnerReadiness("edip_2")).resolves.toBeNull();
  });

  it("parses a connection test's checks", async () => {
    requestGraphQL.mockResolvedValue({
      testEdiCommunicationProfileConnection: {
        success: false,
        checks: [{ name: "Endpoint", status: "failed", message: null }],
      },
    });
    const result = await new EDIService().testProfileConnection("edicp_1");

    expect(sent().operationName).toBe("TestEdiCommunicationProfileConnection");
    expect(result.checks[0]?.status).toBe("failed");
  });
});

describe("LadingService over GraphQL", () => {
  it("reads null for a partner without a project", async () => {
    requestGraphQL.mockResolvedValue({ ediLadingPartnerProject: null });

    await expect(new LadingService().getPartnerProject("edip_1")).resolves.toBeNull();
    expect(sent()).toEqual(
      expect.objectContaining({
        operationName: "EdiLadingPartnerProject",
        variables: { partnerId: "edip_1" },
      }),
    );
  });

  it("runs a map with its rules, naming the map", async () => {
    requestGraphQL.mockResolvedValue({ runEdiLadingVersion: { ok: true } });

    await new LadingService().run("ldv_1", { map: "tender_in", input: { x12: "ISA*~" } });

    expect(sent().variables).toEqual({
      versionId: "ldv_1",
      mapName: "tender_in",
      input: { x12: "ISA*~" },
      rules: true,
    });
  });
});

describe("EmailService over GraphQL", () => {
  it("lists the newest sends from the log's first page", async () => {
    requestGraphQL.mockResolvedValue({
      emailLogs: {
        edges: [
          {
            node: {
              id: "emlmsg_1",
              profileId: "emlprof_1",
              purpose: "Billing",
              provider: "Resend",
              providerMessageId: "",
              status: "Delivered",
              attempts: 1,
              fromEmail: "billing@acme.test",
              fromName: "Acme",
              toRecipients: ["ap@customer.test"],
              ccRecipients: null,
              bccRecipients: null,
              subject: "Invoice 1001",
              lastError: "",
              sentAt: 1_780_000_000,
              deliveredAt: 1_780_000_010,
              failedAt: 0,
              createdAt: 1_780_000_000,
              updatedAt: 1_780_000_010,
            },
          },
        ],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    });
    const logs = await new EmailService().listLogs();

    expect(sent()).toEqual(
      expect.objectContaining({ operationName: "EmailLogs", variables: { input: { first: 20 } } }),
    );
    expect(logs.map((log) => log.subject)).toEqual(["Invoice 1001"]);
  });

  it("sends each purpose's profile when the assignments are saved", async () => {
    requestGraphQL.mockResolvedValue({ updateEmailProfileAssignments: [] });

    await new EmailService().updateAssignments([
      { id: "emlassn_1", purpose: "Billing", profileId: "emlprof_1" },
    ]);

    expect(sent().variables).toEqual({ input: [{ purpose: "Billing", profileId: "emlprof_1" }] });
  });

  it("creates a profile from the form's values, leaving a blank reply-to unset", async () => {
    requestGraphQL.mockResolvedValue({ createEmailProfile: { id: "emlprof_1", version: 1 } });

    await new EmailService().createProfile({
      name: "Billing",
      description: null,
      senderName: "Acme Billing",
      senderEmail: "billing@acme.test",
      replyToEmail: "",
      provider: "Postmark",
      status: "Active",
    });

    expect(sent().variables).toEqual({
      input: {
        name: "Billing",
        description: undefined,
        senderName: "Acme Billing",
        senderEmail: "billing@acme.test",
        replyToEmail: undefined,
        provider: "Postmark",
        status: "Active",
      },
    });
  });
});
