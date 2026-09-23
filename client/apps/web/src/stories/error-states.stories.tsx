import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { NotFoundPage } from "@trenova/shared/components/errors/not-found";
import { StatusScreen } from "@trenova/shared/components/errors/status-screen";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { Meta, StoryObj } from "@storybook/react-vite";

const PROBLEM_BASE = "https://trenova.app/problems";

function apiError(status: number, type: string, detail: string, traceId?: string) {
  return new ApiRequestError(status, {
    type: `${PROBLEM_BASE}/${type}`,
    title: type,
    status,
    detail,
    traceId,
  });
}

const SAMPLES: { label: string; error: unknown }[] = [
  { label: "Network", error: new TypeError("Failed to fetch") },
  {
    label: "Stale build",
    error: new TypeError("Failed to fetch dynamically imported module: /assets/page-4f1c.js"),
  },
  { label: "Session", error: apiError(401, "authentication-error", "Session expired") },
  { label: "Permission", error: apiError(403, "authorization-error", "Missing shipment:read") },
  { label: "Not found", error: apiError(404, "resource-not-found", "Shipment not found") },
  { label: "Rate limit", error: apiError(429, "rate-limit-exceeded", "Slow down") },
  { label: "Timeout", error: apiError(504, "request-timeout", "Upstream timed out") },
  {
    label: "Server",
    error: apiError(500, "database-error", "pq: deadlock detected", "01J8Z3QK4M2N"),
  },
  {
    label: "Rejected",
    error: apiError(422, "business-rule-violation", "The shipment is already billed"),
  },
  {
    label: "Crash",
    error: new TypeError("Cannot read properties of undefined (reading 'stops')"),
  },
];

function noop() {}

const meta = {
  title: "UI/Error states",
  parameters: {
    docs: {
      description: {
        component:
          "ErrorState classifies whatever was thrown and says what happened and what to do. NotFoundPage and StatusScreen take the whole window.",
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Section: Story = {
  render: () => (
    <div className="grid gap-4 md:grid-cols-2">
      {SAMPLES.map((sample) => (
        <div key={sample.label} className="border-border bg-card rounded-lg border">
          <ErrorState error={sample.error} onRetry={noop} />
        </div>
      ))}
    </div>
  ),
};

export const Compact: Story = {
  render: () => (
    <div className="grid max-w-xl gap-3">
      {SAMPLES.slice(0, 4).map((sample) => (
        <ErrorState key={sample.label} error={sample.error} onRetry={noop} layout="compact" />
      ))}
    </div>
  ),
};

export const NotFound: Story = {
  parameters: { layout: "fullscreen" },
  render: () => (
    <div className="-m-6">
      <NotFoundPage path="/admin/agent-control4" onGoHome={noop} onGoBack={noop} />
    </div>
  ),
};

export const WholeWindow: Story = {
  parameters: { layout: "fullscreen" },
  render: () => (
    <div className="-m-6">
      <StatusScreen meta="500">
        <ErrorState error={SAMPLES[7].error} layout="screen" />
      </StatusScreen>
    </div>
  ),
};
