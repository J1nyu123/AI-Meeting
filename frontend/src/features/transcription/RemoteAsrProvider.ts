import type {
  RemoteAsrTransport,
  TranscriptionCallbacks,
  TranscriptionProvider,
} from "./types";

/** Transcription provider backed by a replaceable remote transport. */
export class RemoteAsrProvider implements TranscriptionProvider {
  private readonly transport: RemoteAsrTransport;

  constructor(transport: RemoteAsrTransport) {
    this.transport = transport;
  }
  isSupported() {
    return true;
  }
  start(callbacks: TranscriptionCallbacks) {
    return this.transport.connect(callbacks);
  }
  stop() {
    return this.transport.disconnect();
  }
  dispose() {
    void this.transport.disconnect();
  }
}
