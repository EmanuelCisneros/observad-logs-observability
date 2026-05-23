"use client";

import { useEffect, useState } from "react";
import { useTail } from "@/hooks/useTail";
import { Button, LiveDot, Input, LevelBadge } from "@/components/ui/primitives";
import { fmtTimestamp, levelColor } from "@/lib/utils";
import type { Level } from "@/types";

interface TailPanelProps {
  initialService?: string;
  initialLevel?: Level | "";
  onFiltersChange?: (service: string, level: Level | "") => void;
}

export function TailPanel({
  initialService = "",
  initialLevel = "",
  onFiltersChange,
}: TailPanelProps) {
  const [enabled, setEnabled] = useState(false);
  const [service, setService] = useState(initialService);
  const [level, setLevel] = useState<Level | "">(initialLevel);

  useEffect(() => {
    setService(initialService);
    setLevel(initialLevel);
  }, [initialService, initialLevel]);

  const { logs, connected, error, clear } = useTail({
    service: service || undefined,
    level: (level as Level) || undefined,
    enabled,
  });

  function updateService(value: string) {
    setService(value);
    onFiltersChange?.(value, level);
  }

  function updateLevel(value: Level | "") {
    setLevel(value);
    onFiltersChange?.(service, value);
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-ink-500 bg-ink-700 p-3">
        <LiveDot active={connected} />

        <Input
          placeholder="service filter"
          value={service}
          onChange={(e) => updateService(e.target.value)}
          className="h-7 w-32 text-xs"
          disabled={enabled}
        />

        <select
          value={level}
          onChange={(e) => updateLevel(e.target.value as Level | "")}
          disabled={enabled}
          className="h-7 rounded border border-ink-500 bg-ink-700 px-2 font-mono text-xs text-mist-100 focus:outline-none"
        >
          <option value="">all levels</option>
          {(["debug", "info", "warn", "error", "fatal"] as Level[]).map((l) => (
            <option key={l} value={l}>{l}</option>
          ))}
        </select>

        <Button
          variant={enabled ? "danger" : "primary"}
          size="sm"
          onClick={() => {
            if (enabled) { clear(); }
            setEnabled((v) => !v);
          }}
        >
          {enabled ? "Stop" : "Start"} live tail
        </Button>

        {logs.length > 0 && (
          <Button variant="ghost" size="sm" onClick={clear}>
            Clear
          </Button>
        )}

        <span className="ml-auto font-mono text-[11px] text-mist-400">
          {logs.length} lines
        </span>
      </div>

      {error && (
        <p className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-400">
          {error}
        </p>
      )}

      <div className="h-[460px] overflow-auto rounded-lg border border-ink-500 bg-ink-900 font-mono text-xs">
        {!enabled && logs.length === 0 && (
          <div className="flex h-full items-center justify-center text-mist-400">
            Press <span className="mx-1 rounded bg-ink-600 px-1.5 py-0.5">Start live tail</span> to stream logs
          </div>
        )}

        {logs.map((l, i) => (
          <div
            key={l.id + i}
            className="flex items-start gap-3 border-b border-ink-700 px-3 py-1.5 last:border-0 hover:bg-ink-800/50"
          >
            <span className="shrink-0 tabular-nums text-mist-400">
              {fmtTimestamp(l.timestamp)}
            </span>
            <LevelBadge level={l.level} className="shrink-0 self-center" />
            <span className="shrink-0 text-signal/70">{l.service}</span>
            <span className={`flex-1 break-all ${levelColor(l.level)}`}>
              {l.message}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
