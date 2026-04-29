"use client";

import { useState, useEffect, useCallback } from "react";

interface ServiceHealth {
  name: string;
  status: "available" | "unavailable" | "not_configured";
}

interface HealthData {
  services: ServiceHealth[];
  checked_at: string;
}

interface Props {
  /** Base URL ของ API (อาจเป็นค่าว่างถ้ายังไม่ได้กรอก) */
  apiUrl: string;
}

type StatusKey =
  | "available"
  | "unavailable"
  | "not_configured"
  | "checking"
  | "idle";

const STATUS_CONFIG: Record<
  StatusKey,
  { dot: string; text: string; labelClass: string }
> = {
  available: {
    dot: "bg-green-500",
    text: "available",
    labelClass: "text-green-700",
  },
  unavailable: {
    dot: "bg-red-500",
    text: "unavailable",
    labelClass: "text-red-600",
  },
  not_configured: {
    dot: "bg-gray-300",
    text: "not set",
    labelClass: "text-gray-400",
  },
  checking: {
    dot: "bg-yellow-400 animate-pulse",
    text: "checking…",
    labelClass: "text-yellow-600",
  },
  idle: { dot: "bg-gray-300", text: "—", labelClass: "text-gray-400" },
};

export default function ServiceStatusBar({ apiUrl }: Props) {
  const [selfStatus, setSelfStatus] = useState<StatusKey>("idle");
  const [depServices, setDepServices] = useState<ServiceHealth[]>([]);
  const [checkedAt, setCheckedAt] = useState<string>("");

  const doCheck = useCallback(async (url: string) => {
    if (!url) {
      setSelfStatus("idle");
      setDepServices([]);
      return;
    }
    try {
      new URL(url);
    } catch {
      setSelfStatus("unavailable");
      setDepServices([]);
      return;
    }

    setSelfStatus("checking");
    try {
      const clean = url.replace(/\/+$/, "");
      const res = await fetch(`${clean}/health`, {
        signal: AbortSignal.timeout(5000),
      });
      if (res.ok) {
        const data: HealthData = await res.json();
        setSelfStatus("available");
        setDepServices(data.services ?? []);
        setCheckedAt(data.checked_at ?? "");
      } else {
        setSelfStatus("unavailable");
        setDepServices([]);
      }
    } catch {
      setSelfStatus("unavailable");
      setDepServices([]);
    }
  }, []);

  // debounce 600 ms เมื่อ apiUrl เปลี่ยน
  useEffect(() => {
    if (!apiUrl) {
      setSelfStatus("idle");
      setDepServices([]);
      return;
    }
    const t = setTimeout(() => doCheck(apiUrl), 600);
    return () => clearTimeout(t);
  }, [apiUrl, doCheck]);

  if (selfStatus === "idle") return null;

  const rows: { name: string; status: StatusKey }[] = [
    { name: "MissionProgress", status: selfStatus },
    ...depServices.map((s) => ({
      name: s.name,
      status: s.status as StatusKey,
    })),
  ];

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs py-1">
      {rows.map((svc) => {
        const cfg = STATUS_CONFIG[svc.status] ?? STATUS_CONFIG.unavailable;
        return (
          <span key={svc.name} className="flex items-center gap-1.5">
            <span className={`w-2 h-2 rounded-full flex-shrink-0 ${cfg.dot}`} />
            <span className="text-gray-600">{svc.name}</span>
            <span className={`font-medium ${cfg.labelClass}`}>{cfg.text}</span>
          </span>
        );
      })}
      {checkedAt && selfStatus === "available" && (
        <span className="text-gray-400 ml-1">
          · ตรวจสอบล่าสุด {new Date(checkedAt).toLocaleTimeString("th-TH")}
        </span>
      )}
    </div>
  );
}
