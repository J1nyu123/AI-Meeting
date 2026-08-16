import { mergeTranscript } from "./dedupe";
import { BrowserSpeechProvider } from "./BrowserSpeechProvider";
import { PcmMicrophoneSource } from "./PcmMicrophoneSource";
import type { TranscriptionCallbacks, TranscriptionProvider } from "./types";
import { AudioToTextWebSocket } from "@/services/audioToTextWs";

export class XunfeiPreferredProvider implements TranscriptionProvider {
  private static failures = 0;
  private transport: AudioToTextWebSocket | null = null;
  private microphone: PcmMicrophoneSource | null = null;
  private browser = new BrowserSpeechProvider();
  private remoteCommitted = "";
  private fallbackStarted = false;
  private remoteFinished = false;
  isSupported() { return Boolean(navigator.mediaDevices?.getUserMedia) || this.browser.isSupported(); }
  async start(callbacks: TranscriptionCallbacks) {
    if (!navigator.mediaDevices?.getUserMedia) {
      await this.startBrowser(callbacks);
      return;
    }
    if (XunfeiPreferredProvider.failures >= 3) { await this.startBrowser(callbacks); return; }
    const transport = new AudioToTextWebSocket(); const microphone = new PcmMicrophoneSource(); this.transport = transport; this.microphone = microphone;
    transport.onSnapshot = (snapshot) => { this.remoteCommitted = snapshot.committedText; this.remoteFinished = snapshot.status === "final"; callbacks.onSnapshot?.(snapshot); if (!callbacks.onSnapshot) { callbacks.onPartial(snapshot.liveText); if (snapshot.status === "final") callbacks.onFinal(snapshot.committedText); } };
    transport.onConnected = () => transport.sendCommand("start_transcription");
    const fallback = async (message: string) => {
      if (this.fallbackStarted || this.remoteFinished) return;
      this.fallbackStarted = true;
      callbacks.onError(message);
      XunfeiPreferredProvider.failures += 1;
      await microphone.stop();
      transport.disconnect();
      await this.startBrowser(callbacks);
    };
    transport.onError = (message) => { void fallback(message); };
    transport.onDisconnected = () => { if (!this.remoteFinished) void fallback("远程语音识别已断开，已切换浏览器识别"); };
    try { await transport.connect(); await microphone.start((chunk) => transport.sendAudio(chunk)); callbacks.onStateChange?.("listening"); }
    catch (error) { await fallback(error instanceof Error ? error.message : "远程语音识别不可用，已切换浏览器识别"); }
  }
  private startBrowser(callbacks: TranscriptionCallbacks) { const base = this.remoteCommitted; return this.browser.start({ ...callbacks, onFinal: (text) => callbacks.onFinal(mergeTranscript(base, text)) }); }
  async stop() { this.transport?.sendCommand("stop_transcription"); const browserStop = this.browser.stop(); await this.microphone?.stop(); await browserStop; }
  dispose() { void this.microphone?.stop(); this.transport?.disconnect(); this.browser.dispose(); }
}
