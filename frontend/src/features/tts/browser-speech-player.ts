export class BrowserSpeechPlayer {
  private current: SpeechSynthesisUtterance | null = null;

  static isSupported() {
    return (
      "speechSynthesis" in window &&
      typeof window.SpeechSynthesisUtterance !== "undefined"
    );
  }

  speak(text: string, signal?: AbortSignal) {
    if (!BrowserSpeechPlayer.isSupported()) {
      return Promise.reject(new Error("当前浏览器不支持中文语音播报"));
    }

    this.stop();
    const utterance = new SpeechSynthesisUtterance(text.trim());
    utterance.lang = "zh-CN";
    utterance.rate = 1;
    utterance.pitch = 1;
    this.current = utterance;

    return new Promise<void>((resolve, reject) => {
      let settled = false;

      const cleanup = () => {
        signal?.removeEventListener("abort", handleAbort);
        utterance.onend = null;
        utterance.onerror = null;
        if (this.current === utterance) this.current = null;
      };
      const finish = (error?: Error) => {
        if (settled) return;
        settled = true;
        cleanup();
        if (error) reject(error);
        else resolve();
      };
      const handleAbort = () => {
        window.speechSynthesis.cancel();
        finish(new DOMException("Speech playback aborted", "AbortError"));
      };

      utterance.onend = () => finish();
      utterance.onerror = (event) => {
        const reason = event.error || "浏览器语音播报失败";
        finish(
          reason === "canceled" || reason === "interrupted"
            ? new DOMException("Speech playback aborted", "AbortError")
            : new Error(reason),
        );
      };
      signal?.addEventListener("abort", handleAbort, { once: true });

      if (signal?.aborted) {
        handleAbort();
        return;
      }

      window.speechSynthesis.resume();
      window.speechSynthesis.speak(utterance);
    });
  }

  stop() {
    if (!this.current) return;
    window.speechSynthesis.cancel();
    this.current = null;
  }
}
