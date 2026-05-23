"use client";

import { useCallback, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  DEFAULT_FILTERS,
  LEVELS,
  type Level,
  type SearchFilters,
} from "@/types";

const LEVEL_SET = new Set<string>(LEVELS);

function parseLevels(raw: string | null): Level[] {
  if (!raw) return [];
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter((s): s is Level => LEVEL_SET.has(s));
}

function parseIntParam(raw: string | null, fallback: number): number {
  if (!raw) return fallback;
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) ? n : fallback;
}

export function filtersFromParams(sp: URLSearchParams): SearchFilters {
  const order = sp.get("order");
  return {
    from:     sp.get("from") ?? DEFAULT_FILTERS.from,
    to:       sp.get("to") ?? "",
    levels:   parseLevels(sp.get("levels")),
    services: sp.get("services")?.split(",").map((s) => s.trim()).filter(Boolean) ?? [],
    search:   sp.get("search") ?? "",
    limit:    parseIntParam(sp.get("limit"), DEFAULT_FILTERS.limit),
    offset:   parseIntParam(sp.get("offset"), 0),
    order:    order === "asc" ? "asc" : "desc",
  };
}

export function filtersToSearchParams(f: SearchFilters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  if (f.levels.length) p.set("levels", f.levels.join(","));
  if (f.services.length) p.set("services", f.services.join(","));
  if (f.search.trim()) p.set("search", f.search.trim());
  if (f.limit !== DEFAULT_FILTERS.limit) p.set("limit", String(f.limit));
  if (f.offset > 0) p.set("offset", String(f.offset));
  if (f.order !== "desc") p.set("order", f.order);
  return p;
}

export function useFilters(defaults?: Partial<SearchFilters>) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const filters = useMemo(() => {
    const base = filtersFromParams(searchParams);
    return { ...DEFAULT_FILTERS, ...defaults, ...base };
  }, [searchParams, defaults]);

  const setFilters = useCallback(
    (next: SearchFilters | ((prev: SearchFilters) => SearchFilters)) => {
      const resolved =
        typeof next === "function" ? next(filters) : next;
      const qs = filtersToSearchParams(resolved).toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [filters, pathname, router],
  );

  const patchFilters = useCallback(
    (patch: Partial<SearchFilters>) => {
      setFilters({ ...filters, ...patch });
    },
    [filters, setFilters],
  );

  return { filters, setFilters, patchFilters };
}

export function tailParamsFromURL(sp: URLSearchParams) {
  const service = sp.get("tail_service") ?? "";
  const levelRaw = sp.get("tail_level") ?? "";
  const level = LEVEL_SET.has(levelRaw) ? (levelRaw as Level) : "";
  return { service, level: level as Level | "" };
}

export function mergeTailParams(
  current: URLSearchParams,
  service: string,
  level: Level | "",
): URLSearchParams {
  const p = new URLSearchParams(current.toString());
  if (service) p.set("tail_service", service);
  else p.delete("tail_service");
  if (level) p.set("tail_level", level);
  else p.delete("tail_level");
  return p;
}
