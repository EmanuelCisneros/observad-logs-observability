"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import type { Level, Log } from "@/types";

const MAX_TAIL_LINES = 500;

interface TailOptions {
  service?: string;
  level?: Level;
  enabled?: boolean;
}

interface TailState {
  logs: Log[];
  connected: boolean;
  error: string | null;
}

export function useTail({ service, level, enabled = true }: TailOptions) {
  const [state, setState] = useState<TailState>({
    logs: [],
    connected: false,
    error: null,
  });

  const wsRef     = useRef<WebSocket | null>(null);
  const retryRef  = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attempt   = useRef(0);
  const unmounted = useRef(false);

  const connect = useCallback(() => {
    if (!enabled || unmounted.current) return;

    const url = api.tailUrl(service, level);
    const ws  = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      if (unmounted.current) { ws.close(); return; }
      attempt.current = 0;
      setState((s) => ({ ...s, connected: true, error: null }));
    };

    ws.onmessage = (evt) => {
      if (unmounted.current) return;
      try {
        const log = JSON.parse(evt.data as string) as Log;
        setState((s) => ({
          ...s,
          logs: [log, ...s.logs].slice(0, MAX_TAIL_LINES),
        }));
      } catch {
        // ignore malformed frames
      }
    };

    ws.onerror = () => {};

    ws.onclose = (evt) => {
      if (unmounted.current) return;
      setState((s) => ({ ...s, connected: false }));

      if (evt.code === 1000 || !enabled) return;

      const delay = Math.min(1000 * 2 ** attempt.current, 30_000);
      attempt.current += 1;
      setState((s) => ({
        ...s,
        error: `Disconnected. Reconnecting in ${Math.round(delay / 1000)}s…`,
      }));
      retryRef.current = setTimeout(connect, delay);
    };
  }, [service, level, enabled]);

  useEffect(() => {
    if (enabled) {
      connect();
    }
    return () => {
      unmounted.current = true;
      if (retryRef.current) clearTimeout(retryRef.current);
      wsRef.current?.close(1000);
    };
  }, [connect, enabled]);

  const clear = useCallback(() => {
    setState((s) => ({ ...s, logs: [] }));
  }, []);

  return { ...state, clear };
}
