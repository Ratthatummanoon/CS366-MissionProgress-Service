"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/config-context";
import { RescueTeam, RescueTeamListResponse } from "@/lib/types";

// Use explicit if-branches so Tailwind class scanner can find all class strings
function getTeamStatusInfo(status: string) {
  if (status === "AVAILABLE") {
    return {
      dot: "bg-green-500 ring-2 ring-green-300",
      badge: "bg-green-500 text-white",
      label: "พร้อมปฏิบัติการ",
      desc: "ทีมว่าง รับภารกิจได้ทันที",
    };
  }
  if (status === "BUSY") {
    return {
      dot: "bg-amber-500 ring-2 ring-amber-300",
      badge: "bg-amber-500 text-white",
      label: "กำลังปฏิบัติการ",
      desc: "ทีมกำลังอยู่ในภารกิจ",
    };
  }
  return {
    dot: "bg-gray-400 ring-2 ring-gray-200",
    badge: "bg-gray-500 text-white",
    label: status === "OFFLINE" ? "ออฟไลน์" : status,
    desc: status === "OFFLINE" ? "ไม่สามารถติดต่อทีมได้" : "",
  };
}

function getMissionStatusInfo(status: string) {
  if (status === "DISPATCHED")
    return {
      badge: "bg-gray-600 text-white",
      icon: "📋",
      label: "รอเดินทาง",
    };
  if (status === "EN_ROUTE")
    return {
      badge: "bg-blue-600 text-white",
      icon: "🚗",
      label: "กำลังเดินทาง",
    };
  if (status === "ON_SITE")
    return {
      badge: "bg-yellow-500 text-white",
      icon: "📍",
      label: "ถึงจุดเกิดเหตุ",
    };
  if (status === "NEED_BACKUP")
    return {
      badge: "bg-red-600 text-white",
      icon: "🆘",
      label: "ต้องการกำลังเสริม",
    };
  if (status === "RESOLVED")
    return {
      badge: "bg-green-600 text-white",
      icon: "✅",
      label: "เสร็จสิ้น",
    };
  return {
    badge: "bg-gray-500 text-white",
    icon: "•",
    label: status,
  };
}

export default function TeamsPage() {
  const router = useRouter();
  const { config, setConfig, logout, isReady } = useConfig();

  const [teams, setTeams] = useState<RescueTeam[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Mission status per team: teamId → active current_status | null
  const [missionStatuses, setMissionStatuses] = useState<
    Record<string, string | null>
  >({});
  const [missionLoading, setMissionLoading] = useState(false);

  // Manual fallback if RescueTeam URL not configured
  const [manualTeamId, setManualTeamId] = useState("");
  const [manualError, setManualError] = useState("");

  const fetchTeams = useCallback(async (rescueTeamUrl: string) => {
    setLoading(true);
    setError("");
    try {
      const clean = rescueTeamUrl.replace(/\/+$/, "");
      const res = await fetch(`${clean}/v1/teams`, {
        headers: { Authorization: "Bearer mock-dispatcher-token-123" },
        signal: AbortSignal.timeout(8000),
      });
      if (!res.ok) {
        setError(`โหลดรายการทีมไม่สำเร็จ (${res.status})`);
        setTeams([]);
        return;
      }
      const data: RescueTeamListResponse = await res.json();
      setTeams(data.teams ?? []);
    } catch {
      setError("เชื่อมต่อ RescueTeam Service ไม่สำเร็จ");
      setTeams([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch active mission status for each team from MissionProgress API
  const fetchMissionStatuses = useCallback(
    async (teamList: RescueTeam[], apiUrl: string, apiKey: string) => {
      if (!apiUrl || !apiKey || teamList.length === 0) return;
      setMissionLoading(true);
      const clean = apiUrl.replace(/\/+$/, "");
      const results = await Promise.allSettled(
        teamList.map(async (team) => {
          try {
            const res = await fetch(`${clean}/missions`, {
              headers: {
                "x-api-key": apiKey,
                "X-Rescue-Team-ID": team.team_id,
              },
              signal: AbortSignal.timeout(5000),
            });
            if (!res.ok) return { teamId: team.team_id, status: null };
            const data = await res.json();
            const missions: Array<{ current_status: string }> =
              data.missions ?? [];
            const active = missions.find(
              (m) => m.current_status !== "RESOLVED",
            );
            return {
              teamId: team.team_id,
              status: active?.current_status ?? null,
            };
          } catch {
            return { teamId: team.team_id, status: null };
          }
        }),
      );
      const map: Record<string, string | null> = {};
      for (const r of results) {
        if (r.status === "fulfilled") {
          map[r.value.teamId] = r.value.status;
        }
      }
      setMissionStatuses(map);
      setMissionLoading(false);
    },
    [],
  );

  useEffect(() => {
    if (!isReady) return;
    if (!config?.apiUrl || !config?.apiKey) {
      router.push("/");
      return;
    }
    if (config.rescueTeamUrl) {
      fetchTeams(config.rescueTeamUrl);
    }
  }, [
    isReady,
    config?.apiUrl,
    config?.apiKey,
    config?.rescueTeamUrl,
    router,
    fetchTeams,
  ]);

  // After teams load, fetch mission statuses from MissionProgress
  useEffect(() => {
    if (teams.length > 0 && config?.apiUrl && config?.apiKey) {
      fetchMissionStatuses(teams, config.apiUrl, config.apiKey);
    }
  }, [teams, config?.apiUrl, config?.apiKey, fetchMissionStatuses]);

  const selectTeam = (team: RescueTeam) => {
    if (!config) return;
    setConfig({
      ...config,
      teamId: team.team_id,
      teamName: team.team_name || team.team_id,
    });
    router.push("/dashboard");
  };

  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const tid = manualTeamId.trim();
    if (!tid) {
      setManualError("กรุณากรอก Team ID");
      return;
    }
    if (!config) return;
    setConfig({ ...config, teamId: tid, teamName: tid });
    router.push("/dashboard");
  };

  const handleLogout = () => {
    logout();
    router.push("/");
  };

  if (!isReady) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-3xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-bold text-gray-900">เลือกทีมกู้ภัย</h1>
            <p className="text-sm text-gray-500 mt-0.5">
              {config?.apiUrl
                ? (() => {
                    try {
                      return new URL(config.apiUrl).hostname;
                    } catch {
                      return config.apiUrl;
                    }
                  })()
                : ""}
            </p>
          </div>
          <button
            onClick={handleLogout}
            className="text-sm text-gray-400 hover:text-red-500 transition px-3 py-1.5 rounded-lg border border-gray-200 hover:border-red-200"
          >
            ออกจากระบบ
          </button>
        </div>

        {/* Team list from RescueTeam service */}
        {config?.rescueTeamUrl ? (
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
              <p className="text-sm font-medium text-gray-700">
                ทีมพร้อมปฏิบัติการ{" "}
                <span className="text-xs font-normal text-gray-400">
                  (AVAILABLE)
                </span>
              </p>
              <div className="flex items-center gap-3">
                {(loading || missionLoading) && (
                  <span className="text-xs text-blue-500 animate-pulse">
                    กำลังโหลด…
                  </span>
                )}
                {!loading && teams.length > 0 && (
                  <span className="text-xs text-gray-400">
                    {teams.length} ทีม
                  </span>
                )}
              </div>
            </div>

            {loading ? (
              <div className="py-12 flex justify-center">
                <div className="animate-spin rounded-full h-7 w-7 border-b-2 border-blue-600" />
              </div>
            ) : error ? (
              <div className="py-8 text-center">
                <p className="text-sm text-red-500 mb-3">{error}</p>
                <button
                  onClick={() => fetchTeams(config.rescueTeamUrl!)}
                  className="text-sm text-blue-600 hover:underline"
                >
                  ลองใหม่
                </button>
              </div>
            ) : teams.length === 0 ? (
              <p className="py-8 text-center text-sm text-gray-400">
                ไม่พบทีมในระบบ
              </p>
            ) : (
              <div className="divide-y divide-gray-50">
                {teams.map((team) => {
                  const ts = getTeamStatusInfo(team.status);
                  const activeMissionStatus =
                    missionStatuses[team.team_id] ?? null;
                  const ms = activeMissionStatus
                    ? getMissionStatusInfo(activeMissionStatus)
                    : null;
                  const hasFetchedMission =
                    Object.prototype.hasOwnProperty.call(
                      missionStatuses,
                      team.team_id,
                    );

                  return (
                    <button
                      key={team.team_id}
                      onClick={() => selectTeam(team)}
                      className="w-full text-left px-5 py-4 hover:bg-blue-50 transition group"
                    >
                      <div className="flex items-start justify-between gap-4">
                        {/* Left: team info */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-semibold text-gray-900 group-hover:text-blue-700 transition">
                              {team.team_name || team.team_id}
                            </span>
                            {team.team_name && (
                              <span className="text-xs text-gray-400 font-mono">
                                {team.team_id}
                              </span>
                            )}
                            {team.team_type && (
                              <span className="text-xs bg-blue-50 text-blue-600 rounded px-1.5 py-0.5">
                                {team.team_type}
                              </span>
                            )}
                          </div>
                          {team.capabilities &&
                            team.capabilities.length > 0 && (
                              <div className="flex flex-wrap gap-1 mt-1.5">
                                {team.capabilities.map((cap) => (
                                  <span
                                    key={cap}
                                    className="text-xs bg-gray-100 text-gray-500 rounded px-1.5 py-0.5"
                                  >
                                    {cap}
                                  </span>
                                ))}
                              </div>
                            )}
                          {team.location && (
                            <p className="text-xs text-gray-400 mt-1">
                              📍{" "}
                              {typeof team.location === "string"
                                ? team.location
                                : `${team.location.lat ?? ""}, ${team.location.lng ?? ""}`}
                            </p>
                          )}
                        </div>

                        {/* Right: status column */}
                        <div className="flex flex-col items-end gap-1.5 shrink-0 pt-0.5 min-w-[140px]">
                          {/* RescueTeam availability status */}
                          <span
                            className={`text-xs font-semibold rounded-full px-2.5 py-1 flex items-center gap-1.5 ${ts.badge}`}
                          >
                            <span
                              className={`w-2 h-2 rounded-full shrink-0 ${ts.dot}`}
                            />
                            {ts.label}
                          </span>
                          <span className="text-xs text-gray-400 text-right leading-tight">
                            {ts.desc}
                          </span>

                          {/* Mission progress status from MissionProgress API */}
                          {ms ? (
                            <span
                              className={`text-xs font-semibold rounded-md px-2 py-0.5 flex items-center gap-1 mt-1 ${ms.badge}`}
                            >
                              <span>{ms.icon}</span>
                              <span>{ms.label}</span>
                            </span>
                          ) : team.status === "BUSY" &&
                            !hasFetchedMission &&
                            missionLoading ? (
                            <span className="text-xs text-gray-300 italic mt-1">
                              โหลดภารกิจ…
                            </span>
                          ) : null}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        ) : (
          /* Fallback: RescueTeam URL not configured */
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
            <p className="text-sm text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 mb-4">
              RescueTeam Service ยังไม่ได้กำหนดค่า — กรอก Team ID ด้วยตนเอง
            </p>
            <form onSubmit={handleManualSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Team ID
                </label>
                <input
                  type="text"
                  value={manualTeamId}
                  onChange={(e) => setManualTeamId(e.target.value)}
                  placeholder="เช่น team-001"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                />
              </div>
              {manualError && (
                <p className="text-sm text-red-500">{manualError}</p>
              )}
              <button
                type="submit"
                className="w-full bg-blue-600 text-white rounded-lg px-4 py-2.5 font-medium hover:bg-blue-700 transition"
              >
                เข้าสู่ Dashboard →
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
