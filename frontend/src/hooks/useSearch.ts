"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { SearchFilters } from "@/types";

export function useSearch(filters: SearchFilters) {
  return useQuery({
    queryKey: ["logs", "search", filters],
    queryFn: () => api.searchLogs(filters),
    placeholderData: (prev) => prev,
  });
}
