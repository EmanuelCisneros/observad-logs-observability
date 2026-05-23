export type Level = "debug" | "info" | "warn" | "error" | "fatal";

export const LEVELS: Level[] = ["debug", "info", "warn", "error", "fatal"];

export interface Log {
  id: string;
  timestamp: string;
  level: Level;
  service: string;
  message: string;
  trace_id?: string;
  host?: string;
  attributes?: Record<string, string>;
}

export interface LogPage {
  logs: Log[];
  total: number;
}

export interface ServiceCount {
  service: string;
  count: number;
}

export interface HistogramBucket {
  bucket: string;
  counts: Partial<Record<Level, number>>;
}

export interface Stats {
  total: number;
  by_level: Partial<Record<Level, number>>;
  top_services: ServiceCount[];
  histogram: HistogramBucket[];
}

export interface SearchFilters {
  from: string;
  to: string;
  levels: Level[];
  services: string[];
  search: string;
  limit: number;
  offset: number;
  order: "asc" | "desc";
}

export const DEFAULT_FILTERS: SearchFilters = {
  from: "1h",
  to: "",
  levels: [],
  services: [],
  search: "",
  limit: 100,
  offset: 0,
  order: "desc",
};
