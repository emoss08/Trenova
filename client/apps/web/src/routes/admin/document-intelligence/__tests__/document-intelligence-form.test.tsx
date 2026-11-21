import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { DocumentControl, DocumentExtractionStatus } from "@/types/document-control";
import DocumentIntelligenceForm from "../_components/document-intelligence-form";
import { savedDocumentControl } from "./document-intelligence-fixture";

const READABLE: DocumentExtractionStatus = {
  modelProviderAvailable: true,
  modelProviderReason: "available",
  converterAvailable: true,
  extractionAvailable: true,
};

const served = vi.hoisted(() => ({
  control: null as unknown,
  status: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    get: Object.assign(
      () => ({ queryKey: ["documentControl", "get"], queryFn: async () => served.control }),
      { _def: ["documentControl", "get"] },
    ),
    extractionStatus: Object.assign(
      () => ({
        queryKey: ["documentControl", "extraction-status"],
        queryFn: async () => served.status,
      }),
      { _def: ["documentControl", "extraction-status"] },
    ),
  },
}));

// The save records what was sent and hands a refusal to the form, as the real mutation does.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({ form }: { form: unknown }) => ({
    mutateAsync: async (values: unknown) => {
      served.sent.push(values);
      if (!served.refusal) return;
      const { handleMutationError } = await vi.importActual<
        typeof import("@/hooks/use-api-mutation")
      >("@/hooks/use-api-mutation");
      handleMutationError({ error: served.refusal, form: form as never });
      throw served.refusal;
    },
  }),
}));

function renderForm(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <DocumentIntelligenceForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

function reset() {
  served.control = savedDocumentControl;
  served.status = READABLE;
  served.refusal = null;
  served.sent = [];
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  reset();
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

reset();

const READING_OFF =
  "Uploaded documents are not read, so these settings do nothing until reading is turned on.";

describe("document extraction tabs", () => {
  it("offers reading, filing and shipment drafts", async () => {
    renderForm();
    await screen.findByText("Read uploaded documents");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Reading",
      "Filing",
      "Shipment drafts",
    ]);
  });
});

describe("rows that wait on another", () => {
  it.each([
    ["Read uploaded documents", "Read scanned pages", ""],
    ["Read uploaded documents", "Search by content", ""],
    ["Recognize document kinds", "Ask AI when unsure", "?tab=filing"],
    ["File under a document type", "Add missing types", "?tab=filing"],
    ["Draft shipments", "Draft from", "?tab=drafts"],
  ])("opens %s's %s row with it", async (parent, child, search) => {
    const user = userEvent.setup();

    renderForm(search);
    await screen.findByText("Read uploaded documents");
    expect(screen.queryByText(child)).toBeNull();

    await user.click(screen.getByRole("switch", { name: new RegExp(parent) }));

    expect(await screen.findByText(child)).toBeInTheDocument();
  });
});

describe("while reading is off", () => {
  it.each(["filing", "drafts"])(
    "says on the %s tab that its settings wait on reading, and opens Reading",
    async (tab) => {
      const user = userEvent.setup();

      renderForm(`?tab=${tab}`);
      const panel = await screen.findByRole("tabpanel");
      expect(within(panel).getByText(READING_OFF)).toBeInTheDocument();

      await user.click(within(panel).getByRole("button", { name: "Open Reading" }));

      expect(await screen.findByRole("tab", { name: "Reading", selected: true })).toBeVisible();
    },
  );

  it("says nothing once reading is on", async () => {
    served.control = {
      ...savedDocumentControl,
      enableDocumentIntelligence: true,
    } satisfies DocumentControl;

    renderForm("?tab=filing");
    await screen.findByText("Recognize document kinds");

    expect(screen.queryByText(READING_OFF)).toBeNull();
  });
});

describe("when documents cannot be read", () => {
  it("names the missing document converter on Reading", async () => {
    served.status = { ...READABLE, converterAvailable: false, extractionAvailable: false };

    renderForm();

    expect(await screen.findByText("Extraction is not set up on this server")).toBeInTheDocument();
    expect(
      screen.getByText("Ask an administrator to configure the document converter."),
    ).toBeInTheDocument();
  });

  it("names the missing model provider when the converter is configured", async () => {
    served.status = {
      ...READABLE,
      modelProviderAvailable: false,
      modelProviderReason: "no_model_provider",
      extractionAvailable: false,
    };

    renderForm();

    expect(await screen.findByText("Extraction needs Nova")).toBeInTheDocument();
  });

  it("shows no alert while documents can be read", async () => {
    renderForm();
    await screen.findByText("Read uploaded documents");

    expect(screen.queryByText("Extraction is not set up on this server")).toBeNull();
    expect(screen.queryByText("Extraction needs Nova")).toBeNull();
  });
});

describe("saving", () => {
  it("sends the whole record, the capture settings as they were read", async () => {
    const user = userEvent.setup();

    renderForm();
    await user.click(await screen.findByRole("switch", { name: /Read uploaded documents/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(served.sent).toEqual([{ ...savedDocumentControl, enableDocumentIntelligence: true }]);
  });

  it("sends the records chosen to draft from", async () => {
    served.control = {
      ...savedDocumentControl,
      enableShipmentDraftExtraction: true,
    } satisfies DocumentControl;
    const user = userEvent.setup();

    renderForm("?tab=drafts");
    await user.click(await screen.findByRole("checkbox", { name: "Trailers" }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(served.sent).toEqual([
      {
        ...savedDocumentControl,
        enableShipmentDraftExtraction: true,
        shipmentDraftAllowedResources: ["shipment", "trailer"],
      },
    ]);
  });

  it("refuses drafting from no record, opening Shipment drafts", async () => {
    served.control = {
      ...savedDocumentControl,
      enableShipmentDraftExtraction: true,
    } satisfies DocumentControl;
    const user = userEvent.setup();

    renderForm("?tab=drafts");
    await user.click(await screen.findByRole("checkbox", { name: "Shipments" }));
    await user.click(screen.getByRole("tab", { name: "Reading" }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const drafts = await screen.findByRole("tab", { name: /Shipment drafts/, selected: true });
    expect(within(drafts).getByText("Has fields to fix")).toBeInTheDocument();
    expect(screen.getByText("Choose at least one kind of record to draft from")).toBeVisible();
    expect(served.sent).toEqual([]);
  });

  it("opens and marks Filing when the server refuses one of its settings", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Classification is not available",
      instance: "/document-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["enableAutoClassification"],
          message: "Classification is not available",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm();
    await user.click(await screen.findByRole("switch", { name: /Read uploaded documents/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const filing = await screen.findByRole("tab", { name: /Filing/, selected: true });
    expect(within(filing).getByText("Has fields to fix")).toBeInTheDocument();
    expect(screen.getAllByText("Classification is not available").at(-1)).toBeVisible();
  });
});
