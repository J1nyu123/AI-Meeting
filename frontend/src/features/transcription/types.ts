export type VoiceInputStatus =
  | "idle"
  | "connecting"
  | "listening"
  | "stopping"
  | "error";

export interface TranscriptionSnapshot {
  displayText: string;
  committedText: string;
  liveText: string;
  revision: number;
  final: boolean;
}

export interface VoiceInputCallbacks {
  onStatusChange: (status: VoiceInputStatus) => void;
  onTranscript: (snapshot: TranscriptionSnapshot) => void;
  onError: (message: string) => void;
}
