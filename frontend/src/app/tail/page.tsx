"use client";

import { Suspense, useCallback } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { TailPanel } from "@/components/logs/TailPanel";
import { mergeTailParams, tailParamsFromURL } from "@/hooks/useFilters";
import type { Level } from "@/types";

function TailContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { service, level } = tailParamsFromURL(searchParams);

  const onTailFiltersChange = useCallback(
    (nextService: string, nextLevel: Level | "") => {
      const qs = mergeTailParams(searchParams, nextService, nextLevel).toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams],
  );

  return (
    <TailPanel
      initialService={service}
      initialLevel={level}
      onFiltersChange={onTailFiltersChange}
    />
  );
}

export default function TailPage() {
  return (
    <div className="flex flex-col gap-4 p-6">
      <div>
        <h1 className="font-sans text-xl font-semibold text-mist-100">
          Live Tail
        </h1>
        <p className="mt-0.5 font-mono text-xs text-mist-400">
          Real-time log stream via WebSocket — up to 500 lines in buffer
        </p>
      </div>
      <Suspense fallback={<div className="font-mono text-sm text-mist-400">Loading…</div>}>
        <TailContent />
      </Suspense>
    </div>
  );
}
