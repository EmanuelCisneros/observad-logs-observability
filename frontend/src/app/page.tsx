"use client";

import { Suspense } from "react";
import { useStats } from "@/hooks/useStats";
import { useFilters } from "@/hooks/useFilters";
import { StatsCards } from "@/components/charts/StatsCards";
import { VolumeHistogram } from "@/components/charts/VolumeHistogram";
import { TopServices } from "@/components/charts/TopServices";
import { ApiErrorBanner } from "@/components/ui/ApiErrorBanner";
import { Select } from "@/components/ui/primitives";

const TIME_OPTIONS = [
  { label: "Last 15 minutes", value: "15m" },
  { label: "Last 1 hour",     value: "1h" },
  { label: "Last 3 hours",    value: "3h" },
  { label: "Last 6 hours",    value: "6h" },
  { label: "Last 24 hours",   value: "24h" },
  { label: "Last 7 days",     value: "168h" },
];

function DashboardContent() {
  const { filters, patchFilters } = useFilters({ from: "1h" });
  const { data: stats, isLoading, isError, error, dataUpdatedAt } = useStats(filters);

  const updatedAt = dataUpdatedAt
    ? new Date(dataUpdatedAt).toLocaleTimeString()
    : null;

  return (
    <div className="flex flex-col gap-5 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="font-sans text-xl font-semibold text-mist-100">
            Dashboard
          </h1>
          <p className="mt-0.5 font-mono text-xs text-mist-400">
            {updatedAt ? `Updated at ${updatedAt}` : "Loading…"}
          </p>
        </div>
        <Select
          value={filters.from}
          onChange={(v) => patchFilters({ from: v, offset: 0 })}
          options={TIME_OPTIONS}
          className="w-44"
        />
      </div>

      <ApiErrorBanner error={isError ? error : null} />

      <StatsCards stats={stats} isLoading={isLoading} />

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div className="xl:col-span-2">
          <VolumeHistogram stats={stats} />
        </div>
        <div>
          <TopServices stats={stats} />
        </div>
      </div>

      {!isLoading && !isError && stats && stats.total === 0 && (
        <div className="rounded-lg border border-signal/30 bg-signal/5 p-5">
          <h2 className="font-mono text-sm font-semibold text-signal">
            No logs yet — send your first batch
          </h2>
          <pre className="mt-3 overflow-auto rounded bg-ink-900 p-3 font-mono text-xs text-mist-200">
{`curl -X POST http://localhost:8080/api/v1/logs \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: dev-local-key" \\
  -d '{
    "logs": [
      {
        "service": "my-api",
        "level":   "info",
        "message": "server started on :3000"
      }
    ]
  }'`}
          </pre>
        </div>
      )}
    </div>
  );
}

export default function DashboardPage() {
  return (
    <Suspense fallback={<div className="p-6 font-mono text-sm text-mist-400">Loading…</div>}>
      <DashboardContent />
    </Suspense>
  );
}
