import type { JSMpegSource } from '@cycjimmy/jsmpeg-player';

export interface SourceHandlers {
  onClose: (code: number, reason: string) => void;
}

/**
 * JSMpeg lets us plug in our own data source. Owning the WebSocket (instead of using the built-in one)
 * means we can read the close code and reason the backend sends, and decide ourselves when to reconnect.
 */
export function createSource(handlers: SourceHandlers) {
  return class StreamSource implements JSMpegSource {
    streaming = true;
    established = false;
    completed = false;
    progress = 0;
    private socket: WebSocket | null = null;
    private destination: { write(data: ArrayBuffer): void } | null = null;
    private destroyed = false;

    constructor(private readonly url: string) {}

    connect(destination: { write(data: ArrayBuffer): void }) {
      this.destination = destination;
    }

    start() {
      if (this.destroyed) return;
      const socket = new WebSocket(this.url);
      socket.binaryType = 'arraybuffer';
      socket.onopen = () => {
        this.progress = 1;
      };
      socket.onmessage = (ev: MessageEvent<ArrayBuffer>) => {
        this.established = true;
        if (ev.data instanceof ArrayBuffer) this.destination?.write(ev.data);
      };
      socket.onclose = (ev) => {
        if (!this.destroyed) handlers.onClose(ev.code, ev.reason);
      };
      this.socket = socket;
    }

    resume() {
      /* live source: nothing to resume */
    }

    destroy() {
      this.destroyed = true;
      const socket = this.socket;
      if (!socket) return;
      socket.onopen = socket.onmessage = socket.onclose = null;
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) socket.close();
      this.socket = null;
    }
  };
}
