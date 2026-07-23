import { api } from "@/lib/api";
import { type Channel, Realtime } from "@foony/realtime";

interface RealtimeTokenResponse {
  token: string;
  clientId: string;
  expiresAt: number;
}

export class RealtimeService {
  private client: Realtime | null = null;

  public connect() {
    if (this.client) {
      return this.client;
    }

    this.client = new Realtime({
      authCallback: () => this.authorize(),
    });

    return this.client;
  }

  public getClient() {
    return this.client;
  }

  public isConnectedOrConnecting() {
    const state = this.client?.getState();
    return state === "connected" || state === "connecting";
  }

  public disconnect() {
    this.safeClose();
  }

  public safeClose() {
    if (!this.client) {
      return;
    }

    const client = this.client;
    this.client = null;

    try {
      const state = client.getState();
      if (state !== "closing" && state !== "closed") {
        void client.close();
      }
    } catch {
      // Best-effort close: if already closed/disposed, no action needed.
    }
  }

  public getChannel(name: string): Channel {
    if (!this.client) {
      throw new Error("Realtime client is not connected");
    }

    return this.client.channels.get(name);
  }

  public getUsersPresenceChannelName(orgId: string, buId: string) {
    return `tenant:${orgId}:${buId}:presence:users`;
  }

  public getDataEventsChannelName(orgId: string, buId: string) {
    return `tenant:${orgId}:${buId}:data-events`;
  }

  public connectionState() {
    return this.client?.getState() ?? "closed";
  }

  public async leavePresenceIfPossible(channelName: string) {
    const client = this.client;
    if (!client || client.getState() !== "connected") {
      return;
    }

    const channel = client.channels.get(channelName);
    if (channel.state !== "attached" && channel.state !== "attaching") {
      return;
    }

    try {
      await channel.presence.leave();
    } catch {
      // Best-effort leave: ignore incompatible/closing state races during teardown.
    }
  }

  private async authorize(): Promise<string> {
    const response = await api.get<RealtimeTokenResponse>("/realtime/token-request/");
    return response.token;
  }
}

export const realtimeService = new RealtimeService();
