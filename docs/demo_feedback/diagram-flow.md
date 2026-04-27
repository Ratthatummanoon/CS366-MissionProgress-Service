```mermaid
graph LR

    %% ===== Service Nodes =====
    MP["📋 MissionProgress<br/>Service<br/>(รัฐธรรมนูญ)"]
    FE["🖥️ Rescue Team<br/>Dashboard<br/>(Frontend)"]
    RR["📝 RescueRequest<br/>Service<br/>(Phattharaphum)"]
    MD["🚀 ManageDispatch<br/>Service<br/>(Noppakron)"]
    RT["👥 RescueTeam<br/>Service<br/>(กมลพันธ์)"]
    IT["🔍 IncidentTracking<br/>Service<br/>(Krittamet)"]
    PR["⚡ Prioritization<br/>Service<br/>(Nattasak)"]

    %% ===== Event Bus / Queues =====
    EB[["📨 mission-progress-events<br/>(Custom EventBridge Bus)"]]
    DEFAULT_EB[["📨 default event bus<br/>(AWS EventBridge)"]]
    IT_SQS[["📬 IncidentTracking SQS<br/>(owned by Krittamet)"]]
    MD_SQS[["📬 ManageDispatch SQS<br/>(owned by Noppakron)"]]
    PR_SQS[["📬 Prioritization SQS<br/>(owned by Nattasak)"]]

    %% ===== Sync Inbound: Frontend → MP (POST /missions/{request_id}/progress) =====
    FE -->|"🔵 POST /missions/{request_id}/progress<br/>headers: x-api-key, X-Rescue-Team-ID<br/>fields: new_status,<br/>new_impact_level, image_key<br/>⏱ at: ทีมกดปุ่มบน Dashboard"| MP
    MP -->|"🟢 response 200<br/>fields: mission_id, request_id,<br/>old_status, new_status, updated_at"| FE

    %% ===== Sync Outbound: MP → 3 Services =====
    MP -->|"🔵 GET /v1/rescue-requests/{requestId}<br/>auth: Bearer token<br/>⏱ at: CREATE mission"| RR
    RR -->|"🟢 response 200<br/>fields: master: {requestId, incidentId,<br/>requestType, description, peopleCount,<br/>latitude, longitude, locationDetails}"| MP

    MP -->|"🔵 GET /v1/dispatches?teamId={teamId}<br/>auth: Bearer token<br/>⏱ at: GET mission (on-read)"| MD
    MD -->|"🟢 response 200<br/>fields: teamId,<br/>items[]: {dispatchId, requestId,<br/>status, priorityLevel, dispatchedAt}"| MP

    MP -->|"🔵 GET /v1/teams/{teamId}<br/>auth: Bearer token<br/>⏱ at: GET mission (on-read)"| RT
    RT -->|"🟢 response 200<br/>fields: team_id, team_name, team_type,<br/>status, capabilities, equipment,<br/>location: {lat, lng}"| MP

    MP -->|"🔵 PATCH /v1/teams/{teamId}/status<br/>body: {status: AVAILABLE}<br/>fire-and-forget on RESOLVED"| RT

    %% ===== Async Inbound: ManageDispatch → MP =====
    MD -->|"🟣 publish: DispatchOrderCreated<br/>fields: dispatchId, requestId,<br/>teamId, priorityLevel,<br/>status, dispatchedAt"| DEFAULT_EB
    DEFAULT_EB -->|"🔴 consume: DispatchOrderCreated<br/>→ create Mission status: DISPATCHED<br/>fields: dispatchId, requestId,<br/>teamId, priorityLevel,<br/>status, dispatchedAt"| MP

    %% ===== Async Outbound: MP → Custom EventBridge =====
    MP -->|"🟣 publish: MissionStatusChanged<br/>fields: schema_version, mission_id,<br/>requestId, incident_id, rescue_team_id,<br/>old_status, new_status,<br/>changed_at, changed_by<br/>transitions:<br/>DISPATCHED→EN_ROUTE<br/>EN_ROUTE→ON_SITE<br/>ON_SITE→RESOLVED<br/>NEED_BACKUP→ON_SITE<br/>NEED_BACKUP→RESOLVED"| EB
    MP -->|"🟣 publish: MissionBackupRequested<br/>triggered: ON_SITE→NEED_BACKUP<br/>fields: schema_version, mission_id,<br/>incident_id, rescue_team_id,<br/>requested_at, requested_by, location"| EB
    MP -->|"🟣 publish: ImpactLevelUpdated<br/>triggered: new_impact_level changed<br/>fields: schema_version, mission_id,<br/>incident_id, rescue_team_id,<br/>old_level, new_level,<br/>updated_at, updated_by"| EB

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
    style FE fill:#607D8B,stroke:#333,stroke-width:2px,color:#fff
    style RT fill:#00BCD4,stroke:#333,stroke-width:2px,color:#fff
    style IT fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
    style PR fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style EB fill:#FFF9C4,stroke:#333,stroke-width:2px
    style DEFAULT_EB fill:#E1F5FE,stroke:#333,stroke-width:2px
    style IT_SQS fill:#FFF3E0,stroke:#FF9800,stroke-width:2px
    style MD_SQS fill:#F3E5F5,stroke:#9C27B0,stroke-width:2px
    style PR_SQS fill:#FFEBEE,stroke:#F44336,stroke-width:2px
```
