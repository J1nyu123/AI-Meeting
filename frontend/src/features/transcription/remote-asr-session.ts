import { apiRequest } from "@/api/client";
import { PcmMicrophoneSource } from "./pcm-microphone-source";
import type {
  TranscriptionSnapshot,
  VoiceInputCallbacks,
} from "./types";

interface AsrTicket {
  ticket: string;
  expiresAt: string;
  websocketUrl: string;
}

interface AsrMessage {
  type?: string;
  code?: string;
  message?: string;
  displayText?: string;
  committedText?: string;
  liveText?: string;
  revision?: number;
}

const CONNECTION_TIMEOUT_MS = 10_000;
const FINAL_TRANSCRIPT_TIMEOUT_MS = 6_500;

export class RemoteAsrSession {
  private socket: WebSocket | null = null;
  private microphone = new PcmMicrophoneSource();
  private callbacks: VoiceInputCallbacks | null = null;
  private pingTimer: number | null = null;
  private stopTimer: number | null = null;
  private stopResolver: (() => void) | null = null;
  private lastRevision = 0;
  private manuallyStopping = false;

  static isSupported() {
    return PcmMicrophoneSource.isSupported() && "WebSocket" in window;
  }

  async start(callbacks: VoiceInputCallbacks) {
    await this.dispose();
    this.callbacks = callbacks;
    this.lastRevision = 0;
    this.manuallyStopping = false;
    callbacks.onStatusChange("connecting");

    const ticket = await apiRequest<AsrTicket>("/media/asr/tickets", {
      method: "POST",
    });
    await this.connect(ticket);

    try {
      await this.microphone.start((chunk) => {
        if (this.socket?.readyState === WebSocket.OPEN) {
          this.socket.send(chunk);
        }
      });
    } catch (error) {
      await this.dispose();
      throw error;
    }

    callbacks.onStatusChange("listening");
  }

  async stop() {
    if (!this.socket && !this.callbacks) return;

    this.manuallyStopping = true;
    this.callbacks?.onStatusChange("stopping");
    await this.microphone.stop();

    if (this.socket?.readyState !== WebSocket.OPEN) {
      await this.dispose();
      return;
    }

    await new Promise<void>((resolve) => {
      this.stopResolver = resolve;
      this.stopTimer = window.setTimeout(() => {
        this.finishStop();
      }, FINAL_TRANSCRIPT_TIMEOUT_MS);
      this.socket?.send(JSON.stringify({ type: "stop_transcription" }));
    });
  }

  async dispose() {
    await this.microphone.stop();
    this.clearTimers();

    if (this.socket) {
      this.socket.onopen = null;
      this.socket.onmessage = null;
      this.socket.onerror = null;
      this.socket.onclose = null;
      this.socket.close(1000, "client stopped");
    }

    this.socket = null;
    this.callbacks = null;
    this.stopResolver?.();
    this.stopResolver = null;
  }

  private connect(ticket: AsrTicket) {
    return new Promise<void>((resolve, reject) => {
      const websocketUrl = new URL(ticket.websocketUrl, window.location.origin);
      websocketUrl.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";

      const socket = new WebSocket(websocketUrl);
      this.socket = socket;
      let connected = false;
      const timeout = window.setTimeout(() => {
        reject(new Error("语音识别连接超时，请重试"));
        void this.dispose();
      }, CONNECTION_TIMEOUT_MS);

      socket.onopen = () => {
        this.startPing();
      };
      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(String(event.data)) as AsrMessage;

          if (message.type === "connected" && !connected) {
            connected = true;
            window.clearTimeout(timeout);
            socket.send(JSON.stringify({ type: "start_transcription" }));
            resolve();
            return;
          }

          this.handleMessage(message);
        } catch {
          this.fail("语音识别返回了无法解析的数据");
        }
      };
      socket.onerror = () => {
        if (!connected) {
          window.clearTimeout(timeout);
          reject(new Error("无法连接语音识别服务"));
        } else if (!this.manuallyStopping) {
          this.fail("语音识别连接发生异常");
        }
      };
      socket.onclose = (event) => {
        window.clearTimeout(timeout);
        this.stopPing();
        this.socket = null;

        if (!connected) {
          reject(new Error(event.reason || "语音识别连接失败"));
        } else if (this.manuallyStopping) {
          this.finishStop();
        } else {
          this.fail(event.reason || "语音识别连接已断开");
        }
      };
    });
  }

  private handleMessage(message: AsrMessage) {
    if (message.type === "error") {
      this.fail(message.message || message.code || "语音识别失败");
      return;
    }

    if (message.type === "transcription_stopped") {
      this.finishStop();
      return;
    }

    if (message.type !== "transcription" && message.type !== "final") return;

    const revision = Number(message.revision || 0);
    if (revision > 0 && revision <= this.lastRevision) return;
    if (revision > 0) this.lastRevision = revision;

    const snapshot: TranscriptionSnapshot = {
      displayText: message.displayText || message.committedText || "",
      committedText: message.committedText || "",
      liveText: message.liveText || "",
      revision,
      final: message.type === "final",
    };
    this.callbacks?.onTranscript(snapshot);
  }

  private finishStop() {
    const callbacks = this.callbacks;
    this.clearTimers();
    this.stopResolver?.();
    this.stopResolver = null;

    if (this.socket) {
      this.socket.onclose = null;
      this.socket.close(1000, "transcription stopped");
      this.socket = null;
    }

    this.callbacks = null;
    callbacks?.onStatusChange("idle");
  }

  private fail(message: string) {
    const callbacks = this.callbacks;
    callbacks?.onError(message);
    callbacks?.onStatusChange("error");
    void this.dispose();
  }

  private startPing() {
    this.stopPing();
    this.pingTimer = window.setInterval(() => {
      if (this.socket?.readyState === WebSocket.OPEN) {
        this.socket.send(JSON.stringify({ type: "ping" }));
      }
    }, 15_000);
  }

  private stopPing() {
    if (this.pingTimer !== null) window.clearInterval(this.pingTimer);
    this.pingTimer = null;
  }

  private clearTimers() {
    this.stopPing();
    if (this.stopTimer !== null) window.clearTimeout(this.stopTimer);
    this.stopTimer = null;
  }
}
