import service from "@/lib/request";
import { resolveApiBaseUrl, resolveRuntimeWsBaseUrl, resolveWsBaseUrl } from "@/config/env";
import type { TranscriptionSnapshot } from "@/features/transcription/types";

type TicketResponse = { ticket: string; expiresAt: string; websocketUrl: string };
type ASRMessage = Partial<TranscriptionSnapshot> & { type?: string; code?: string; message?: string; revision?: number };

export class AudioToTextWebSocket {
  private ws: WebSocket | null = null;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private pendingBinaryQueue: Array<ArrayBuffer | Blob> = [];
  private readonly maxPendingBinaryChunks = 64;
  private lastRevision = 0;
  private lastMessageKey: string | null = null;
  public onSnapshot?: (snapshot: TranscriptionSnapshot) => void;
  public onTranscription?: (text: string) => void;
  public onFinal?: (text: string) => void;
  public onError?: (error: string) => void;
  public onConnected?: () => void;
  public onDisconnected?: () => void;

  async connect() {
    if (this.ws && (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN)) return;
    const ticket = await service.post<TicketResponse>("/v1/media/asr/tickets");
    const wsBase = resolveRuntimeWsBaseUrl(window.location, resolveWsBaseUrl(import.meta.env.VITE_WS_BASE_URL));
    const apiBase = resolveApiBaseUrl(import.meta.env.VITE_API_BASE_URL);
    const serverPath = ticket.websocketUrl?.trim();
    const path = serverPath ? `${apiBase}${serverPath.replace(/^\/api/, "")}` : `${apiBase}/v1/media/asr/ws?ticket=${encodeURIComponent(ticket.ticket)}`;
    this.resetCursor();
    await new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(`${wsBase}${path}`);
      this.ws = ws;
      let opened = false;
      let ready = false;
      const readyTimer = window.setTimeout(() => { ws.close(1000, "handshake timeout"); reject(new Error("远程语音识别握手超时")); }, 10_000);
      ws.onopen = () => { opened = true; this.startPing(); };
      ws.onmessage = (event) => { try { const message = JSON.parse(String(event.data)) as ASRMessage; this.handleMessage(message); if (message.type === "connected" && !ready) { ready = true; window.clearTimeout(readyTimer); this.flushPendingBinaryQueue(); resolve(); } } catch { this.onError?.("语音识别返回了无效数据"); } };
      ws.onerror = () => { if (!opened) reject(new Error("无法连接远程语音识别")); this.onError?.("远程语音识别连接异常"); };
      ws.onclose = (event) => { window.clearTimeout(readyTimer); this.stopPing(); if (!ready) reject(new Error(event.reason || "远程语音识别连接失败")); this.ws = null; this.onDisconnected?.(); };
    });
  }

  private handleMessage(message: ASRMessage) {
    if (message.type === "connected") { this.onConnected?.(); return; }
    if (message.type === "error") { this.onError?.(message.message || message.code || "语音识别异常"); return; }
    if (message.type !== "transcription" && message.type !== "final") return;
    const revision = Number(message.revision || 0);
    const key = `${message.type}:${revision}:${message.displayText || ""}`;
    if ((revision > 0 && revision <= this.lastRevision) || key === this.lastMessageKey) return;
    if (revision > 0) this.lastRevision = revision;
    this.lastMessageKey = key;
    const snapshot: TranscriptionSnapshot = { displayText: message.displayText || "", committedText: message.committedText || "", liveText: message.liveText || "", revision, status: message.type === "final" ? "final" : "running" };
    this.onSnapshot?.(snapshot);
    if (snapshot.status === "final") this.onFinal?.(snapshot.committedText); else this.onTranscription?.(snapshot.displayText);
  }
  private startPing() { this.stopPing(); this.pingInterval = setInterval(() => this.sendCommand("ping"), 15_000); }
  private stopPing() { if (this.pingInterval) clearInterval(this.pingInterval); this.pingInterval = null; }
  sendCommand(type: "ping" | "start_transcription" | "stop_transcription" | "get_status") { if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type })); }
  sendAudio(data: Blob | ArrayBuffer) { if (this.ws?.readyState === WebSocket.OPEN) { this.ws.send(data); return; } if (this.ws?.readyState === WebSocket.CONNECTING) { if (this.pendingBinaryQueue.length >= this.maxPendingBinaryChunks) throw new Error("语音缓冲区已满，请稍后重试"); this.pendingBinaryQueue.push(data); } }
  disconnect() { this.stopPing(); this.pendingBinaryQueue = []; this.resetCursor(); this.ws?.close(1000, "client stopped"); this.ws = null; }
  private flushPendingBinaryQueue() { if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return; this.pendingBinaryQueue.forEach((chunk) => this.ws?.send(chunk)); this.pendingBinaryQueue = []; }
  private resetCursor() { this.lastRevision = 0; this.lastMessageKey = null; }
}
