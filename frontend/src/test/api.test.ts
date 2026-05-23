import { describe, it, expect, vi, beforeEach } from "vitest";
import { DEFAULT_FILTERS, type SearchFilters } from "@/types";

const mockFetch = vi.fn().mockResolvedValue({
  ok: true,
  json: async () => ({ logs: [], total: 0 }),
});
vi.stubGlobal("fetch", mockFetch);

const { api } = await import("@/lib/api");

beforeEach(() => mockFetch.mockClear());

describe("api.searchLogs", () => {
  it("calls the search endpoint", async () => {
    await api.searchLogs(DEFAULT_FILTERS);
    const [url] = mockFetch.mock.calls[0] as [string, unknown];
    expect(url).toContain("/api/v1/logs/search");
  });

  it("passes from and limit as query params", async () => {
    const filters: SearchFilters = { ...DEFAULT_FILTERS, from: "15m", limit: 50 };
    await api.searchLogs(filters);
    const [url] = mockFetch.mock.calls[0] as [string, unknown];
    expect(url).toContain("from=15m");
    expect(url).toContain("limit=50");
  });

  it("passes level filter as comma-separated string", async () => {
    const filters: SearchFilters = {
      ...DEFAULT_FILTERS,
      levels: ["error", "warn"],
    };
    await api.searchLogs(filters);
    const [url] = mockFetch.mock.calls[0] as [string, unknown];
    expect(url).toContain("levels=error%2Cwarn");
  });

  it("omits empty search param", async () => {
    const filters: SearchFilters = { ...DEFAULT_FILTERS, search: "  " };
    await api.searchLogs(filters);
    const [url] = mockFetch.mock.calls[0] as [string, unknown];
    expect(url).not.toContain("search=");
  });
});

describe("api.tailUrl", () => {
  it("returns a ws:// URL", () => {
    const url = api.tailUrl();
    expect(url).toMatch(/^ws/);
    expect(url).toContain("/api/v1/logs/tail");
  });

  it("appends service and level params when provided", () => {
    const url = api.tailUrl("my-api", "error");
    expect(url).toContain("service=my-api");
    expect(url).toContain("level=error");
  });

  it("produces a clean URL when no filters given", () => {
    const url = api.tailUrl();
    expect(url).not.toContain("?");
  });
});

describe("api.getStats", () => {
  it("calls the stats endpoint and normalizes null slices", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        total: 0,
        by_level: null,
        top_services: null,
        histogram: null,
      }),
    });
    const stats = await api.getStats(DEFAULT_FILTERS);
    const [url] = mockFetch.mock.calls[0] as [string, unknown];
    expect(url).toContain("/api/v1/logs/stats");
    expect(stats.histogram).toEqual([]);
    expect(stats.top_services).toEqual([]);
  });
});
