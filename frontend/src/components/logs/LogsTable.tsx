"use client";

import { useState } from "react";
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  createColumnHelper,
  type ColumnDef,
} from "@tanstack/react-table";
import { LevelBadge, Spinner } from "@/components/ui/primitives";
import { fmtTimestamp, cn } from "@/lib/utils";
import type { Log, Level } from "@/types";

const columnHelper = createColumnHelper<Log>();

const COLUMNS: ColumnDef<Log, string>[] = [
  columnHelper.accessor("timestamp", {
    header: "Time",
    size: 130,
    cell: (c) => (
      <span className="font-mono text-[11px] tabular-nums text-mist-300">
        {fmtTimestamp(c.getValue())}
      </span>
    ),
  }),
  columnHelper.accessor("level", {
    header: "Level",
    size: 72,
    cell: (c) => <LevelBadge level={c.getValue() as Level} />,
  }),
  columnHelper.accessor("service", {
    header: "Service",
    size: 120,
    cell: (c) => (
      <span className="font-mono text-xs text-signal/80">{c.getValue()}</span>
    ),
  }),
  columnHelper.accessor("message", {
    header: "Message",
    cell: (c) => (
      <span className="line-clamp-2 font-mono text-xs text-mist-100">
        {c.getValue()}
      </span>
    ),
  }),
];

interface LogsTableProps {
  logs: Log[];
  total: number;
  offset: number;
  limit: number;
  isLoading: boolean;
  onPageChange: (offset: number) => void;
}

export function LogsTable({
  logs,
  total,
  offset,
  limit,
  isLoading,
  onPageChange,
}: LogsTableProps) {
  const [expanded, setExpanded] = useState<string | null>(null);

  const table = useReactTable({
    data: logs,
    columns: COLUMNS as ColumnDef<Log>[],
    getCoreRowModel: getCoreRowModel(),
  });

  const pageCount = Math.max(1, Math.ceil(total / limit));
  const page      = Math.floor(offset / limit);

  return (
    <div className="flex flex-col gap-0 overflow-hidden rounded-lg border border-ink-500">
      <div className="border-b border-ink-500 bg-ink-600">
        {table.getHeaderGroups().map((hg) => (
          <div key={hg.id} className="flex items-center px-3 py-1.5">
            {hg.headers.map((h) => (
              <div
                key={h.id}
                className="shrink-0 font-mono text-[10px] uppercase tracking-widest text-mist-400"
                style={{ width: h.column.columnDef.size, flex: h.column.columnDef.size ? undefined : 1 }}
              >
                {flexRender(h.column.columnDef.header, h.getContext())}
              </div>
            ))}
          </div>
        ))}
      </div>

      {isLoading && logs.length === 0 && (
        <div className="flex h-40 items-center justify-center">
          <Spinner className="h-5 w-5 text-signal" />
        </div>
      )}

      {!isLoading && logs.length === 0 && (
        <div className="flex h-40 items-center justify-center font-mono text-sm text-mist-400">
          No logs match the current filters
        </div>
      )}

      <div className="divide-y divide-ink-600 overflow-auto">
        {table.getRowModel().rows.map((row) => {
          const log       = row.original;
          const isOpen    = expanded === log.id;
          const hasAttrs  = log.attributes && Object.keys(log.attributes).length > 0;

          return (
            <div key={row.id}>
              <div
                role="button"
                onClick={() => setExpanded(isOpen ? null : log.id)}
                className={cn(
                  "flex cursor-pointer items-start gap-0 px-3 py-2 transition-colors",
                  isOpen ? "bg-ink-600" : "hover:bg-ink-600/50",
                  isLoading && "opacity-50",
                )}
              >
                {row.getVisibleCells().map((cell) => (
                  <div
                    key={cell.id}
                    className="shrink-0 py-0.5"
                    style={{
                      width: cell.column.columnDef.size,
                      flex: cell.column.columnDef.size ? undefined : 1,
                    }}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </div>
                ))}
              </div>

              {isOpen && (
                <div className="animate-fade-in border-t border-ink-500 bg-ink-800 px-4 py-3">
                  <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs sm:grid-cols-4">
                    <Detail label="ID"       value={log.id} />
                    <Detail label="Timestamp" value={log.timestamp} />
                    <Detail label="Service"  value={log.service} />
                    <Detail label="Host"     value={log.host ?? "—"} />
                    {log.trace_id && (
                      <Detail label="Trace ID" value={log.trace_id} />
                    )}
                  </div>

                  <p className="mt-3 rounded bg-ink-900 px-3 py-2 font-mono text-xs text-mist-100">
                    {log.message}
                  </p>

                  {hasAttrs && (
                    <div className="mt-2 grid grid-cols-2 gap-1 sm:grid-cols-3">
                      {Object.entries(log.attributes!).map(([k, v]) => (
                        <div key={k} className="font-mono text-[11px]">
                          <span className="text-mist-400">{k}:</span>{" "}
                          <span className="text-mist-200">{v}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="flex items-center justify-between border-t border-ink-500 bg-ink-600 px-3 py-2">
        <span className="font-mono text-[11px] text-mist-400">
          {total.toLocaleString()} total · page {page + 1} / {pageCount}
        </span>
        <div className="flex gap-2">
          <button
            disabled={page === 0}
            onClick={() => onPageChange(Math.max(0, offset - limit))}
            className="font-mono text-xs text-mist-300 disabled:opacity-30 hover:text-mist-100"
          >
            ← Prev
          </button>
          <button
            disabled={page >= pageCount - 1}
            onClick={() => onPageChange(offset + limit)}
            className="font-mono text-xs text-mist-300 disabled:opacity-30 hover:text-mist-100"
          >
            Next →
          </button>
        </div>
      </div>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="font-mono">
      <span className="text-mist-400">{label}: </span>
      <span className="text-mist-200">{value}</span>
    </div>
  );
}
