const normalize = (value: string) =>
  value.replace(/\s+/g, " ").replace(/\s+([，。！？,.!?])/g, "$1").trim();

export function mergeTranscript(base: string, next: string): string {
  const left = normalize(base);
  const right = normalize(next);
  if (!left) return right;
  if (!right || left === right || left.includes(right)) return left;
  if (right.includes(left)) return right;

  const maximum = Math.min(left.length, right.length);
  for (let overlap = maximum; overlap > 0; overlap -= 1) {
    if (left.slice(-overlap) === right.slice(0, overlap)) {
      return normalize(left + right.slice(overlap));
    }
  }
  return `${left} ${right}`;
}

export const mergeSegments = (segments: Iterable<string>) => {
  let merged = "";
  for (const segment of segments) merged = mergeTranscript(merged, segment);
  return merged;
};
