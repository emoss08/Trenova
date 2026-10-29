import { describe, expect, it } from "vitest";
import { goEnumValues } from "@/test/go-source";
import {
  aiProviderPlatformAuthSchema,
  aiProviderPlatformSchema,
  saveAIProviderRequestSchema,
  type AIProviderPlatformDescriptor,
  type AIProviderPreset,
} from "@/types/ai-provider";
import {
  credentialRequired,
  EMPTY_PLATFORM_SETTINGS,
  offeredAuths,
  platformDescriptor,
  platformPayload,
  platformSettingsDraft,
  platformSettingsSummary,
  settingRequired,
  type PlatformSettingsDraft,
} from "../platform-model";
import {
  draftTestInput,
  editorKeyRequired,
  editorProblem,
  editorValuesFromPreset,
  toSaveRequest,
  type PlatformCatalog,
  type ProviderEditorValues,
} from "../provider-editor-model";

const PLATFORM_FILE = "services/tms/internal/modules/agent/domain/aiplatform/platform.go";

/**
 * The catalog's platforms as the server builds them (aiproviderhandler.platformDescriptors):
 * each platform's settings, the ways it signs in in the server's order, the settings each
 * way adds, and only the server's own credentials needing nothing stored.
 */
const PLATFORMS: AIProviderPlatformDescriptor[] = [
  {
    platform: "Direct",
    settings: [],
    auths: [{ auth: "ApiKey", requiresCredential: true, settings: [] }],
    listsModels: true,
  },
  {
    platform: "Bedrock",
    settings: ["region"],
    auths: [
      { auth: "AccessKey", requiresCredential: true, settings: ["accessKeyId"] },
      { auth: "ApiKey", requiresCredential: true, settings: [] },
      { auth: "Server", requiresCredential: false, settings: [] },
    ],
    listsModels: false,
  },
  {
    platform: "VertexAI",
    settings: ["projectId", "region"],
    auths: [
      { auth: "ServiceAccount", requiresCredential: true, settings: [] },
      { auth: "Server", requiresCredential: false, settings: [] },
    ],
    listsModels: false,
  },
  {
    platform: "ClaudePlatformAWS",
    settings: ["region", "workspaceId"],
    auths: [
      { auth: "AccessKey", requiresCredential: true, settings: ["accessKeyId"] },
      { auth: "ApiKey", requiresCredential: true, settings: [] },
      { auth: "Server", requiresCredential: false, settings: [] },
    ],
    listsModels: true,
  },
  {
    platform: "ClaudePlatformGoogleCloud",
    settings: ["projectId", "region", "workspaceId"],
    auths: [
      { auth: "ServiceAccount", requiresCredential: true, settings: [] },
      { auth: "Server", requiresCredential: false, settings: [] },
    ],
    listsModels: true,
  },
  {
    platform: "Foundry",
    settings: ["resource"],
    auths: [
      { auth: "ApiKey", requiresCredential: true, settings: [] },
      { auth: "EntraID", requiresCredential: true, settings: ["tenantId", "clientId"] },
    ],
    listsModels: false,
  },
];

const CATALOG: PlatformCatalog = { platforms: PLATFORMS, serverCloudCredentials: false };
const SERVER_CATALOG: PlatformCatalog = { platforms: PLATFORMS, serverCloudCredentials: true };

const BEDROCK_PRESET: AIProviderPreset = {
  key: "bedrock-claude",
  label: "Claude on Amazon Bedrock",
  kind: "AnthropicMessages",
  platform: "Bedrock",
  baseUrl: "",
  structuredOutputMode: "JSONSchema",
  allowPrivateNetwork: false,
  requiresApiKey: true,
  selfHosted: false,
  exampleModel: "anthropic.claude-opus-5-5",
  notes: "",
  domain: "aws.amazon.com",
  tasks: [],
  embeddingDimensions: 0,
};

/** Every setting typed, with stray spaces, so a test can see which ones are sent. */
const EVERY_SETTING: PlatformSettingsDraft = {
  auth: "AccessKey",
  region: " us-east-1 ",
  projectId: "freight-ops",
  workspaceId: "wrkspc_01",
  accessKeyId: " AKIAIOSFODNN7EXAMPLE ",
  resource: "my-foundry",
  tenantId: "contoso.onmicrosoft.com",
  clientId: "00000000-0000-0000-0000-000000000000",
};

function bedrock(overrides: Partial<ProviderEditorValues> = {}): ProviderEditorValues {
  return {
    ...editorValuesFromPreset(BEDROCK_PRESET, [], CATALOG),
    name: "Bedrock",
    platformSettings: { ...EVERY_SETTING },
    apiKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    ...overrides,
  };
}

function descriptor(platform: AIProviderPlatformDescriptor["platform"]) {
  return platformDescriptor(PLATFORMS, platform);
}

describe("platform enums mirror the server", () => {
  it("offers every platform and way of signing in the server accepts, and no other", () => {
    expect([...aiProviderPlatformSchema.options].sort()).toEqual(
      goEnumValues({ file: PLATFORM_FILE, typeName: "Platform" }).sort(),
    );
    expect([...aiProviderPlatformAuthSchema.options].sort()).toEqual(
      goEnumValues({ file: PLATFORM_FILE, typeName: "PlatformAuth" }).sort(),
    );
  });
});

/**
 * A setting typed for one platform or way of signing in is never sent to another: the
 * server keys a stored credential to the settings, so a stray one would make a save
 * look like a move to a new target.
 */
describe("platformPayload", () => {
  it("sends only what the platform and its way of signing in take, trimmed", () => {
    const payload = platformPayload({
      kind: "AnthropicMessages",
      platform: "Bedrock",
      settings: EVERY_SETTING,
      descriptor: descriptor("Bedrock"),
      serverAllowed: false,
    });

    expect(payload).toEqual({
      platform: "Bedrock",
      platformSettings: {
        auth: "AccessKey",
        region: "us-east-1",
        projectId: "",
        workspaceId: "",
        accessKeyId: "AKIAIOSFODNN7EXAMPLE",
        resource: "",
        tenantId: "",
        clientId: "",
      },
    });
  });

  it("drops the access key ID once the platform signs in another way", () => {
    const payload = platformPayload({
      kind: "AnthropicMessages",
      platform: "Bedrock",
      settings: { ...EVERY_SETTING, auth: "ApiKey" },
      descriptor: descriptor("Bedrock"),
      serverAllowed: false,
    });

    expect(payload.platformSettings).toMatchObject({ auth: "ApiKey", accessKeyId: "" });
  });

  it("sends no platform for a direct provider or for another kind of API", () => {
    expect(
      platformPayload({
        kind: "AnthropicMessages",
        platform: "Direct",
        settings: EVERY_SETTING,
        descriptor: descriptor("Direct"),
        serverAllowed: false,
      }),
    ).toEqual({ platform: "Direct", platformSettings: null });
    expect(
      platformPayload({
        kind: "OpenAIChat",
        platform: "Bedrock",
        settings: EVERY_SETTING,
        descriptor: descriptor("Bedrock"),
        serverAllowed: false,
      }),
    ).toEqual({ platform: "Direct", platformSettings: null });
  });

  it("falls back to the platform's usual way when the one held does not belong to it", () => {
    const payload = platformPayload({
      kind: "AnthropicMessages",
      platform: "VertexAI",
      settings: EVERY_SETTING,
      descriptor: descriptor("VertexAI"),
      serverAllowed: true,
    });

    expect(payload.platformSettings).toMatchObject({
      auth: "ServiceAccount",
      projectId: "freight-ops",
      region: "us-east-1",
      accessKeyId: "",
    });
  });
});

describe("offeredAuths", () => {
  it("offers the server's own credentials only where the server allows them", () => {
    expect(offeredAuths(descriptor("Bedrock"), false, "")).toEqual(["AccessKey", "ApiKey"]);
    expect(offeredAuths(descriptor("Bedrock"), true, "")).toEqual([
      "AccessKey",
      "ApiKey",
      "Server",
    ]);
  });

  it("keeps offering them to a provider saved with them, so it opens as it was saved", () => {
    expect(offeredAuths(descriptor("Bedrock"), false, "Server")).toContain("Server");
  });

  it("offers nothing for a platform the catalog does not describe", () => {
    expect(offeredAuths(undefined, true, "")).toEqual([]);
  });
});

describe("credentialRequired", () => {
  const base = { kind: "AnthropicMessages" as const, kindRequiresKey: false };

  it("asks for nothing stored when the server signs in", () => {
    expect(
      credentialRequired({
        ...base,
        platform: "Bedrock",
        auth: "Server",
        descriptor: descriptor("Bedrock"),
      }),
    ).toBe(false);
  });

  it("asks for the credential every other way of signing in uses, whatever the kind says", () => {
    expect(
      credentialRequired({
        ...base,
        platform: "Foundry",
        auth: "EntraID",
        descriptor: descriptor("Foundry"),
      }),
    ).toBe(true);
  });

  it("leaves a direct provider to its kind", () => {
    expect(
      credentialRequired({ ...base, platform: "Direct", auth: "", descriptor: undefined }),
    ).toBe(false);
    expect(
      credentialRequired({
        ...base,
        kindRequiresKey: true,
        platform: "Direct",
        auth: "",
        descriptor: undefined,
      }),
    ).toBe(true);
  });
});

describe("settingRequired", () => {
  it("needs a Foundry resource only when no base URL stands in for it", () => {
    expect(settingRequired("Foundry", "resource", "")).toBe(true);
    expect(settingRequired("Foundry", "resource", "https://gateway.example.com/anthropic")).toBe(
      false,
    );
  });

  it("lets Claude Platform on Google Cloud default its region, and nothing else", () => {
    expect(settingRequired("ClaudePlatformGoogleCloud", "region", "")).toBe(false);
    expect(settingRequired("ClaudePlatformGoogleCloud", "workspaceId", "")).toBe(true);
    expect(settingRequired("VertexAI", "region", "")).toBe(true);
  });
});

describe("editorValuesFromPreset on a platform", () => {
  it("starts with the platform's first way of signing in the server offers", () => {
    const values = editorValuesFromPreset(BEDROCK_PRESET, [], CATALOG);
    expect(values.platform).toBe("Bedrock");
    expect(values.platformSettings).toEqual({ ...EMPTY_PLATFORM_SETTINGS, auth: "AccessKey" });
  });

  it("starts a preset of another kind directly, whatever platform it names", () => {
    const values = editorValuesFromPreset(
      { ...BEDROCK_PRESET, kind: "OpenAIChat" },
      [],
      SERVER_CATALOG,
    );
    expect(values.platform).toBe("Direct");
    expect(values.platformSettings.auth).toBe("");
  });
});

describe("platformSettingsDraft", () => {
  it("opens a provider saved without settings as blank", () => {
    expect(platformSettingsDraft(null)).toEqual(EMPTY_PLATFORM_SETTINGS);
  });
});

describe("editorProblem on a platform", () => {
  const context = { create: true, keyRequired: true, hasStoredKey: false, catalog: CATALOG };

  it("names the first setting the platform still needs", () => {
    expect(
      editorProblem(bedrock({ platformSettings: { ...EVERY_SETTING, region: "  " } }), context),
    ).toEqual({
      field: "platformSettings.region",
      message: "Add the region",
    });
    expect(
      editorProblem(bedrock({ platformSettings: { ...EVERY_SETTING, accessKeyId: "" } }), context)
        ?.field,
    ).toBe("platformSettings.accessKeyId");
  });

  it("does not ask for a setting the chosen way of signing in does not use", () => {
    expect(
      editorProblem(
        bedrock({ platformSettings: { ...EVERY_SETTING, auth: "ApiKey", accessKeyId: "" } }),
        context,
      ),
    ).toBeNull();
  });

  it("refuses the server's credentials where the server does not allow them", () => {
    const server = bedrock({ apiKey: "", platformSettings: { ...EVERY_SETTING, auth: "Server" } });
    expect(editorProblem(server, { ...context, keyRequired: false })?.field).toBe(
      "platformSettings.auth",
    );
    expect(
      editorProblem(server, { ...context, keyRequired: false, catalog: SERVER_CATALOG }),
    ).toBeNull();
  });
});

describe("editorKeyRequired", () => {
  it("needs no key when the server signs in, even for a preset that asks for one", () => {
    const server = bedrock({ platformSettings: { ...EVERY_SETTING, auth: "Server" } });
    expect(editorKeyRequired(server, { catalog: SERVER_CATALOG, kindRequiresKey: true })).toBe(
      false,
    );
    expect(editorKeyRequired(bedrock(), { catalog: CATALOG, kindRequiresKey: false })).toBe(true);
  });
});

describe("toSaveRequest on a platform", () => {
  it("saves the platform and only its settings, and the server accepts the shape", () => {
    const request = toSaveRequest(bedrock(), { editing: false, enable: false, catalog: CATALOG });

    expect(request.platform).toBe("Bedrock");
    expect(request.platformSettings).toMatchObject({
      auth: "AccessKey",
      region: "us-east-1",
      accessKeyId: "AKIAIOSFODNN7EXAMPLE",
      projectId: "",
    });
    expect(saveAIProviderRequestSchema.safeParse(request).success).toBe(true);
  });

  /*
   * The server keeps no key for a provider that signs in as itself, and refuses an edit
   * that leaves a stored key behind for a new way of signing in. Leaving the key out, as a
   * blank key field otherwise means, would make the switch impossible to save.
   */
  it("clears the stored key when the provider switches to the server's credentials", () => {
    const request = toSaveRequest(
      bedrock({
        apiKey: "typed-before-switching",
        keepPreviousKey: true,
        platformSettings: { ...EVERY_SETTING, auth: "Server" },
      }),
      { editing: true, enable: false, catalog: SERVER_CATALOG },
    );

    expect(request.apiKey).toBe("");
    expect(request.keepPreviousKey).toBe(false);
    expect(request.platformSettings).toMatchObject({ auth: "Server", accessKeyId: "" });
  });

  it("drops the platform when the kind moves off the Anthropic protocol", () => {
    const request = toSaveRequest(bedrock({ kind: "OpenAIChat" }), {
      editing: true,
      enable: false,
      catalog: CATALOG,
    });

    expect(request.platform).toBe("Direct");
    expect(request.platformSettings).toBeNull();
  });
});

describe("draftTestInput on a platform", () => {
  it("tests the platform as typed", () => {
    const input = draftTestInput(bedrock(), null, CATALOG);
    expect(input.endpoint).toMatchObject({
      platform: "Bedrock",
      platformSettings: { auth: "AccessKey", region: "us-east-1" },
      apiKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    });
  });

  it("sends no key when the server signs in, even one left in the field", () => {
    const input = draftTestInput(
      bedrock({ platformSettings: { ...EVERY_SETTING, auth: "Server" } }),
      "aiprv_1",
      SERVER_CATALOG,
    );
    expect(input.endpoint.apiKey).toBeNull();
  });
});

describe("platformSettingsSummary", () => {
  it("names the way of signing in, then each filled setting", () => {
    expect(
      platformSettingsSummary({
        ...EMPTY_PLATFORM_SETTINGS,
        auth: "AccessKey",
        region: "us-east-1",
        accessKeyId: "AKIAIOSFODNN7EXAMPLE",
      }),
    ).toBe("IAM access key · Region us-east-1 · Access key ID AKIAIOSFODNN7EXAMPLE");
    expect(platformSettingsSummary(null)).toBe("");
  });
});
