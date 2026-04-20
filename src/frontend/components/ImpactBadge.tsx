"use client";

import { IMPACT_LABELS, IMPACT_COLORS } from "@/lib/types";

interface ImpactBadgeProps {
  level: number;
}

export default function ImpactBadge({ level }: ImpactBadgeProps) {
  const label = IMPACT_LABELS[level] || `Level ${level}`;
  const color = IMPACT_COLORS[level] || "bg-gray-100 text-gray-800";
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${color}`}
    >
      Impact: {label}
    </span>
  );
}
