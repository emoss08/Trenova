export type TWebsocketStatuses =
  | "SUCCESS"
  | "FAILURE"
  | "WARNING"
  | "PROCESSING"
  | "INFO";

export type WebSocketMessageProps = {
  action?: string | null;
  step?: number | null;
  status?: TWebsocketStatuses | null;
  message:
    | string
    | Array<
        Array<{
          invoice_number: string;
          missing_documents: Array<string>;
        }>
      >;
};

/**
 * Represents a WebSocket connection, providing lifecycle event handling.
 */
export class WebSocketConnection {
  private socket: WebSocket;

  constructor(
    url: string,
    private eventHandlers: Handlers,
  ) {
    this.socket = new WebSocket(url);

    this.socket.onopen = this.handleOpen.bind(this);
    this.socket.onmessage = this.handleMessage.bind(this);
    this.socket.onclose = this.handleClose.bind(this);
    this.socket.onerror = this.handleError.bind(this);
  }

  private handleOpen(event: Event) {
    this.eventHandlers.onOpen?.(event);
  }

  private handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage?.(event);
  }

  private handleClose(event: CloseEvent) {
    this.eventHandlers.onClose?.(event);
  }

  private handleError(event: Event) {
    this.eventHandlers.onError?.(event);
  }

  public send(data: any) {
    this.socket.send(data);
  }

  public receive(handler: (event: WebSocketEvent) => void) {
    this.socket.onmessage = handler;
  }

  public close() {
    this.socket.close();
  }
}

interface Handlers {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onError?: (event: Event) => void;
  onClose?: (event: CloseEvent) => void;
}

interface WebSocketEventMap {
  close: CloseEvent;
  error: Event;
  message: MessageEvent;
  open: Event;
}

export type WebSocketEvent = WebSocketEventMap[keyof WebSocketEventMap];

export interface WebSocketManager {
  connect: (
    id: string,
    url: string,
    handlers: Handlers,
  ) => WebSocketConnection | boolean;
  disconnect: (id: string) => void;
  get: (id: string) => WebSocketConnection;
  send: (id: string, data: any) => void;
  sendJson: (id: string, data: any) => void;
  receive: (id: string, handler: (event: WebSocketEvent) => void) => void;
  has: (id: string) => boolean;
}

export function createWebsocketManager(): {
  disconnect: (id: string) => WebSocketConnection;
  disconnectFromAll: () => void;
  receive: (id: string, handler: (event: WebSocketEvent) => void) => void;
  sendJson: (id: string, data: any) => void;
  get: (id: string) => WebSocketConnection;
  has: (id: string) => boolean;
  send: (id: string, data: any) => void;
  connect: (
    id: string,
    url: string,
    handlers: Handlers,
  ) => WebSocketConnection | boolean;
} {
  const connections = new Map<string, WebSocketConnection>();

  function connect(
    id: string,
    url: string,
    handlers: Handlers,
  ): WebSocketConnection | boolean {
    if (connections.has(id)) {
      return connections.delete(id);
    }

    const connection = new WebSocketConnection(url, handlers);
    connections.set(id, connection);

    return connection;
  }

  function disconnect(id: string) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.close();
    connections.delete(id);

    return connection;
  }

  function disconnectFromAll() {
    connections.forEach((connection) => {
      connection.close();
    });
    connections.clear();
  }

  function send(id: string, data: any) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`No connection with id ${id} found`);
    }

    connection.send(data);
  }

  function sendJson(id: string, data: any) {
    send(id, JSON.stringify(data));
  }

  function get(id: string) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    return connection;
  }

  function receive(id: string, handler: (event: WebSocketEvent) => void) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`No connection with id ${id} found`);
    }

    connection.receive(handler);
  }

  function has(id: string) {
    return connections.has(id);
  }

  return {
    connect,
    disconnect,
    send,
    disconnectFromAll,
    sendJson,
    receive,
    get,
    has,
  };
}
