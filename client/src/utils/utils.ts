/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
type WebSocketEventHandlers = {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
};

class WebSocketConnection {
  socket: WebSocket;

  constructor(url: string, private eventHandlers: WebSocketEventHandlers = {}) {
    this.socket = new WebSocket(url);

    this.socket.onopen = this.handleOpen.bind(this);
    this.socket.onmessage = this.handleMessage.bind(this);
    this.socket.onclose = this.handleClose.bind(this);
    this.socket.onerror = this.handleError.bind(this);
  }

  handleOpen(event: Event) {
    this.eventHandlers.onOpen && this.eventHandlers.onOpen(event);
  }

  handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage && this.eventHandlers.onMessage(event);
  }

  handleClose(event: CloseEvent) {
    this.eventHandlers.onClose && this.eventHandlers.onClose(event);
  }

  handleError(event: Event) {
    this.eventHandlers.onError && this.eventHandlers.onError(event);
  }

  close() {
    this.socket.close();
  }
}

export class WebSocketManager {
  private connections = new Map<string, WebSocketConnection>();

  connect(id: string, url: string, eventHandlers: WebSocketEventHandlers = {}) {
    if (this.connections.has(id)) {
      throw new Error(`WebSocket connection with id "${id}" already exists`);
    }

    const connection = new WebSocketConnection(url, eventHandlers);
    this.connections.set(id, connection);
  }

  disconnect(id: string) {
    const connection = this.connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.close();
    this.connections.delete(id);
  }

  disconnectAll() {
    for (const id of this.connections.keys()) {
      this.disconnect(id);
    }
  }
}

export const API_URL = import.meta.env.VITE_API_URL as string;
