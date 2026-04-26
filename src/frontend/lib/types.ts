export interface MissionAssignment {
  mission_id: string;
  request_id: string;
  incident_id: string;
  rescue_team_id: string;
  current_status: MissionStatus;
  latest_impact_level: number;
  started_at: string;
  last_updated_at: string;
}

export interface TimelineEntry {
  mission_id: string;
  timestamp: string;
  log_id: string;
  action_type: string;
  description: string;
  performed_by: string;
  new_status?: string;
  old_status?: string;
  note?: string;
  location?: string;
  image_key?: string;
}

export interface MissionDetailResponse {
  request_id: string;
  incident_id: string;
  mission_id: string;
  rescue_team_id: string;
  current_status: MissionStatus;
  latest_impact_level: number;
  started_at: string;
  last_updated_at: string;
  description?: string;
  location?: string;
  incident_type?: string;
  timeline: TimelineEntry[];
  data_source: "full" | "partial";
}

export interface ListMissionsResponse {
  team_id: string;
  total_missions: number;
  missions: MissionAssignment[];
}

export interface ProgressResponse {
  message: string;
  mission_id: string;
  request_id: string;
  incident_id: string;
  old_status: string;
  new_status: string;
  updated_at: string;
}

export interface PresignedURLResponse {
  upload_url: string;
  image_key: string;
  expires_in: number;
  message: string;
}

export interface APIError {
  error: string;
  code: string;
  message: string;
}

export type MissionStatus =
  | "DISPATCHED"
  | "EN_ROUTE"
  | "ON_SITE"
  | "NEED_BACKUP"
  | "RESOLVED";

export const STATUS_LABELS: Record<MissionStatus, string> = {
  DISPATCHED: "รอเดินทาง",
  EN_ROUTE: "กำลังเดินทาง",
  ON_SITE: "ถึงจุดเกิดเหตุ",
  NEED_BACKUP: "ต้องการกำลังเสริม",
  RESOLVED: "เสร็จสิ้น",
};

export const STATUS_COLORS: Record<MissionStatus, string> = {
  DISPATCHED: "bg-gray-100 text-gray-800 border-gray-300",
  EN_ROUTE: "bg-blue-100 text-blue-800 border-blue-300",
  ON_SITE: "bg-yellow-100 text-yellow-800 border-yellow-300",
  NEED_BACKUP: "bg-red-100 text-red-800 border-red-300",
  RESOLVED: "bg-green-100 text-green-800 border-green-300",
};

export const VALID_TRANSITIONS: Record<MissionStatus, MissionStatus[]> = {
  DISPATCHED: ["EN_ROUTE"],
  EN_ROUTE: ["ON_SITE"],
  ON_SITE: ["NEED_BACKUP", "RESOLVED"],
  NEED_BACKUP: ["ON_SITE", "RESOLVED"],
  RESOLVED: [],
};

export const IMPACT_LABELS: Record<number, string> = {
  1: "ต่ำ",
  2: "ปานกลาง",
  3: "สูง",
  4: "วิกฤต",
};

export const IMPACT_COLORS: Record<number, string> = {
  1: "bg-green-100 text-green-800",
  2: "bg-yellow-100 text-yellow-800",
  3: "bg-orange-100 text-orange-800",
  4: "bg-red-100 text-red-800",
};
