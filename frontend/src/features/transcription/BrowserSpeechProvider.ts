import { mergeSegments, mergeTranscript } from "./dedupe";
import type {
  TranscriptionCallbacks,
  TranscriptionProvider,
} from "./types";

type SpeechAlternative = { transcript: string };
type SpeechResult = {
  isFinal: boolean;
  length: number;
  [index: number]: SpeechAlternative;
};
type SpeechResultList = { length: number; [index: number]: SpeechResult };
type SpeechEvent = { resultIndex: number; results: SpeechResultList };
type SpeechErrorEvent = { error?: string; message?: string };
type SpeechRecognitionLike = {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  onresult: ((event: SpeechEvent) => void) | null;
  onerror: ((event: SpeechErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
};
type SpeechRecognitionConstructor = new () => SpeechRecognitionLike;
type SpeechWindow = Window & {
  SpeechRecognition?: SpeechRecognitionConstructor;
  webkitSpeechRecognition?: SpeechRecognitionConstructor;
};

export class BrowserSpeechProvider implements TranscriptionProvider {
  private recognition: SpeechRecognitionLike | null = null;
  private callbacks: TranscriptionCallbacks | null = null;
  private active = false;
  private restartCount = 0;
  private restartTimer: number | null = null;
  private epoch = 0;
  private committed = "";
  private finalSegments = new Map<string, string>();

  isSupported() {
    if (typeof window === "undefined") return false;
    const target = window as SpeechWindow;
    return Boolean(target.SpeechRecognition || target.webkitSpeechRecognition);
  }

  async start(callbacks: TranscriptionCallbacks) {
    this.dispose();
    this.callbacks = callbacks;
    this.active = true;
    this.restartCount = 0;
    this.committed = "";
    this.finalSegments.clear();
    if (!this.isSupported()) {
      this.active = false;
      callbacks.onStateChange?.("unsupported");
      callbacks.onError("当前浏览器不支持语音识别，请使用文字输入");
      return;
    }
    this.createAndStart();
  }

  async stop() {
    this.active = false;
    if (this.restartTimer !== null) window.clearTimeout(this.restartTimer);
    this.restartTimer = null;
    try {
      this.recognition?.stop();
    } finally {
      this.recognition = null;
      this.callbacks?.onStateChange?.("idle");
    }
  }

  dispose() {
    this.active = false;
    if (this.restartTimer !== null && typeof window !== "undefined") {
      window.clearTimeout(this.restartTimer);
    }
    this.restartTimer = null;
    try {
      this.recognition?.abort();
    } catch {
      // Browser may already have closed the recognizer.
    }
    this.recognition = null;
  }

  private createAndStart() {
    const target = window as SpeechWindow;
    const Constructor =
      target.SpeechRecognition || target.webkitSpeechRecognition;
    if (!Constructor || !this.active) return;
    const recognition = new Constructor();
    this.epoch += 1;
    const currentEpoch = this.epoch;
    recognition.lang = "zh-CN";
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.onresult = (event) => this.onResult(event, currentEpoch);
    recognition.onerror = (event) => this.onError(event);
    recognition.onend = () => this.onEnd();
    this.recognition = recognition;
    recognition.start();
    this.callbacks?.onStateChange?.("listening");
  }

  private onResult(event: SpeechEvent, epoch: number) {
    this.restartCount = 0;
    const liveSegments: string[] = [];
    for (let index = event.resultIndex; index < event.results.length; index += 1) {
      const result = event.results[index];
      const text = result?.[0]?.transcript?.trim() || "";
      if (!text) continue;
      if (result.isFinal) {
        const segmentId = `${epoch}:${index}`;
        if (!this.finalSegments.has(segmentId)) {
          this.finalSegments.set(segmentId, text);
          const next = mergeTranscript(this.committed, text);
          if (next !== this.committed) {
            this.committed = next;
            this.callbacks?.onFinal(this.committed);
          }
        }
      } else {
        liveSegments.push(text);
      }
    }
    this.callbacks?.onPartial(mergeSegments(liveSegments));
  }

  private onError(event: SpeechErrorEvent) {
    const fatal = new Set(["not-allowed", "service-not-allowed", "audio-capture"]);
    if (event.error && fatal.has(event.error)) {
      this.active = false;
      this.callbacks?.onStateChange?.("error");
      this.callbacks?.onError("无法使用麦克风语音识别，请改用文字输入");
      return;
    }
    if (event.error !== "no-speech" && event.error !== "aborted") {
      this.callbacks?.onError(event.message || event.error || "语音识别异常");
    }
  }

  private onEnd() {
    this.recognition = null;
    if (!this.active) return;
    if (this.restartCount >= 3) {
      this.active = false;
      this.callbacks?.onStateChange?.("error");
      this.callbacks?.onError("语音识别连续中断，请使用文字输入");
      return;
    }
    const delay = 250 * 2 ** this.restartCount;
    this.restartCount += 1;
    this.callbacks?.onStateChange?.("restarting");
    this.restartTimer = window.setTimeout(() => {
      this.restartTimer = null;
      this.createAndStart();
    }, delay);
  }
}
