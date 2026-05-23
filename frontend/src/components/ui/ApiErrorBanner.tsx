"use client";

import { ApiError } from "@/lib/api";

interface ApiErrorBannerProps {
  error: Error | null;
  title?: string;
}

export function ApiErrorBanner({ error, title = "API error" }: ApiErrorBannerProps) {
  if (!error) return null;

  const detail =
    error instanceof ApiError
      ? error.status === 0
        ? "Cannot reach the backend — is observad running?"
        : error.message
      : error.message;

  return (
    <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3">
      <p className="font-mono text-sm font-semibold text-red-400">{title}</p>
      <p className="mt-1 font-mono text-xs text-red-300/90">{detail}</p>
      {error instanceof ApiError && error.status === 401 && (
        <p className="mt-2 font-mono text-[11px] text-mist-400">
          Set <code className="text-mist-200">NEXT_PUBLIC_API_KEY</code> in{" "}
          <code className="text-mist-200">frontend/.env.local</code> to match{" "}
          <code className="text-mist-200">OBSERVA_INGEST_API_KEY</code>.
        </p>
      )}
    </div>
  );
}
