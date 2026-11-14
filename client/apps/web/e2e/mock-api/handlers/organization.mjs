// The organization settings page's operations (`/admin/organization-settings`): the settings
// read and save, and the branding routes. An uploaded logo or icon is kept in the state and
// served back as a data URL, so the page and the sidebar show exactly what was uploaded.
import { ORG } from "../fixtures/session.mjs";

const TEXAS = { id: "st_tx", name: "Texas", abbreviation: "TX", countryName: "United States" };

function organizationOf(state) {
  state.organization ??= {
    settings: {
      id: ORG.id,
      version: 1,
      createdAt: 1700000000,
      updatedAt: 1700000000,
      bucketName: "trenova-mock",
      businessUnitId: "bu_mock",
      loginSlug: "trenova-logistics",
      name: ORG.name,
      scacCode: "TRNV",
      dotNumber: "1234567",
      logoUrl: "",
      iconUrl: "",
      addressLine1: "1200 Main Street",
      addressLine2: "",
      city: "Dallas",
      stateId: TEXAS.id,
      postalCode: "75201",
      timezone: "America/Chicago",
      taxId: "",
      brokerageEnabled: true,
      assetOperationsEnabled: true,
      state: TEXAS,
    },
    images: { logo: null, icon: null },
  };
  return state.organization;
}

const FIELD = { logo: "logoUrl", icon: "iconUrl" };

const imageOf = (path) => /\/branding\/(logo|icon)\/?$/.exec(path)[1];

const dataUrl = (image) =>
  image ? `data:${image.type};base64,${image.content.toString("base64")}` : "";

/** The organization as the sidebar's switcher lists it. */
export function currentOrganization(state) {
  const { settings } = organizationOf(state);
  return {
    id: settings.id,
    name: settings.name,
    city: settings.city,
    state: TEXAS.abbreviation,
    logoUrl: settings.logoUrl || null,
    iconUrl: settings.iconUrl || null,
    isDefault: true,
    isCurrent: true,
  };
}

export const ORGANIZATION_HANDLERS = {
  OrganizationSettings: (state) => ({ organization: organizationOf(state).settings }),

  UpdateOrganizationSettings: (state, { input }) => {
    const held = organizationOf(state);
    held.settings = {
      ...held.settings,
      ...Object.fromEntries(Object.entries(input).filter(([, value]) => value !== null)),
      version: held.settings.version + 1,
    };
    return { updateOrganization: held.settings };
  },
};

export const ORGANIZATION_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/organizations\/[^/]+\/branding\/?$/,
    (state) => {
      const { images } = organizationOf(state);
      return { logoUrl: dataUrl(images.logo), iconUrl: dataUrl(images.icon) };
    },
  ],
  [
    "POST",
    /^\/api\/v1\/organizations\/[^/]+\/branding\/(logo|icon)\/?$/,
    (state, path, body) => {
      const held = organizationOf(state);
      const image = imageOf(path);
      held.images[image] = { type: body.file.type, content: body.file.content };
      held.settings = {
        ...held.settings,
        [FIELD[image]]: `organization/${image}/${body.file.name}`,
        version: held.settings.version + 1,
      };
      return held.settings;
    },
  ],
  [
    "DELETE",
    /^\/api\/v1\/organizations\/[^/]+\/branding\/(logo|icon)\/?$/,
    (state, path) => {
      const held = organizationOf(state);
      const image = imageOf(path);
      held.images[image] = null;
      held.settings = {
        ...held.settings,
        [FIELD[image]]: "",
        version: held.settings.version + 1,
      };
      return held.settings;
    },
  ],
];
