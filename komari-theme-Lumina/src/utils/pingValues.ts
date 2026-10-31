export function isValidPingLatency(value: number | null | undefined) {
  return value != null && Number.isFinite(value) && value >= 0;
}

export function isLostPingSample(value: number | null | undefined) {
  return value != null && Number.isFinite(value) && value < 0;
}
