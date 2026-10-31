/**
 * Every tab's `BroadcastChannel` in one test process: a message reaches every other channel
 * of its name, synchronously, and each channel keeps what it received. Stub it in with
 * `vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel)` and reset `open` between tests.
 */
export class FakeBroadcastChannel {
  static open: FakeBroadcastChannel[] = [];

  readonly name: string;
  readonly received: unknown[] = [];
  private readonly listeners: ((event: MessageEvent<unknown>) => void)[] = [];

  constructor(name: string) {
    this.name = name;
    FakeBroadcastChannel.open.push(this);
  }

  addEventListener(_type: "message", listener: (event: MessageEvent<unknown>) => void): void {
    this.listeners.push(listener);
  }

  postMessage(data: unknown): void {
    for (const channel of FakeBroadcastChannel.open) {
      if (channel !== this && channel.name === this.name) {
        channel.deliver(data);
      }
    }
  }

  close(): void {}

  private deliver(data: unknown): void {
    this.received.push(data);
    for (const listener of this.listeners) {
      listener(new MessageEvent("message", { data }));
    }
  }
}
