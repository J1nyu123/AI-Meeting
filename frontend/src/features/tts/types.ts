export type TtsSpeakOptions = {
  lang?: string;
  signal?: AbortSignal;
};

export interface TtsProvider {
  isSupported(): boolean;
  speak(text: string, options?: TtsSpeakOptions): Promise<void>;
  stop(): void;
  dispose(): void;
}
