"use client";

import { MissionStatus, STATUS_LABELS, STATUS_COLORS } from "@/lib/types";

interface StatusBadgeProps {
  status: MissionStatus;
  size?: "sm" | "md";
}

export default function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const sizeClass = size === "sm" ? "px-2 py-0.5 text-xs" : "px-3 py-1 text-sm";
  return (
    <span
      className={`inline-flex items-center rounded-full border font-medium ${STATUS_COLORS[status]} ${sizeClass}`}
    >
      {STATUS_LABELS[status]}
    </span>
  );
}
