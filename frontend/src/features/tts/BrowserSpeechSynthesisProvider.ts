import type { TtsProvider, TtsSpeakOptions } from "./types";

const abortError = () => new DOMException("Speech synthesis aborted", "AbortError");

export class BrowserSpeechSynthesisProvider implements TtsProvider {
  private current: SpeechSynthesisUtterance | null = null;
  private cancelCurrent: (() => void) | null = null;

  isSupported() {
    return (
      typeof window !== "undefined" &&
      typeof window.speechSynthesis !== "undefined" &&
      typeof SpeechSynthesisUtterance !== "undefined"
    );
  }

  speak(text: string, options?: TtsSpeakOptions) {
    const normalized = text.trim();
    if (!normalized) return Promise.resolve();
    if (!this.isSupported()) {
      return Promise.reject(new Error("Browser speech synthesis is unsupported"));
    }

    this.stop();
    const synthesis = window.speechSynthesis;
    const utterance = new SpeechSynthesisUtterance(normalized);
    utterance.lang = options?.lang || "zh-CN";
    utterance.rate = 1;
    utterance.pitch = 1;

    return new Promise<void>((resolve, reject) => {
      let settled = false;
      const cleanup = () => {
        options?.signal?.removeEventListener("abort", handleAbort);
        utterance.onend = null;
        utterance.onerror = null;
        if (this.current === utterance) {
          this.current = null;
          this.cancelCurrent = null;
        }
      };
      const finish = (error?: Error) => {
        if (settled) return;
        settled = true;
        cleanup();
        if (error) reject(error);
        else resolve();
      };
      const handleAbort = () => {
        synthesis.cancel();
        finish(abortError());
      };

      utterance.onend = () => finish();
      utterance.onerror = (event) => {
        const reason = event.error || "speech synthesis failed";
        finish(
          reason === "canceled" || reason === "interrupted"
            ? abortError()
            : new Error(reason),
        );
      };
      this.current = utterance;
      this.cancelCurrent = () => finish(abortError());
      options?.signal?.addEventListener("abort", handleAbort, { once: true });
      if (options?.signal?.aborted) {
        handleAbort();
        return;
      }

      synthesis.resume();
      synthesis.speak(utterance);
    });
  }

  stop() {
    if (!this.current) return;
    window.speechSynthesis.cancel();
    this.cancelCurrent?.();
  }

  dispose() {
    this.stop();
  }
}
