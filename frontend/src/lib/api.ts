import type { LogPage, SearchFilters, Stats } from "@/types";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

const API_KEY = process.env.NEXT_PUBLIC_API_KEY ?? "";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

function authHeaders(): HeadersInit {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (API_KEY) headers["X-API-Key"] = API_KEY;
  return headers;
}

function filtersToParams(f: SearchFilters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  if (f.levels.length) p.set("levels", f.levels.join(","));
  if (f.services.length) p.set("services", f.services.join(","));
  if (f.search.trim()) p.set("search", f.search.trim());
  p.set("limit", String(f.limit));
  p.set("offset", String(f.offset));
  p.set("order", f.order);
  return p;
}

async function request<T>(path: string): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      headers: authHeaders(),
      cache: "no-store",
    });
  } catch (e) {
    throw new ApiError(0, `cannot reach API: ${(e as Error).message}`);
  }

  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) detail = body.error;
    } catch {
      // no JSON body
    }
    throw new ApiError(res.status, detail);
  }
  return (await res.json()) as T;
}

function normalizeStats(raw: Stats): Stats {
  return {
    ...raw,
    by_level: raw.by_level ?? {},
    top_services: raw.top_services ?? [],
    histogram: raw.histogram ?? [],
  };
}

function normalizeLogPage(raw: LogPage): LogPage {
  return {
    ...raw,
    logs: raw.logs ?? [],
    total: raw.total ?? 0,
  };
}

export const api = {
  searchLogs(filters: SearchFilters): Promise<LogPage> {
    return request<LogPage>(`/api/v1/logs/search?${filtersToParams(filters)}`).then(
      normalizeLogPage,
    );
  },

  getStats(filters: SearchFilters): Promise<Stats> {
    return request<Stats>(`/api/v1/logs/stats?${filtersToParams(filters)}`).then(
      normalizeStats,
    );
  },

  tailUrl(service?: string, level?: string): string {
    const ws = API_BASE.replace(/^http/, "ws");
    const p = new URLSearchParams();
    if (service) p.set("service", service);
    if (level) p.set("level", level);
    if (API_KEY) p.set("api_key", API_KEY);
    const qs = p.toString();
    return `${ws}/api/v1/logs/tail${qs ? `?${qs}` : ""}`;
  },
};
