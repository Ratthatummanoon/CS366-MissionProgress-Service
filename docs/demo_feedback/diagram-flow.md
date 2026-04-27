```mermaid
graph LR

    %% ===== Service Nodes =====
    MP["📋 MissionProgress<br/>Service<br/>(รัฐธรรมนูญ)"]
    RR["📝 RescueRequest<br/>Service<br/>(Phattharaphum)"]
    MD["🚀 ManageDispatch<br/>Service<br/>(Noppakron)"]
    RT["👥 RescueTeam<br/>Service<br/>(กมลพันธ์)"]
    IT["🔍 IncidentTracking<br/>Service<br/>(Krittamet)"]
    PR["⚡ Prioritization<br/>Service<br/>(Nattasak)"]

    %% ===== Event Bus / Queues =====
    EB[["📨 mission-progress-events<br/>(Custom EventBridge Bus)"]]
    DEFAULT_EB[["📨 default event bus<br/>(AWS EventBridge)"]]
    IT_SQS[["📬 IncidentTracking SQS<br/>(owned by Krittamet)"]]
    MD_SQS[["📬 Dispatch SQS<br/>(owned by Noppakron)"]]
    PR_SQS[["📬 Prioritization SQS<br/>(owned by Nattasak)"]]

    %% ===== Sync Inbound: RescueTeam → MP (POST /missions/{request_id}/progress) =====
    RT -->|"🔵 POST /missions/{request_id}/progress<br/>fields: new_status,<br/>new_impact_level, image_key<br/>⏱ at: ทีมรายงานจากหน้างาน"| MP
    MP -->|"🟢 response 200<br/>fields: updated mission status"| RT

    %% ===== Sync Outbound: MP → 3 Services =====
    MP -->|"🔵 GET /v1/rescue-requests/{requestId}<br/>auth: Bearer token<br/>⏱ at: CREATE mission"| RR
    RR -->|"🟢 response 200<br/>fields: description, location,<br/>requestType, peopleCount, incidentId"| MP

    MP -->|"🔵 GET /v1/dispatches?teamId={teamId}<br/>auth: Bearer token<br/>⏱ at: GET mission (on-read)"| MD
    MD -->|"🟢 response 200<br/>fields: dispatch details, status, priorityLevel"| MP

    MP -->|"🔵 GET /v1/teams/{teamId}<br/>auth: Bearer token<br/>⏱ at: GET mission (on-read)"| RT
    RT -->|"🟢 response 200<br/>fields: team name, capabilities, location"| MP

    MP -->|"🔵 PATCH /v1/teams/{teamId}/status<br/>body: {status: AVAILABLE}<br/>fire-and-forget on RESOLVED"| RT

    %% ===== Async Inbound: ManageDispatch → MP =====
    MD -->|"🟣 publish: DispatchOrderCreated<br/>fields: dispatchId, requestId,<br/>teamId, priorityLevel"| DEFAULT_EB
    DEFAULT_EB -->|"🔴 consume: DispatchOrderCreated<br/>→ create Mission status: DISPATCHED<br/>fields: dispatchId, requestId,<br/>teamId, priorityLevel"| MP

    %% ===== Async Outbound: MP → Custom EventBridge =====
    MP -->|"🟣 publish: MissionStatusChanged<br/>fields: mission_id, incident_id,<br/>rescue_team_id, old_status,<br/>new_status, changed_at, changed_by<br/>transitions:<br/>DISPATCHED→EN_ROUTE<br/>EN_ROUTE→ON_SITE<br/>ON_SITE→RESOLVED<br/>NEED_BACKUP→ON_SITE<br/>NEED_BACKUP→RESOLVED"| EB
    MP -->|"🟣 publish: MissionBackupRequested<br/>triggered: ON_SITE→NEED_BACKUP<br/>fields: mission_id, incident_id,<br/>rescue_team_id, requested_at,<br/>location"| EB
    MP -->|"🟣 publish: ImpactLevelUpdated<br/>triggered: new_impact_level changed<br/>fields: mission_id, incident_id,<br/>rescue_team_id, old_level,<br/>new_level, updated_at"| EB

    %% ===== EventBridge → SQS =====
    EB -->|"🟣 route: MissionStatusChanged<br/>+ ImpactLevelUpdated"| IT_SQS
    EB -->|"🟣 route: MissionStatusChanged<br/>new_status: RESOLVED only"| MD_SQS
    EB -->|"🟣 route: MissionBackupRequested<br/>+ ImpactLevelUpdated"| PR_SQS

    %% ===== SQS → Consumer Services =====
    IT_SQS -->|"🔴 consume<br/>fields: mission_id, incident_id,<br/>new_status, old/new_level<br/>เช่น EN_ROUTE=ลงพื้นที่แล้ว<br/>RESOLVED=จบภารกิจ"| IT
    MD_SQS -->|"🔴 consume<br/>fields: mission_id, incident_id,<br/>new_status: RESOLVED<br/>→ ปิด Dispatch Order"| MD
    PR_SQS -->|"🔴 consume<br/>fields: mission_id, incident_id,<br/>rescue_team_id, location,<br/>old/new_level<br/>→ จัดลำดับใหม่<br/>→ Dispatch ทีมเสริม<br/>→ วนกลับมา MP อีกรอบ 🔄"| PR

    %% ===== Styling =====
    style MP fill:#4CAF50,stroke:#333,stroke-width:3px,color:#fff
    style RR fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style MD fill:#9C27B0,stroke:#333,stroke-width:2px,color:#fff
    style RT fill:#00BCD4,stroke:#333,stroke-width:2px,color:#fff
    style IT fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
    style PR fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style EB fill:#FFF9C4,stroke:#333,stroke-width:2px
    style DEFAULT_EB fill:#E1F5FE,stroke:#333,stroke-width:2px
    style IT_SQS fill:#FFF3E0,stroke:#FF9800,stroke-width:2px
    style MD_SQS fill:#F3E5F5,stroke:#9C27B0,stroke-width:2px
    style PR_SQS fill:#FFEBEE,stroke:#F44336,stroke-width:2px
```
