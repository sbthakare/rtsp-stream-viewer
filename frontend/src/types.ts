export interface StreamItem {
  id: string;
  url: string;
  name: string;
}

export type TileStatus = 'connecting' | 'live' | 'reconnecting' | 'paused' | 'error';

export type GridColumns = 'auto' | 1 | 2 | 3 | 4;
