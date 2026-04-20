"use client";

import { MissionStatus, VALID_TRANSITIONS } from "@/lib/types";

const ALL_STATES: MissionStatus[] = [
  "DISPATCHED",
  "EN_ROUTE",
  "ON_SITE",
  "NEED_BACKUP",
  "RESOLVED",
];

const STATE_POSITIONS: Record<MissionStatus, { x: number; y: number }> = {
  DISPATCHED: { x: 50, y: 50 },
  EN_ROUTE: { x: 200, y: 50 },
  ON_SITE: { x: 350, y: 50 },
  NEED_BACKUP: { x: 350, y: 150 },
  RESOLVED: { x: 500, y: 100 },
};

interface StateMachineDiagramProps {
  currentStatus: MissionStatus;
}

export default function StateMachineDiagram({
  currentStatus,
}: StateMachineDiagramProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <h3 className="text-sm font-semibold text-gray-700 mb-3">
        State Machine
      </h3>
      <svg viewBox="0 0 600 200" className="w-full h-auto">
        <defs>
          <marker
            id="arrow"
            markerWidth="10"
            markerHeight="7"
            refX="10"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" fill="#9ca3af" />
          </marker>
          <marker
            id="arrow-active"
            markerWidth="10"
            markerHeight="7"
            refX="10"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" fill="#3b82f6" />
          </marker>
        </defs>

        {ALL_STATES.map((from) => {
          const targets = VALID_TRANSITIONS[from];
          return targets.map((to) => {
            const fromPos = STATE_POSITIONS[from];
            const toPos = STATE_POSITIONS[to];
            const isActive = from === currentStatus;
            const dx = toPos.x - fromPos.x;
            const dy = toPos.y - fromPos.y;
            const dist = Math.sqrt(dx * dx + dy * dy);
            const nx = dx / dist;
            const ny = dy / dist;
            const x1 = fromPos.x + nx * 50;
            const y1 = fromPos.y + ny * 18;
            const x2 = toPos.x - nx * 50;
            const y2 = toPos.y - ny * 18;

            return (
              <line
                key={`${from}-${to}`}
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke={isActive ? "#3b82f6" : "#d1d5db"}
                strokeWidth={isActive ? 2 : 1.5}
                markerEnd={isActive ? "url(#arrow-active)" : "url(#arrow)"}
              />
            );
          });
        })}

        {ALL_STATES.map((state) => {
          const pos = STATE_POSITIONS[state];
          const isCurrent = state === currentStatus;
          return (
            <g key={state}>
              <rect
                x={pos.x - 45}
                y={pos.y - 16}
                width={90}
                height={32}
                rx={6}
                fill={isCurrent ? "#3b82f6" : "#f3f4f6"}
                stroke={isCurrent ? "#2563eb" : "#d1d5db"}
                strokeWidth={isCurrent ? 2 : 1}
              />
              <text
                x={pos.x}
                y={pos.y + 4}
                textAnchor="middle"
                className="text-[10px] font-medium"
                fill={isCurrent ? "white" : "#374151"}
              >
                {state}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}
