export class PcmMicrophoneSource {
  private context: AudioContext | null = null;
  private stream: MediaStream | null = null;
  private source: MediaStreamAudioSourceNode | null = null;
  private processor: ScriptProcessorNode | null = null;
  private pending: number[] = [];
  async start(onChunk: (chunk: ArrayBuffer) => void) {
    await this.stop();
    this.stream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, sampleRate: 16_000 } });
    const Constructor = window.AudioContext || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Constructor) throw new Error("当前浏览器不支持音频采集");
    this.context = new Constructor({ sampleRate: 16_000 });
    this.source = this.context.createMediaStreamSource(this.stream);
    this.processor = this.context.createScriptProcessor(4096, 1, 1);
    this.processor.onaudioprocess = (event) => {
      const samples = resampleTo16k(event.inputBuffer.getChannelData(0), this.context?.sampleRate || 16_000);
      for (const value of samples) { const sample = Math.max(-1, Math.min(1, value)); this.pending.push(sample < 0 ? sample * 0x8000 : sample * 0x7fff); }
      while (this.pending.length >= 640) { const pcm = Int16Array.from(this.pending.splice(0, 640)); onChunk(pcm.buffer); }
    };
    this.source.connect(this.processor); this.processor.connect(this.context.destination);
  }
  async stop() { this.processor?.disconnect(); this.source?.disconnect(); this.stream?.getTracks().forEach((track) => track.stop()); await this.context?.close().catch(() => undefined); this.processor = null; this.source = null; this.stream = null; this.context = null; this.pending = []; }
}

export function resampleTo16k(input: Float32Array, sourceRate: number) {
  if (sourceRate === 16_000) return input;
  const outputLength = Math.max(1, Math.round(input.length * 16_000 / sourceRate));
  const output = new Float32Array(outputLength);
  const ratio = sourceRate / 16_000;
  for (let index = 0; index < outputLength; index += 1) {
    const position = index * ratio;
    const left = Math.min(input.length - 1, Math.floor(position));
    const right = Math.min(input.length - 1, left + 1);
    const fraction = position - left;
    output[index] = input[left] * (1 - fraction) + input[right] * fraction;
  }
  return output;
}
