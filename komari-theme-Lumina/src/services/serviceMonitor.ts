export interface ServiceConfig {
  id: number;
  name: string;
  type: "http" | "astrbot";
  url: string;
  interval: number;
  timeout: number;
  expected_status: number;
  keyword: string;
  auth_type: string;
  api_path: string;
  bot_id: string;
  enabled: boolean;
  public: boolean;
}

export interface ServiceCheck {
  id: number;
  time: string;
  up: boolean;
  latency: number;
  status_code: number;
  error?: string;
}

export interface ServiceView {
  id: number;
  name: string;
  type: string;
  status: "up" | "down" | "unknown" | "paused";
  latest: ServiceCheck | null;
  availability_24h: number | null;
  checks_24h: number;
  config?: ServiceConfig;
  has_token?: boolean;
}

export const newServiceConfig = (): ServiceConfig => ({
  id: 0, name: "", type: "http", url: "", interval: 60, timeout: 10,
  expected_status: 0, keyword: "", auth_type: "apikey", api_path: "/api/v1/bots",
  bot_id: "", enabled: true, public: false,
});

export async function serviceRequest<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, {
    credentials: "same-origin", cache: "no-store", signal,
    method: body === undefined ? "GET" : "POST",
    headers: body === undefined ? {} : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const result = await response.json();
  if (!response.ok || result.status === "error") throw new Error(result.message || `HTTP ${response.status}`);
  return result.data as T;
}
