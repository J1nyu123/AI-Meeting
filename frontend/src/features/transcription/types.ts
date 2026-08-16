export type TranscriptionState = {
  liveText: string;
  committedText: string;
  status: "idle" | "listening" | "restarting" | "unsupported" | "error";
};

export type TranscriptionCallbacks = {
  onPartial: (text: string) => void;
  onFinal: (text: string) => void;
  onStateChange?: (state: TranscriptionState["status"]) => void;
  onError: (message: string) => void;
};

export interface TranscriptionProvider {
  isSupported(): boolean;
  start(callbacks: TranscriptionCallbacks): Promise<void>;
  stop(): Promise<void>;
  dispose(): void;
}

export interface RemoteAsrTransport {
  connect(callbacks: TranscriptionCallbacks): Promise<void>;
  disconnect(): Promise<void>;
}
