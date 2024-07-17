/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
  message: string;
};

interface WebSocketOptions {
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
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

/**
 * Represents a WebSocket connection, providing lifecycle event handling.
 */
export class WebSocketConnection {
  private socket: WebSocket;
  private url: string;
  private reconnectInterval: number;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number;

  constructor(
    url: string,
    private eventHandlers: Handlers,
    options: WebSocketOptions = {},
  ) {
    this.url = url;
    this.reconnectInterval = options.reconnectInterval || 5000;
    this.maxReconnectAttempts = options.maxReconnectAttempts || 5;
    this.socket = this.createWebSocket();
  }

  private createWebSocket(): WebSocket {
    const socket = new WebSocket(this.url);
    socket.onopen = this.handleOpen.bind(this);
    socket.onmessage = this.handleMessage.bind(this);
    socket.onclose = this.handleClose.bind(this);
    socket.onerror = this.handleError.bind(this);
    return socket;
  }

  private handleOpen(event: Event) {
    this.reconnectAttempts = 0; // Reset reconnect attempts
    this.eventHandlers.onOpen?.(event);
  }

  private handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage?.(event);
  }

  private handleClose(event: CloseEvent) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      setTimeout(() => {
        this.reconnectAttempts++;
        this.socket = this.createWebSocket();
      }, this.reconnectInterval);
    }
    this.eventHandlers.onClose?.(event);
  }

  private handleError(event: Event) {
    console.error("WebSocket error:", event);
    this.eventHandlers.onError?.(event);
  }

  public send<T>(data: T) {
    this.socket.send(JSON.stringify(data));
  }

  public receive<T>(handler: (data: T) => void) {
    this.socket.onmessage = (event: MessageEvent) => {
      const data = JSON.parse(event.data) as T;
      handler(data);
    };
  }

  public close() {
    this.socket.close();
  }

  public isConnected(): boolean {
    return this.socket.readyState === WebSocket.OPEN;
  }

  public isConnecting(): boolean {
    return this.socket.readyState === WebSocket.CONNECTING;
  }

  public isClosed(): boolean {
    return this.socket.readyState === WebSocket.CLOSED;
  }
}

export interface WebSocketManager {
  connect: (
    id: string,
    url: string,
    handlers: Handlers,
    options?: WebSocketOptions,
  ) => WebSocketConnection | boolean;
  disconnect: (id: string) => void;
  get: (id: string) => WebSocketConnection;
  send: <T>(id: string, data: T) => void;
  sendJson: (id: string, data: any) => void;
  receive: <T>(id: string, handler: (data: T) => void) => void;
  has: (id: string) => boolean;
}

export function createWebsocketManager(): WebSocketManager {
  const connections = new Map<string, WebSocketConnection>();

  function connect(
    id: string,
    url: string,
    handlers: Handlers,
    options: WebSocketOptions = {},
  ): WebSocketConnection | boolean {
    if (connections.has(id)) {
      return connections.get(id) as WebSocketConnection;
    }

    const connection = new WebSocketConnection(url, handlers, options);
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
  }

  function send<T>(id: string, data: T) {
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

  function receive<T>(id: string, handler: (data: T) => void) {
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
    sendJson,
    receive,
    get,
    has,
  };
}
