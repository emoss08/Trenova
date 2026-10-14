import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { useController, type Control } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

type PickerProps = { control: Control; name: string; label: string };

const mocks = vi.hoisted(() => ({
  requestGraphQL: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError },
}));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: (...args: unknown[]) => mocks.requestGraphQL(...args),
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({ control, name, label }: PickerProps) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/autocomplete-fields", () => ({
  RoleAutocompleteField: ({ control, name, label }: PickerProps) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={((field.value as string[]) ?? []).join(",")}
        onChange={(event) =>
          field.onChange(event.target.value ? event.target.value.split(",") : [])
        }
      />
    );
  },
}));

vi.mock("../user-roles-editor", () => ({ UserRolesEditor: () => null }));

vi.mock("@/services/api", () => ({
  apiService: {
    userService: {
      getUserOrganizations: vi.fn(async () => []),
      listOrganizationMemberships: vi.fn(async () => []),
    },
  },
}));

const { UserPanel } = await import("../user-panel");

const detail = {
  id: "usr_1",
  businessUnitId: "bu_1",
  currentOrganizationId: "org_1",
  status: "Active",
  name: "Dana Whitfield",
  username: "dwhitfield",
  emailAddress: "dana@example.com",
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "America/Chicago",
  timeFormat: "TwentyFourHour",
  locale: "es",
  isLocked: true,
  mustChangePassword: false,
  jobTitle: "Dispatcher",
  phoneNumber: "",
  avatarStyle: "dusk",
  availabilityStatus: "away",
  version: 4,
  createdAt: 1_760_000_000,
  updatedAt: 1_760_000_100,
  lastLoginAt: null,
  passwordChangedAt: null,
};

function renderPanel(mode: "create" | "edit") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  return render(
    <NuqsTestingAdapter>
      <QueryClientProvider client={queryClient}>
        <UserPanel
          open
          mode={mode}
          row={mode === "edit" ? ({ id: "usr_1" } as never) : null}
          onOpenChange={() => {}}
        />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

function inputFor(name: string): HTMLInputElement {
  const element = document.getElementById(`input-${name}`);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`input-${name} not rendered`);
  }
  return element;
}

function callsTo(operationName: string) {
  return mocks.requestGraphQL.mock.calls
    .map(([params]) => params as { operationName: string; variables: unknown })
    .filter((params) => params.operationName === operationName);
}

async function fillCreateForm() {
  const user = userEvent.setup();
  await user.type(await waitFor(() => inputFor("name")), "Morgan Ellis");
  await user.type(inputFor("username"), "mellis");
  await user.type(inputFor("emailAddress"), "morgan@example.com");
  await user.type(screen.getByLabelText("Timezone"), "America/Denver");
  await user.type(screen.getByLabelText("Roles"), "rol_dispatch,rol_billing");
  return user;
}

function submit(formId: string) {
  const form = document.getElementById(formId);
  if (!form) {
    throw new Error(`${formId} not rendered`);
  }
  fireEvent.submit(form);
}

beforeEach(() => {
  mocks.requestGraphQL.mockReset();
  mocks.toastSuccess.mockReset();
  mocks.toastError.mockReset();
});

afterEach(() => {
  cleanup();
});

describe("UserPanel create", () => {
  it("creates the user through GraphQL with role ids and names the invited address", async () => {
    mocks.requestGraphQL.mockImplementation(async ({ operationName, variables }) => {
      if (operationName === "CreateUser") {
        return {
          createUser: {
            ...detail,
            id: "usr_2",
            name: variables.input.name,
            emailAddress: variables.input.emailAddress,
            status: "Active",
          },
        };
      }
      throw new Error(`unexpected ${operationName}`);
    });
    renderPanel("create");

    expect(screen.queryByLabelText(/password/i)).not.toBeInTheDocument();
    expect(screen.getByText(/they set their own password/i)).toBeInTheDocument();

    await fillCreateForm();
    submit("panel-create-form");

    await waitFor(() => expect(callsTo("CreateUser")).toHaveLength(1));
    expect(callsTo("CreateUser")[0].variables).toEqual({
      input: {
        status: "Active",
        name: "Morgan Ellis",
        username: "mellis",
        emailAddress: "morgan@example.com",
        timezone: "America/Denver",
        timeFormat: "TwelveHour",
        locale: "en",
        roleIds: ["rol_dispatch", "rol_billing"],
      },
    });
    await waitFor(() =>
      expect(mocks.toastSuccess).toHaveBeenCalledWith("User created", {
        description: "An invitation to set up their account was sent to morgan@example.com.",
      }),
    );
  });

  it("shows the server's duplicate email error on the email field", async () => {
    mocks.requestGraphQL.mockImplementation(async () => {
      throw new GraphQLRequestError({
        kind: "graphql",
        message: "validation failed",
        status: 200,
        graphQLErrors: [
          {
            message: "validation failed",
            type: "https://trenova.app/problems/validation-error",
            extensions: { type: "https://trenova.app/problems/validation-error" },
            errors: [
              {
                field: "emailAddress",
                message: "A user with this email address already exists",
                code: "DUPLICATE",
              },
            ],
          },
        ],
      });
    });
    renderPanel("create");

    await fillCreateForm();
    submit("panel-create-form");

    expect(
      await screen.findByText("A user with this email address already exists"),
    ).toBeInTheDocument();
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
  });
});

describe("UserPanel edit", () => {
  it("shows a skeleton until the user is loaded", async () => {
    mocks.requestGraphQL.mockImplementation(() => new Promise(() => {}));
    renderPanel("edit");

    await waitFor(() => expect(document.querySelector('[aria-busy="true"]')).not.toBeNull());
    expect(document.getElementById("input-name")).toBeNull();
    expect(callsTo("UserDetail")[0].variables).toEqual({ id: "usr_1" });
  });

  it("loads the user by id rather than from the table row", async () => {
    mocks.requestGraphQL.mockImplementation(async ({ operationName }) =>
      operationName === "UserDetail" ? { user: detail } : {},
    );
    renderPanel("edit");

    await waitFor(() => expect(inputFor("name")).toHaveValue("Dana Whitfield"));
    expect(screen.getByLabelText("Time format")).toHaveValue("24-hour");
    expect(screen.getByLabelText("Language")).toHaveValue("es");
    expect(inputFor("jobTitle")).toHaveValue("Dispatcher");
    expect(screen.getByRole("button", { name: /resend invitation/i })).toBeInTheDocument();
  });

  it("patches only the toggle that was turned off", async () => {
    mocks.requestGraphQL.mockImplementation(async ({ operationName, variables }) => {
      if (operationName === "UserDetail") {
        return { user: detail };
      }
      if (operationName === "PatchUser") {
        return { patchUser: { ...detail, ...variables.input, version: 5 } };
      }
      throw new Error(`unexpected ${operationName}`);
    });
    const user = userEvent.setup();
    renderPanel("edit");

    await waitFor(() => expect(inputFor("name")).toHaveValue("Dana Whitfield"));
    const locked = screen.getByRole("switch", { name: /account locked/i });
    await waitFor(() => expect(locked).toHaveAttribute("aria-checked", "true"));
    await user.click(locked);
    expect(locked).toHaveAttribute("aria-checked", "false");
    submit("panel-edit-form");

    await waitFor(() => expect(callsTo("PatchUser")).toHaveLength(1));
    expect(callsTo("PatchUser")[0].variables).toEqual({
      id: "usr_1",
      input: { isLocked: false },
    });
  });
});
