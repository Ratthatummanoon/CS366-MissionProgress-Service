"use client";

import { Suspense, useEffect, useState, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useConfig, getClient } from "@/lib/config-context";
import {
  MissionDetailResponse,
  MissionStatus,
  VALID_TRANSITIONS,
  STATUS_LABELS,
} from "@/lib/types";
import Navbar from "@/components/Navbar";
import StatusBadge from "@/components/StatusBadge";
import ImpactBadge from "@/components/ImpactBadge";
import StateMachineDiagram from "@/components/StateMachineDiagram";

function MissionDetailContent() {
  const searchParams = useSearchParams();
  const incidentId = searchParams.get("id") || "";
  const router = useRouter();
  const { config, isReady } = useConfig();
  const [mission, setMission] = useState<MissionDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Update form
  const [newStatus, setNewStatus] = useState<MissionStatus | "">("");
  const [note, setNote] = useState("");
  const [location, setLocation] = useState("");
  const [impactLevel, setImpactLevel] = useState<number | undefined>();
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitMsg, setSubmitMsg] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isReady && !config) router.push("/");
  }, [isReady, config, router]);

  useEffect(() => {
    if (config && incidentId) loadMission();
  }, [config, incidentId]);

  async function loadMission() {
    const client = getClient();
    if (!client) return;
    setLoading(true);
    setError("");
    try {
      const data = await client.getMission(incidentId);
      setMission(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "โหลดข้อมูลไม่สำเร็จ");
    } finally {
      setLoading(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!newStatus || !mission) return;
    const client = getClient();
    if (!client) return;
    setSubmitting(true);
    setSubmitMsg("");

    try {
      let imageKey: string | undefined;

      // Upload image first if present
      if (imageFile) {
        const presigned = await client.getPresignedUrl(
          incidentId,
          imageFile.name,
          imageFile.type,
        );
        await client.uploadFile(presigned.upload_url, imageFile);
        imageKey = presigned.image_key;
      }

      const body: {
        new_status: string;
        note?: string;
        current_location?: string;
        new_impact_level?: number;
        image_key?: string;
      } = { new_status: newStatus };
      if (note.trim()) body.note = note.trim();
      if (location.trim()) body.current_location = location.trim();
      if (impactLevel) body.new_impact_level = impactLevel;
      if (imageKey) body.image_key = imageKey;

      const res = await client.reportProgress(incidentId, body);
      setSubmitMsg(`อัปเดตสถานะเป็น ${res.new_status} สำเร็จ`);

      // Reset form
      setNewStatus("");
      setNote("");
      setLocation("");
      setImpactLevel(undefined);
      setImageFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";

      // Reload mission
      await loadMission();
    } catch (err) {
      setSubmitMsg(
        `❌ ${err instanceof Error ? err.message : "อัปเดตไม่สำเร็จ"}`,
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (!isReady || !config) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (!incidentId) {
    return (
      <div className="min-h-screen flex flex-col">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center text-gray-500">
            <span className="text-4xl block mb-3">🔍</span>
            <p className="text-lg font-medium">ไม่ได้ระบุ Incident ID</p>
            <button
              onClick={() => router.push("/dashboard")}
              className="mt-4 text-blue-600 hover:underline text-sm"
            >
              กลับหน้ารวม
            </button>
          </div>
        </div>
      </div>
    );
  }

  const validTransitions = mission
    ? VALID_TRANSITIONS[mission.current_status] || []
    : [];

  return (
    <div className="min-h-screen flex flex-col">
      <Navbar />

      <main className="flex-1 max-w-5xl mx-auto w-full px-4 py-6">
        {/* Back button */}
        <button
          onClick={() => router.push("/dashboard")}
          className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4"
        >
          ← กลับหน้ารวม
        </button>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-3 mb-4">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : mission ? (
          <div className="space-y-6">
            {/* Header */}
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <div className="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">
                    {mission.incident_id}
                  </h1>
                  <p className="text-sm text-gray-500 mt-1">
                    Mission ID: {mission.mission_id}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <ImpactBadge level={mission.latest_impact_level} />
                  <StatusBadge status={mission.current_status} size="md" />
                </div>
              </div>

              {/* Incident details if full data */}
              {mission.data_source === "full" && (
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4 pt-4 border-t border-gray-100">
                  {mission.incident_type && (
                    <div>
                      <span className="text-xs text-gray-500">ประเภท</span>
                      <p className="font-medium">{mission.incident_type}</p>
                    </div>
                  )}
                  {mission.location && (
                    <div>
                      <span className="text-xs text-gray-500">สถานที่</span>
                      <p className="font-medium">{mission.location}</p>
                    </div>
                  )}
                  {mission.description && (
                    <div className="sm:col-span-3">
                      <span className="text-xs text-gray-500">รายละเอียด</span>
                      <p className="font-medium">{mission.description}</p>
                    </div>
                  )}
                </div>
              )}

              {mission.data_source === "partial" && (
                <div className="mt-4 pt-4 border-t border-gray-100">
                  <p className="text-xs text-amber-600 bg-amber-50 rounded px-2 py-1 inline-block">
                    ⚠️ แสดงข้อมูลบางส่วน — IncidentTracking Service
                    ไม่พร้อมใช้งาน
                  </p>
                </div>
              )}

              <div className="grid grid-cols-2 gap-4 mt-4 pt-4 border-t border-gray-100 text-sm text-gray-500">
                <div>
                  เริ่มภารกิจ:{" "}
                  {new Date(mission.started_at).toLocaleString("th-TH")}
                </div>
                <div>
                  อัปเดตล่าสุด:{" "}
                  {new Date(mission.last_updated_at).toLocaleString("th-TH")}
                </div>
              </div>
            </div>

            {/* State Machine Diagram */}
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h2 className="text-lg font-semibold mb-4">State Machine</h2>
              <StateMachineDiagram currentStatus={mission.current_status} />
            </div>

            {/* Update Status Form */}
            {validTransitions.length > 0 && (
              <div className="bg-white rounded-xl border border-gray-200 p-6">
                <h2 className="text-lg font-semibold mb-4">
                  อัปเดตสถานะภารกิจ
                </h2>
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      สถานะใหม่
                    </label>
                    <div className="flex gap-2 flex-wrap">
                      {validTransitions.map((s) => (
                        <button
                          type="button"
                          key={s}
                          onClick={() => setNewStatus(s)}
                          className={`px-4 py-2 rounded-lg border text-sm font-medium transition ${
                            newStatus === s
                              ? "border-blue-500 bg-blue-50 text-blue-700"
                              : "border-gray-200 bg-white text-gray-700 hover:bg-gray-50"
                          }`}
                        >
                          {STATUS_LABELS[s]} ({s})
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        หมายเหตุ
                      </label>
                      <textarea
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        rows={2}
                        placeholder="บันทึกเพิ่มเติม..."
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none resize-none"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        ตำแหน่งปัจจุบัน
                      </label>
                      <input
                        type="text"
                        value={location}
                        onChange={(e) => setLocation(e.target.value)}
                        placeholder="เช่น 13.7563,100.5018"
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        ระดับความรุนแรง (1-4)
                      </label>
                      <select
                        value={impactLevel || ""}
                        onChange={(e) =>
                          setImpactLevel(
                            e.target.value ? Number(e.target.value) : undefined,
                          )
                        }
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none bg-white"
                      >
                        <option value="">ไม่เปลี่ยน</option>
                        <option value="1">1 — ต่ำ</option>
                        <option value="2">2 — ปานกลาง</option>
                        <option value="3">3 — สูง</option>
                        <option value="4">4 — วิกฤต</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        แนบรูปภาพ
                      </label>
                      <input
                        ref={fileInputRef}
                        type="file"
                        accept="image/*"
                        onChange={(e) =>
                          setImageFile(e.target.files?.[0] || null)
                        }
                        className="w-full text-sm text-gray-500 file:mr-3 file:rounded-lg file:border-0 file:bg-blue-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
                      />
                    </div>
                  </div>

                  {submitMsg && (
                    <div
                      className={`rounded-lg px-3 py-2 text-sm ${
                        submitMsg.startsWith("❌")
                          ? "bg-red-50 text-red-700 border border-red-200"
                          : "bg-green-50 text-green-700 border border-green-200"
                      }`}
                    >
                      {submitMsg}
                    </div>
                  )}

                  <button
                    type="submit"
                    disabled={!newStatus || submitting}
                    className="bg-blue-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-blue-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {submitting ? "กำลังส่ง..." : "อัปเดตสถานะ"}
                  </button>
                </form>
              </div>
            )}

            {/* Timeline */}
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h2 className="text-lg font-semibold mb-4">
                ไทม์ไลน์ ({mission.timeline?.length || 0} รายการ)
              </h2>

              {!mission.timeline || mission.timeline.length === 0 ? (
                <p className="text-gray-500 text-sm">ยังไม่มีรายการ</p>
              ) : (
                <div className="relative">
                  <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200" />
                  <div className="space-y-6">
                    {mission.timeline.map((entry, i) => (
                      <div key={entry.log_id || i} className="relative pl-10">
                        <div className="absolute left-2.5 top-1.5 w-3 h-3 rounded-full border-2 border-blue-400 bg-white" />
                        <div className="bg-gray-50 rounded-lg p-4">
                          <div className="flex items-center justify-between flex-wrap gap-2">
                            <span className="text-sm font-semibold text-gray-900">
                              {entry.action_type}
                            </span>
                            <span className="text-xs text-gray-500">
                              {new Date(entry.timestamp).toLocaleString(
                                "th-TH",
                              )}
                            </span>
                          </div>
                          <p className="text-sm text-gray-700 mt-1">
                            {entry.description}
                          </p>

                          {(entry.old_status || entry.new_status) && (
                            <div className="flex items-center gap-2 mt-2 text-xs">
                              {entry.old_status && (
                                <StatusBadge
                                  status={entry.old_status as MissionStatus}
                                  size="sm"
                                />
                              )}
                              {entry.old_status && entry.new_status && (
                                <span className="text-gray-400">→</span>
                              )}
                              {entry.new_status && (
                                <StatusBadge
                                  status={entry.new_status as MissionStatus}
                                  size="sm"
                                />
                              )}
                            </div>
                          )}

                          <div className="flex flex-wrap gap-3 mt-2 text-xs text-gray-500">
                            {entry.performed_by && (
                              <span>โดย: {entry.performed_by}</span>
                            )}
                            {entry.location && <span>📍 {entry.location}</span>}
                            {entry.note && (
                              <span className="text-gray-600">
                                💬 {entry.note}
                              </span>
                            )}
                            {entry.image_key && (
                              <span className="text-blue-600">
                                📷 มีรูปภาพแนบ
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}

export default function MissionDetailPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      }
    >
      <MissionDetailContent />
    </Suspense>
  );
}
