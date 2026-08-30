const TARGET_SAMPLE_RATE = 16_000;
const SAMPLES_PER_CHUNK = 640;

type AudioWindow = Window & {
  webkitAudioContext?: typeof AudioContext;
};

export class PcmMicrophoneSource {
  private context: AudioContext | null = null;
  private stream: MediaStream | null = null;
  private source: MediaStreamAudioSourceNode | null = null;
  private processor: ScriptProcessorNode | null = null;
  private pendingSamples: number[] = [];

  static isSupported() {
    const target = window as AudioWindow;
    const mediaNavigator = navigator as unknown as {
      mediaDevices?: { getUserMedia?: unknown };
    };
    return Boolean(
      typeof mediaNavigator.mediaDevices?.getUserMedia === "function" &&
        ("AudioContext" in window || "webkitAudioContext" in target),
    );
  }

  async start(onChunk: (chunk: ArrayBuffer) => void) {
    await this.stop();

    const target = window as AudioWindow;
    const AudioContextConstructor =
      window.AudioContext || target.webkitAudioContext;

    if (!PcmMicrophoneSource.isSupported() || !AudioContextConstructor) {
      throw new Error("当前浏览器不支持麦克风音频采集");
    }

    this.stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
    });
    this.context = new AudioContextConstructor();
    await this.context.resume();

    this.source = this.context.createMediaStreamSource(this.stream);
    this.processor = this.context.createScriptProcessor(4096, 1, 1);
    this.processor.onaudioprocess = (event) => {
      if (!this.context) return;

      const samples = resampleTo16k(
        event.inputBuffer.getChannelData(0),
        this.context.sampleRate,
      );

      for (const value of samples) {
        const sample = Math.max(-1, Math.min(1, value));
        this.pendingSamples.push(
          sample < 0 ? sample * 0x8000 : sample * 0x7fff,
        );
      }

      while (this.pendingSamples.length >= SAMPLES_PER_CHUNK) {
        const pcm = Int16Array.from(
          this.pendingSamples.splice(0, SAMPLES_PER_CHUNK),
        );
        onChunk(pcm.buffer);
      }
    };

    this.source.connect(this.processor);
    this.processor.connect(this.context.destination);
  }

  async stop() {
    if (this.processor) {
      this.processor.onaudioprocess = null;
      this.processor.disconnect();
    }
    this.source?.disconnect();
    this.stream?.getTracks().forEach((track) => track.stop());

    if (this.context && this.context.state !== "closed") {
      await this.context.close().catch(() => undefined);
    }

    this.processor = null;
    this.source = null;
    this.stream = null;
    this.context = null;
    this.pendingSamples = [];
  }
}

export function resampleTo16k(input: Float32Array, sourceRate: number) {
  if (sourceRate === TARGET_SAMPLE_RATE) return input;

  const outputLength = Math.max(
    1,
    Math.round((input.length * TARGET_SAMPLE_RATE) / sourceRate),
  );
  const output = new Float32Array(outputLength);
  const ratio = sourceRate / TARGET_SAMPLE_RATE;

  for (let index = 0; index < outputLength; index += 1) {
    const position = index * ratio;
    const left = Math.min(input.length - 1, Math.floor(position));
    const right = Math.min(input.length - 1, left + 1);
    const fraction = position - left;
    output[index] = input[left] * (1 - fraction) + input[right] * fraction;
  }

  return output;
}
