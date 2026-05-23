"use client";

import { compact, levelBg } from "@/lib/utils";
import { Card, Spinner } from "@/components/ui/primitives";
import type { Stats } from "@/types";
import { LEVELS } from "@/types";

interface StatsCardsProps {
  stats?: Stats;
  isLoading: boolean;
}

export function StatsCards({ stats, isLoading }: StatsCardsProps) {
  if (isLoading && !stats) {
    return (
      <div className="flex h-24 items-center justify-center">
        <Spinner className="h-5 w-5 text-signal" />
      </div>
    );
  }
  if (!stats) return null;

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
      <Card className="col-span-2 sm:col-span-1">
        <p className="font-mono text-[11px] uppercase tracking-widest text-mist-400">
          Total
        </p>
        <p className="mt-1 font-mono text-2xl font-bold text-mist-100">
          {compact(stats.total)}
        </p>
      </Card>

      {LEVELS.map((lvl) => {
        const count = stats.by_level[lvl] ?? 0;
        return (
          <Card key={lvl} className="relative overflow-hidden">
            <span
              className={`absolute inset-y-0 left-0 w-0.5 rounded-l ${levelBg(lvl).split(" ")[0]}`}
            />
            <p className="font-mono text-[11px] uppercase tracking-widest text-mist-400">
              {lvl}
            </p>
            <p
              className={`mt-1 font-mono text-2xl font-bold ${
                count > 0 ? "text-mist-100" : "text-ink-400"
              }`}
            >
              {compact(count)}
            </p>
          </Card>
        );
      })}
    </div>
  );
}
