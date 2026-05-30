import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { type EmailProfile, type EmailProfileAssignment } from "@/types/email";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Mail, Send, ShieldAlert } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

const purposes = [
  "General",
  "Billing",
  "Reporting",
  "Operations",
  "Authentication",
  "Notifications",
] as const;

const emptyProfile: EmailProfile = {
  name: "",
  description: "",
  senderName: "",
  senderEmail: "",
  replyToEmail: "",
  provider: "Resend",
  status: "Active",
};

const emailProviders = ["Resend", "Postmark"] as const;

export function EmailProfilesPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<EmailProfile>(emptyProfile);
  const [testRecipient, setTestRecipient] = useState("");
  const profilesQuery = useQuery(queries.email.profiles());
  const assignmentsQuery = useQuery(queries.email.assignments());

  const profiles = profilesQuery.data?.results ?? [];
  const assignments = assignmentsQuery.data ?? [];
  const assignmentByPurpose = new Map(
    assignments.map((assignment) => [assignment.purpose, assignment.profileId]),
  );

  const saveProfile = useMutation({
    mutationFn: (profile: EmailProfile) =>
      profile.id
        ? apiService.emailService.updateProfile(profile.id, profile)
        : apiService.emailService.createProfile(profile),
    onSuccess: async () => {
      setDraft(emptyProfile);
      await queryClient.invalidateQueries({ queryKey: queries.email.profiles().queryKey });
      toast.success("Email profile saved");
    },
  });

  const saveAssignments = useMutation({
    mutationFn: (payload: EmailProfileAssignment[]) => apiService.emailService.updateAssignments(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queries.email.assignments().queryKey });
      toast.success("Assignments updated");
    },
  });

  const testSend = useMutation({
    mutationFn: (profile: EmailProfile) =>
      apiService.emailService.testProfile(profile.id ?? "", {
        to: testRecipient,
        subject: "Trenova email profile test",
        text: `This test verifies the selected Trenova email profile can send through ${profile.provider}.`,
        html: `<p>This test verifies the selected Trenova email profile can send through ${profile.provider}.</p>`,
      }),
    onSuccess: () => toast.success("Test email queued"),
  });

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold">Email Profiles</h1>
          <p className="text-sm text-muted-foreground">Sender profiles and purpose routing.</p>
        </div>
        <div className="flex items-center gap-2">
          <input
            className="h-9 w-64 rounded-md border bg-background px-3 text-sm"
            placeholder="test-recipient@example.com"
            value={testRecipient}
            onChange={(event) => setTestRecipient(event.target.value)}
          />
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <section className="overflow-hidden rounded-md border">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/50 text-left text-xs text-muted-foreground uppercase">
              <tr>
                <th className="px-3 py-2">Profile</th>
                <th className="px-3 py-2">Sender</th>
                <th className="px-3 py-2">Provider</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {profiles.map((profile) => (
                <tr key={profile.id} className="border-b last:border-0">
                  <td className="px-3 py-2 font-medium">{profile.name}</td>
                  <td className="px-3 py-2">
                    <div>{profile.senderName}</div>
                    <div className="text-xs text-muted-foreground">{profile.senderEmail}</div>
                  </td>
                  <td className="px-3 py-2">{profile.provider}</td>
                  <td className="px-3 py-2">
                    <span className="inline-flex items-center gap-1 rounded border px-2 py-1 text-xs">
                      <CheckCircle2 className="size-3" />
                      {profile.status}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-2">
                      <button className="rounded border px-2 py-1 text-xs" onClick={() => setDraft(profile)}>
                        Edit
                      </button>
                      <button
                        className="inline-flex items-center gap-1 rounded border px-2 py-1 text-xs"
                        disabled={!testRecipient || !profile.id || testSend.isPending}
                        onClick={() => testSend.mutate(profile)}
                      >
                        <Send className="size-3" />
                        Test
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {profiles.length === 0 && (
                <tr>
                  <td className="px-3 py-8 text-center text-muted-foreground" colSpan={5}>
                    No email profiles configured.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </section>

        <section className="rounded-md border p-3">
          <div className="mb-3 flex items-center gap-2 font-medium">
            <Mail className="size-4" />
            Profile
          </div>
          <div className="space-y-2">
            {(["name", "senderName", "senderEmail", "replyToEmail"] as const).map((field) => (
              <input
                key={field}
                className="h-9 w-full rounded-md border bg-background px-3 text-sm"
                placeholder={field}
                value={draft[field] ?? ""}
                onChange={(event) => setDraft({ ...draft, [field]: event.target.value })}
              />
            ))}
            <textarea
              className="min-h-20 w-full rounded-md border bg-background px-3 py-2 text-sm"
              placeholder="description"
              value={draft.description ?? ""}
              onChange={(event) => setDraft({ ...draft, description: event.target.value })}
            />
            <select
              className="h-9 w-full rounded-md border bg-background px-3 text-sm"
              value={draft.provider}
              onChange={(event) =>
                setDraft({ ...draft, provider: event.target.value as EmailProfile["provider"] })
              }
            >
              {emailProviders.map((provider) => (
                <option key={provider} value={provider}>
                  {provider}
                </option>
              ))}
            </select>
            <select
              className="h-9 w-full rounded-md border bg-background px-3 text-sm"
              value={draft.status}
              onChange={(event) => setDraft({ ...draft, status: event.target.value as EmailProfile["status"] })}
            >
              <option value="Active">Active</option>
              <option value="Inactive">Inactive</option>
            </select>
            <button
              className="h-9 w-full rounded-md bg-primary px-3 text-sm text-primary-foreground"
              disabled={saveProfile.isPending}
              onClick={() => saveProfile.mutate(draft)}
            >
              Save Profile
            </button>
          </div>
        </section>
      </div>

      <section className="rounded-md border p-3">
        <div className="mb-3 flex items-center gap-2 font-medium">
          <ShieldAlert className="size-4" />
          Purpose Assignments
        </div>
        <div className="grid gap-2 md:grid-cols-3">
          {purposes.map((purpose) => (
            <label key={purpose} className="grid gap-1 text-sm">
              <span className="text-xs text-muted-foreground">{purpose}</span>
              <select
                className="h-9 rounded-md border bg-background px-3"
                value={assignmentByPurpose.get(purpose) ?? ""}
                onChange={(event) => {
                  const next = purposes
                    .map((item) => ({
                      purpose: item,
                      profileId: item === purpose ? event.target.value : assignmentByPurpose.get(item) ?? "",
                    }))
                    .filter((assignment) => assignment.profileId);
                  saveAssignments.mutate(next);
                }}
              >
                <option value="">Unassigned</option>
                {profiles.map((profile) => (
                  <option key={profile.id} value={profile.id}>
                    {profile.name}
                  </option>
                ))}
              </select>
            </label>
          ))}
        </div>
      </section>
    </div>
  );
}
