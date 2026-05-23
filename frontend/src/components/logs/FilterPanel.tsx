"use client";

import { useCallback, useState } from "react";
import { Button, Input } from "@/components/ui/primitives";
import { cn, levelBg } from "@/lib/utils";
import { LEVELS, type Level, type SearchFilters } from "@/types";

const TIME_PRESETS = [
  { label: "15m", value: "15m" },
  { label: "1h",  value: "1h" },
  { label: "3h",  value: "3h" },
  { label: "6h",  value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d",  value: "168h" },
];

interface FilterPanelProps {
  filters: SearchFilters;
  onChange: (f: SearchFilters) => void;
  isLoading?: boolean;
}

export function FilterPanel({ filters, onChange, isLoading }: FilterPanelProps) {
  const [search, setSearch] = useState(filters.search);

  const set = useCallback(
    <K extends keyof SearchFilters>(key: K, value: SearchFilters[K]) => {
      onChange({ ...filters, [key]: value, offset: 0 });
    },
    [filters, onChange],
  );

  function toggleLevel(lvl: Level) {
    const next = filters.levels.includes(lvl)
      ? filters.levels.filter((l) => l !== lvl)
      : [...filters.levels, lvl];
    set("levels", next);
  }

  function submitSearch(e: React.FormEvent) {
    e.preventDefault();
    set("search", search);
  }

  return (
    <div className="space-y-3 rounded-lg border border-ink-500 bg-ink-700 p-3">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="font-mono text-[11px] uppercase tracking-widest text-mist-400">
          Range
        </span>
        {TIME_PRESETS.map((p) => (
          <button
            key={p.value}
            onClick={() => set("from", p.value)}
            className={cn(
              "rounded border px-2 py-0.5 font-mono text-xs transition-colors",
              filters.from === p.value
                ? "border-signal bg-signal/10 text-signal"
                : "border-ink-500 text-mist-300 hover:border-ink-400",
            )}
          >
            {p.label}
          </button>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <span className="font-mono text-[11px] uppercase tracking-widest text-mist-400">
          Level
        </span>
        {LEVELS.map((lvl) => {
          const active = filters.levels.includes(lvl);
          return (
            <button
              key={lvl}
              onClick={() => toggleLevel(lvl)}
              className={cn(
                "rounded border px-2 py-0.5 font-mono text-[11px] uppercase tracking-wider transition-all",
                active
                  ? cn("border-transparent", levelBg(lvl))
                  : "border-ink-500 text-mist-400 hover:border-ink-400 hover:text-mist-300",
              )}
            >
              {lvl}
            </button>
          );
        })}
        {filters.levels.length > 0 && (
          <button
            onClick={() => set("levels", [])}
            className="font-mono text-[11px] text-mist-400 hover:text-red-400"
          >
            ✕ clear
          </button>
        )}
      </div>

      <form onSubmit={submitSearch} className="flex gap-2">
        <Input
          placeholder="Search message…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-8 text-xs"
        />
        <Button
          type="submit"
          variant="primary"
          size="sm"
          disabled={isLoading}
          className="shrink-0"
        >
          Search
        </Button>
        {(filters.search || filters.levels.length || filters.services.length) && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              setSearch("");
              onChange({
                ...filters,
                search: "",
                levels: [],
                services: [],
                offset: 0,
              });
            }}
          >
            Reset
          </Button>
        )}
      </form>
    </div>
  );
}
