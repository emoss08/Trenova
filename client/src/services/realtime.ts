import { api } from "@/lib/api";
import * as Ably from "ably";

interface RealtimeTokenRequest {
  keyName: string;
  clientId: string;
  nonce: string;
  mac: string;
  capability: string;
  timestamp: number;
  ttl: number;
}

export class RealtimeService {
  private client: Ably.Realtime | null = null;

  public connect() {
    if (this.client) {
      return this.client;
    }

    this.client = new Ably.Realtime({
      autoConnect: true,
      authCallback: (_tokenParams, callback) => {
        void this.authorize(callback);
      },
    });

    return this.client;
  }

  public getClient() {
    return this.client;
  }

  public isConnectedOrConnecting() {
    const state = this.client?.connection.state;
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
      if (client.connection.state !== "closing" && client.connection.state !== "closed") {
        client.close();
      }
    } catch {
      // Best-effort close: if already closed/disposed, no action needed.
    }
  }

  public getChannel(name: string) {
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
    return this.client?.connection.state ?? "closed";
  }

  public async leavePresenceIfPossible(channelName: string) {
    const client = this.client;
    if (!client || client.connection.state !== "connected") {
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

  private async authorize(
    callback: (
      error: string | Ably.ErrorInfo | null,
      tokenRequestOrDetails:
        | string
        | Ably.TokenDetails
        | Ably.TokenRequest
        | null,
    ) => void,
  ) {
    try {
      const tokenRequest = await api.get<RealtimeTokenRequest>(
        "/realtime/token-request/",
      );
      callback(null, tokenRequest as Ably.TokenRequest);
    } catch {
      callback("Failed to fetch realtime token request", null);
    }
  }
}
