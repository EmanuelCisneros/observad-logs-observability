"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { filtersToSearchParams, filtersFromParams } from "@/hooks/useFilters";

const NAV = [
  {
    href: "/",
    label: "Dashboard",
    icon: (
      <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
        <path d="M2 11a1 1 0 011-1h2a1 1 0 011 1v5a1 1 0 01-1 1H3a1 1 0 01-1-1v-5zM8 7a1 1 0 011-1h2a1 1 0 011 1v9a1 1 0 01-1 1H9a1 1 0 01-1-1V7zM14 4a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1h-2a1 1 0 01-1-1V4z" />
      </svg>
    ),
  },
  {
    href: "/logs",
    label: "Logs",
    icon: (
      <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
        <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h8a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
      </svg>
    ),
  },
  {
    href: "/tail",
    label: "Live Tail",
    icon: (
      <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
      </svg>
    ),
  },
];

export function Sidebar() {
  const path = usePathname();
  const searchParams = useSearchParams();
  const sharedQS = filtersToSearchParams(filtersFromParams(searchParams)).toString();

  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-ink-500 bg-ink-800">
      <div className="flex h-14 items-center gap-2.5 border-b border-ink-500 px-4">
        <span className="flex h-7 w-7 items-center justify-center rounded bg-signal text-ink-900 font-mono text-sm font-bold">
          O
        </span>
        <span className="font-sans text-sm font-semibold tracking-wide text-mist-100">
          observa
        </span>
        <span className="ml-auto rounded bg-ink-600 px-1.5 py-0.5 font-mono text-[10px] text-mist-400">
          MVP
        </span>
      </div>

      <nav className="flex flex-col gap-0.5 p-2 pt-3">
        {NAV.map((item) => {
          const active = path === item.href;
          const href = sharedQS && item.href !== "/tail"
            ? `${item.href}?${sharedQS}`
            : item.href;
          return (
            <Link
              key={item.href}
              href={href}
              className={cn(
                "flex items-center gap-2.5 rounded px-3 py-2 font-sans text-sm transition-colors",
                active
                  ? "bg-signal/10 text-signal"
                  : "text-mist-300 hover:bg-ink-600 hover:text-mist-100",
              )}
            >
              {item.icon}
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="mt-auto border-t border-ink-500 p-3">
        <p className="font-mono text-[10px] text-ink-400">
          API: {process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080"}
        </p>
      </div>
    </aside>
  );
}
