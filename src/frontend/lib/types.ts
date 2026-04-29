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

export interface TeamLocation {
  lat: number;
  lng: number;
}

export interface MissionDetailResponse {
  request_id: string;
  incident_id: string;
  mission_id: string;
  dispatch_id?: string;
  rescue_team_id: string;
  current_status: MissionStatus;
  latest_impact_level: number;
  started_at: string;
  last_updated_at: string;
  description?: string;
  location?: string;
  incident_type?: string;
  // RescueTeam Service enrichment
  team_name?: string;
  team_type?: string;
  capabilities?: string[];
  equipment?: string[];
  team_location?: TeamLocation;
  // ManageDispatch Service enrichment
  dispatch_status?: string;
  priority_level?: number;
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

export interface ViewURLResponse {
  view_url: string;
  image_key: string;
  expires_in: number;
  message: string;
}

export interface APIError {
  error: string;
  code: string;
  message: string;
}

// RescueTeam Service types
export interface RescueTeamLocation {
  lat: number;
  lng: number;
  source?: string;
  updated_at?: string;
}

export interface RescueTeam {
  team_id: string;
  team_name?: string;
  team_type?: string;
  status: "AVAILABLE" | "BUSY" | "OFFLINE";
  location?: RescueTeamLocation;
  capabilities?: string[];
  equipment?: string[];
  updated_at?: string;
}

export interface RescueTeamListResponse {
  teams: RescueTeam[];
  trace_id?: string;
}

// ManageDispatch Service types
export interface DispatchItem {
  dispatchId: string;
  requestId: string;
  status: "PENDING" | "ACCEPT" | "DECLINE";
  priorityLevel?: number;
  dispatchedAt: string;
  note?: string;
}

export interface DispatchListResponse {
  teamId: string;
  items: DispatchItem[];
}

// RescueRequest Service types (citizen status endpoint)
export interface RescueRequestLocation {
  latitude?: number;
  longitude?: number;
  locationDetails?: string;
  addressLine?: string;
  province?: string;
  district?: string;
  subdistrict?: string;
}

export interface RescueRequestCitizenStatus {
  requestId: string;
  incidentId?: string;
  requestType?: string;
  status?: string;
  statusMessage?: string;
  description?: string;
  peopleCount?: number;
  specialNeeds?: string | string[] | null;
  contactName?: string;
  contactPhoneMasked?: string;
  location?: RescueRequestLocation;
  priorityLevel?: string;
  submittedAt?: string;
  lastUpdatedAt?: string;
  stateVersion?: number;
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
