"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { SearchFilters } from "@/types";

export function useStats(filters: SearchFilters) {
  return useQuery({
    queryKey: ["logs", "stats", filters],
    queryFn: () => api.getStats(filters),
    staleTime: 15_000,
    refetchInterval: 15_000,
    placeholderData: (prev) => prev,
  });
}
