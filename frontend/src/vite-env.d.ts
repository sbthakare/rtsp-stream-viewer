/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string;
  readonly VITE_DEMO_STREAMS?: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module '@cycjimmy/jsmpeg-player' {
  export interface JSMpegSource {
    streaming: boolean;
    established: boolean;
    completed: boolean;
    progress: number;
    connect(destination: { write(data: ArrayBuffer): void }): void;
    start(): void;
    resume(secondsHeadroom?: number): void;
    destroy(): void;
  }
  export interface JSMpegOptions {
    canvas?: HTMLCanvasElement;
    source?: new (url: string, options: Record<string, unknown>) => JSMpegSource;
    autoplay?: boolean;
    audio?: boolean;
    loop?: boolean;
    pauseWhenHidden?: boolean;
    videoBufferSize?: number;
    disableGl?: boolean;
    onVideoDecode?: (decoder: unknown, time: number) => void;
    [key: string]: unknown;
  }
  export class Player {
    constructor(url: string, options?: JSMpegOptions);
    destroy(): void;
  }
  const JSMpeg: { Player: typeof Player };
  export default JSMpeg;
}
