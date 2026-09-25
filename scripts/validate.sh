#!/usr/bin/env sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
set -a
if [ -f .env ]; then . ./.env; else . ./.env.example; fi
set +a

command -v jq >/dev/null 2>&1 || { echo "jq is required for API validation" >&2; exit 1; }
(cd backend && go test ./... && go build ./...)
(cd frontend && npm ci --no-audit --no-fund && npm run typecheck && npm run build)
docker compose config --quiet
docker compose down -v --remove-orphans
docker compose up -d --build

cleanup() { docker compose down -v --remove-orphans; }
if [ "${KEEP_RUNNING:-0}" = "1" ]; then
  trap cleanup INT TERM
else
  trap cleanup EXIT INT TERM
fi

backend_url="http://127.0.0.1:${BACKEND_PORT:-19518}"
frontend_url="http://127.0.0.1:${FRONTEND_PORT:-18518}"
i=0
until curl -fsS "$backend_url/healthz" | jq -e '.data.status == "ok" and .data.database == "ready" and .data.redis == "ready"' >/dev/null; do
  i=$((i+1))
  [ "$i" -lt 60 ] || { docker compose logs; exit 1; }
  sleep 2
done
curl -fsS "$frontend_url/" >/dev/null

login() {
  curl -fsS -X POST "$backend_url/api/auth/login" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"Admin123!\"}" | jq -er '.data.token'
}

admin_token=$(login admin)
viewer_token=$(login viewer)
operator_token=$(login operator)
reviewer_token=$(login reviewer)

curl -fsS "$backend_url/api/session" -H "Authorization: Bearer $admin_token" | jq -e '.data.role == "admin" and (.data.requestId | length > 0)' >/dev/null
curl -fsS "$backend_url/api/runtime" -H "Authorization: Bearer $admin_token" | jq -e '.data.appName and .data.databaseDriver == "postgres" and .data.redisEnabled' >/dev/null

for resource in generators carriers manifests checks; do
  curl -fsS "$backend_url/api/$resource?page=1&pageSize=20" -H "Authorization: Bearer $viewer_token" | jq -e '.data | type == "array"' >/dev/null
done

now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
stamp=$(date '+%s')
manifest_code="TM-VALIDATE-$stamp"
manifest_payload=$(jq -nc --arg code "$manifest_code" --arg now "$now" '{
  code:$code,name:"空卷验收联单",description:"Compose API validation",
  generatorCode:"WG-001",carrierCode:"CP-002",wasteCode:"HW08-900-249-08",quantityKg:680.5,destination:"合规处置中心 A",
  facility:"东区危废暂存区",owner:"operator",category:"危废转运",riskLevel:"medium",metricValue:68,metricUnit:"score",
  effectiveAt:$now,evidence:"minio://evidence/validation/manifest.pdf",relatedCode:"VALIDATION"
}')

viewer_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests" \
  -H "Authorization: Bearer $viewer_token" -H 'Content-Type: application/json' -d "$manifest_payload")
[ "$viewer_status" = "403" ]

created=$(curl -fsS -X POST "$backend_url/api/manifests" -H "Authorization: Bearer $operator_token" \
  -H 'X-Request-ID: validation-manifest-create' -H 'Content-Type: application/json' -d "$manifest_payload")
manifest_id=$(printf '%s' "$created" | jq -er '.data.id')
manifest_version=$(printf '%s' "$created" | jq -er '.data.version')
printf '%s' "$created" | jq -e '.data.status == "draft" and .data.generatorCode == "WG-001" and .data.carrierCode == "CP-002"' >/dev/null

skip_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d "{\"status\":\"in_transit\",\"expectedVersion\":$manifest_version,\"reason\":\"skip must be rejected\"}")
[ "$skip_status" = "422" ]

submitted=$(curl -fsS -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'X-Request-ID: validation-manifest-submit' -H 'Content-Type: application/json' \
  -d "{\"status\":\"submitted\",\"expectedVersion\":$manifest_version,\"reason\":\"generator and carrier evidence verified\"}")
printf '%s' "$submitted" | jq -e '.data.status == "submitted" and .data.version == 2' >/dev/null

stale_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d '{"status":"in_transit","expectedVersion":1,"reason":"stale version must conflict"}')
[ "$stale_status" = "409" ]

blocked_code="TM-BLOCKED-$stamp"
blocked_payload=$(printf '%s' "$manifest_payload" | jq --arg code "$blocked_code" '.code=$code | .carrierCode="CP-001"')
blocked=$(curl -fsS -X POST "$backend_url/api/manifests" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$blocked_payload")
blocked_id=$(printf '%s' "$blocked" | jq -er '.data.id')
blocked_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests/$blocked_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d '{"status":"submitted","expectedVersion":1,"reason":"unverified carrier must block"}')
[ "$blocked_status" = "422" ]

check_code="CC-VALIDATE-$stamp"
check_payload=$(jq -nc --arg code "$check_code" --arg manifest "$manifest_code" --arg now "$now" '{
  code:$code,name:"空卷验收核验",description:"Compose compliance decision validation",manifestCode:$manifest,
  checklist:"产废许可、承运资质、联单数量、处置去向",decisionBasis:"",facility:"复核中心",owner:"reviewer",
  category:"联单复核",riskLevel:"medium",metricValue:92,metricUnit:"score",effectiveAt:$now,
  evidence:"minio://evidence/validation/check.pdf",relatedCode:$manifest
}')
check=$(curl -fsS -X POST "$backend_url/api/checks" -H "Authorization: Bearer $operator_token" \
  -H 'X-Request-ID: validation-check-create' -H 'Content-Type: application/json' -d "$check_payload")
check_id=$(printf '%s' "$check" | jq -er '.data.id')
check_version=$(printf '%s' "$check" | jq -er '.data.version')

operator_decision=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/checks/$check_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d "{\"status\":\"pass\",\"expectedVersion\":$check_version,\"reason\":\"operator cannot decide\"}")
[ "$operator_decision" = "403" ]

decision=$(curl -fsS -X POST "$backend_url/api/checks/$check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'X-Request-ID: validation-reviewer-decision' -H 'Content-Type: application/json' \
  -d "{\"status\":\"pass\",\"expectedVersion\":$check_version,\"reason\":\"all evidence groups verified\"}")
printf '%s' "$decision" | jq -e '.data.status == "pass" and .data.version == 2 and .data.decisionBasis == "all evidence groups verified"' >/dev/null

viewer_audit_status=$(curl -sS -o /dev/null -w '%{http_code}' "$backend_url/api/audits" -H "Authorization: Bearer $viewer_token")
[ "$viewer_audit_status" = "403" ]
audits=$(curl -fsS "$backend_url/api/audits?page=1&pageSize=100" -H "Authorization: Bearer $reviewer_token")
printf '%s' "$audits" | jq -e '([.data[].requestId]) as $ids | ($ids | index("validation-manifest-submit")) != null and ($ids | index("validation-reviewer-decision")) != null' >/dev/null
curl -fsS "$backend_url/api/audit-summary?windowHours=24" -H "Authorization: Bearer $reviewer_token" | jq -e '.data.total >= 5 and .data.transitions >= 2' >/dev/null

docker compose ps
[ "${KEEP_RUNNING:-0}" = "1" ] && echo "KEEP_RUNNING=1: containers left running for built-in Browser validation"
