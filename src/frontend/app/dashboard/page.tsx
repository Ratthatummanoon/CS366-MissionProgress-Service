"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useConfig, getClient } from "@/lib/config-context";
import { MissionAssignment, MissionStatus } from "@/lib/types";
import Navbar from "@/components/Navbar";
import StatusBadge from "@/components/StatusBadge";
import ImpactBadge from "@/components/ImpactBadge";

const STATUS_FILTERS: (MissionStatus | "ALL")[] = [
  "ALL",
  "DISPATCHED",
  "EN_ROUTE",
  "ON_SITE",
  "NEED_BACKUP",
  "RESOLVED",
];

export default function DashboardPage() {
  const router = useRouter();
  const { config, isReady } = useConfig();
  const [missions, setMissions] = useState<MissionAssignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<MissionStatus | "ALL">("ALL");

  useEffect(() => {
    if (isReady && !config) {
      router.push("/");
    }
  }, [isReady, config, router]);

  useEffect(() => {
    if (!config) return;
    loadMissions();
  }, [config, filter]);

  async function loadMissions() {
    const client = getClient();
    if (!client) return;
    setLoading(true);
    setError("");
    try {
      const status = filter === "ALL" ? undefined : filter;
      const data = await client.listMissions(status);
      setMissions(data.missions || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "โหลดข้อมูลไม่สำเร็จ");
    } finally {
      setLoading(false);
    }
  }

  if (!isReady || !config) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  const statusCounts = missions.reduce(
    (acc, m) => {
      acc[m.current_status] = (acc[m.current_status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>,
  );

  return (
    <div className="min-h-screen flex flex-col">
      <Navbar />

      <main className="flex-1 max-w-7xl mx-auto w-full px-4 py-6">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">ภารกิจทั้งหมด</h1>
            <p className="text-sm text-gray-500 mt-1">
              ทีม {config.teamId} — {missions.length} ภารกิจ
            </p>
          </div>
          <button
            onClick={loadMissions}
            disabled={loading}
            className="flex items-center gap-2 bg-white border border-gray-300 rounded-lg px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition disabled:opacity-50"
          >
            <span className={loading ? "animate-spin" : ""}>🔄</span>
            รีเฟรช
          </button>
        </div>

        {/* Status summary cards */}
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 mb-6">
          {(
            [
              "DISPATCHED",
              "EN_ROUTE",
              "ON_SITE",
              "NEED_BACKUP",
              "RESOLVED",
            ] as MissionStatus[]
          ).map((s) => (
            <div
              key={s}
              onClick={() => setFilter(filter === s ? "ALL" : s)}
              className={`bg-white rounded-lg border p-3 cursor-pointer transition hover:shadow-sm ${
                filter === s
                  ? "border-blue-500 ring-1 ring-blue-500"
                  : "border-gray-200"
              }`}
            >
              <div className="text-2xl font-bold text-gray-900">
                {statusCounts[s] || 0}
              </div>
              <StatusBadge status={s} size="sm" />
            </div>
          ))}
        </div>

        {/* Filter bar */}
        <div className="flex gap-2 mb-4 flex-wrap">
          {STATUS_FILTERS.map((s) => (
            <button
              key={s}
              onClick={() => setFilter(s)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition ${
                filter === s
                  ? "bg-blue-600 text-white"
                  : "bg-white border border-gray-200 text-gray-600 hover:bg-gray-50"
              }`}
            >
              {s === "ALL" ? "ทั้งหมด" : s}
            </button>
          ))}
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-3 mb-4">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : missions.length === 0 ? (
          <div className="text-center py-20 text-gray-500">
            <span className="text-4xl block mb-3">📋</span>
            <p className="text-lg font-medium">ไม่พบภารกิจ</p>
            <p className="text-sm mt-1">
              {filter !== "ALL"
                ? `ไม่มีภารกิจสถานะ ${filter}`
                : "ยังไม่มีภารกิจที่มอบหมายให้ทีมนี้"}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {missions.map((m) => (
              <div
                key={m.mission_id}
                onClick={() =>
                  router.push(
                    `/mission?id=${encodeURIComponent(m.incident_id)}`,
                  )
                }
                className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-md hover:border-gray-300 transition cursor-pointer"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div>
                      <div className="font-semibold text-gray-900">
                        {m.incident_id}
                      </div>
                      <div className="text-xs text-gray-500 mt-0.5">
                        Mission: {m.mission_id}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <ImpactBadge level={m.latest_impact_level} />
                    <StatusBadge status={m.current_status} />
                  </div>
                </div>
                <div className="flex items-center gap-4 mt-3 text-xs text-gray-500">
                  <span>
                    เริ่ม:{" "}
                    {new Date(m.started_at).toLocaleString("th-TH", {
                      dateStyle: "short",
                      timeStyle: "short",
                    })}
                  </span>
                  <span>
                    อัปเดตล่าสุด:{" "}
                    {new Date(m.last_updated_at).toLocaleString("th-TH", {
                      dateStyle: "short",
                      timeStyle: "short",
                    })}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
