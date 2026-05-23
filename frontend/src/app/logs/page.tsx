"use client";

import { Suspense } from "react";
import { useSearch } from "@/hooks/useSearch";
import { useStats } from "@/hooks/useStats";
import { useFilters } from "@/hooks/useFilters";
import { FilterPanel } from "@/components/logs/FilterPanel";
import { LogsTable } from "@/components/logs/LogsTable";
import { ApiErrorBanner } from "@/components/ui/ApiErrorBanner";

function LogsContent() {
  const { filters, setFilters } = useFilters();
  const { data: page, isLoading, isFetching, isError, error } = useSearch(filters);
  const { data: stats } = useStats(filters);

  function handlePageChange(offset: number) {
    setFilters((f) => ({ ...f, offset }));
  }

  return (
    <div className="flex flex-col gap-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="font-sans text-xl font-semibold text-mist-100">
          Log Explorer
        </h1>
        {stats && (
          <span className="font-mono text-xs text-mist-400">
            {stats.total.toLocaleString()} events in range
          </span>
        )}
      </div>

      <ApiErrorBanner error={isError ? error : null} />

      <FilterPanel
        filters={filters}
        onChange={setFilters}
        isLoading={isFetching}
      />

      <LogsTable
        logs={page?.logs ?? []}
        total={page?.total ?? 0}
        offset={filters.offset}
        limit={filters.limit}
        isLoading={isLoading}
        onPageChange={handlePageChange}
      />
    </div>
  );
}

export default function LogsPage() {
  return (
    <Suspense fallback={<div className="p-6 font-mono text-sm text-mist-400">Loading…</div>}>
      <LogsContent />
    </Suspense>
  );
}
