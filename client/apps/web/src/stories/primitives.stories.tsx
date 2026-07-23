import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@trenova/shared/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { MoreHorizontalIcon, SaveIcon, SearchIcon, SettingsIcon, TrashIcon } from "lucide-react";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";

const selectItems = [
  { label: "All statuses", value: "all" },
  { label: "Ready", value: "ready" },
  { label: "Needs review", value: "review" },
];

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="grid gap-3">
      <h2 className="text-base font-semibold">{title}</h2>
      <div className="flex flex-wrap items-center gap-3 rounded-md border bg-background p-4">
        {children}
      </div>
    </section>
  );
}

function StatefulControls() {
  const [checked, setChecked] = useState(false);
  const [enabled, setEnabled] = useState(true);
  const [date, setDate] = useState<Date | undefined>(new Date("2026-05-25T12:00:00"));
  const [status, setStatus] = useState("all");

  return (
    <>
      <Section title="Inputs">
        <Input className="max-w-64" placeholder="Search shipments" leftElement={<SearchIcon />} />
        <Textarea className="max-w-80" placeholder="Add a carrier note" minRows={3} />
        <Select
          value={status}
          items={selectItems}
          onValueChange={(value) => setStatus(value ?? "all")}
        >
          <SelectTrigger className="w-48">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectLabel>Status</SelectLabel>
              {selectItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </Section>

      <Section title="Buttons and Badges">
        <Button>
          <SaveIcon data-icon="inline-start" />
          Save
        </Button>
        <Button variant="outline">Cancel</Button>
        <Button variant="destructive">
          <TrashIcon data-icon="inline-start" />
          Delete
        </Button>
        <Button isLoading loadingText="Saving" />
        <Badge variant="active">Active</Badge>
        <Badge variant="warning">Delayed</Badge>
        <Badge variant="info">In review</Badge>
      </Section>

      <Section title="Toggles">
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={checked} onCheckedChange={(value) => setChecked(value === true)} />
          Require receipt
        </label>
        <label className="flex items-center gap-2 text-sm">
          <Switch checked={enabled} onCheckedChange={setEnabled} />
          Auto-dispatch
        </label>
      </Section>

      <Section title="Tabs and Calendar">
        <Tabs defaultValue="overview" className="w-full max-w-xl">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="exceptions">Exceptions</TabsTrigger>
          </TabsList>
          <TabsContent value="overview" className="rounded-md border p-3 text-sm">
            Tender status, mileage, and appointment visibility.
          </TabsContent>
          <TabsContent value="exceptions" className="rounded-md border p-3 text-sm">
            OS&D, detention, and billing exception review.
          </TabsContent>
        </Tabs>
        <Calendar mode="single" selected={date} onSelect={setDate} />
      </Section>
    </>
  );
}

function OverlayControls() {
  return (
    <Section title="Overlays">
      <Popover>
        <PopoverTrigger render={<Button variant="outline">Open popover</Button>} />
        <PopoverContent className="w-72">
          <PopoverHeader>
            <PopoverTitle>Shipment filters</PopoverTitle>
            <PopoverDescription>Filter the board without changing saved views.</PopoverDescription>
          </PopoverHeader>
          <Input placeholder="Filter by carrier" />
        </PopoverContent>
      </Popover>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="outline">
              <MoreHorizontalIcon data-icon="inline-start" />
              Actions
            </Button>
          }
        />
        <DropdownMenuContent className="w-52">
          <DropdownMenuGroup>
            <DropdownMenuLabel>Shipment</DropdownMenuLabel>
            <DropdownMenuItem title="Open" description="View shipment details" />
            <DropdownMenuItem
              title="Configure"
              startContent={<SettingsIcon />}
              endContent={<DropdownMenuShortcut>⌘K</DropdownMenuShortcut>}
            />
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem color="danger" title="Void shipment" />
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog>
        <DialogTrigger render={<Button variant="outline">Open dialog</Button>} />
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm tender update</DialogTitle>
            <DialogDescription>
              This changes the carrier tender state for the selected shipment.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline">Cancel</Button>
            <Button>Confirm</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Tooltip>
        <TooltipTrigger render={<Button variant="ghost">Hover target</Button>} />
        <TooltipContent>Tooltip content</TooltipContent>
      </Tooltip>
    </Section>
  );
}

const meta = {
  title: "UI/Primitives",
  parameters: {
    docs: {
      description: {
        component:
          "Reusable Trenova UI primitives for fast visual debugging of controls, overlays, loading states, and composition.",
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Gallery: Story = {
  render: () => (
    <div className="grid gap-6">
      <StatefulControls />
      <OverlayControls />
    </div>
  ),
};

export const LoadingStates: Story = {
  render: () => (
    <div className="grid gap-6">
      <Section title="Skeletons">
        <div className="grid w-full max-w-md gap-2">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      </Section>
      <Section title="Spinners">
        <Spinner />
        <Spinner variant="circle" />
        <Spinner variant="circle-filled" />
        <Spinner variant="ellipsis" />
        <Spinner variant="ring" />
        <Spinner variant="bars" />
      </Section>
    </div>
  ),
};

export const OverlayInteraction: Story = {
  render: () => <OverlayControls />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await userEvent.click(canvas.getByRole("button", { name: /open popover/i }));
    await expect(await within(document.body).findByText("Shipment filters")).toBeInTheDocument();

    await userEvent.click(canvas.getByRole("button", { name: /actions/i }));
    await expect(await within(document.body).findByText("Void shipment")).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    await userEvent.click(canvas.getByRole("button", { name: /open dialog/i }));
    await expect(
      await within(document.body).findByText("Confirm tender update"),
    ).toBeInTheDocument();
  },
};
