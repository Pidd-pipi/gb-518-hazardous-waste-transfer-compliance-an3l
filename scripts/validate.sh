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

intransit=$(curl -fsS -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d '{"status":"in_transit","expectedVersion":2,"reason":"carrier departed with sealed load"}')
printf '%s' "$intransit" | jq -e '.data.status == "in_transit" and .data.version == 3' >/dev/null

# 核验在联单签收前建立，此时通过必须被闸门拒绝。
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

pass_before_receipt=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/checks/$check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'Content-Type: application/json' \
  -d "{\"status\":\"pass\",\"expectedVersion\":$check_version,\"reason\":\"must not pass before receipt\"}")
[ "$pass_before_receipt" = "422" ]

# 签收必须带实收重量；计划 680.5 kg，5% 上限为 714.525 kg。
receive_no_weight=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d '{"status":"received","expectedVersion":3,"reason":"actual weight is mandatory"}')
[ "$receive_no_weight" = "422" ]

receive_overweight_no_reason=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d '{"status":"received","expectedVersion":3,"reason":"overweight requires reason","receivedWeightKg":720}')
[ "$receive_overweight_no_reason" = "422" ]

received=$(curl -fsS -X POST "$backend_url/api/manifests/$manifest_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'X-Request-ID: validation-manifest-receive' -H 'Content-Type: application/json' \
  -d '{"status":"received","expectedVersion":3,"reason":"site weighing within tolerance","receivedWeightKg":685}')
printf '%s' "$received" | jq -e '.data.status == "received" and .data.version == 4 and .data.receivedWeightKg == 685 and .data.weightDiffKg == 4.5 and (.data.receivedAt | type == "string")' >/dev/null

decision=$(curl -fsS -X POST "$backend_url/api/checks/$check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'X-Request-ID: validation-reviewer-decision' -H 'Content-Type: application/json' \
  -d "{\"status\":\"pass\",\"expectedVersion\":$check_version,\"reason\":\"all evidence groups verified\"}")
printf '%s' "$decision" | jq -e '.data.status == "pass" and .data.version == 2 and .data.decisionBasis == "all evidence groups verified"' >/dev/null

# 超重签收：自动转驳回，保存实收重量、差值与差异原因；核验只能不通过并升级复核。
overweight_code="TM-OVERWEIGHT-$stamp"
overweight_payload=$(printf '%s' "$manifest_payload" | jq --arg code "$overweight_code" '.code=$code | .quantityKg=680.5')
overweight=$(curl -fsS -X POST "$backend_url/api/manifests" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$overweight_payload")
overweight_id=$(printf '%s' "$overweight" | jq -er '.data.id')
for step_status in submitted in_transit; do
  overweight=$(curl -fsS -X POST "$backend_url/api/manifests/$overweight_id/transition" \
    -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
    -d "$(jq -nc --arg status "$step_status" --argjson version "$(printf '%s' "$overweight" | jq -er '.data.version')" '{status:$status,expectedVersion:$version,reason:"move overweight manifest forward"}')")
done
overweight_version=$(printf '%s' "$overweight" | jq -er '.data.version')
auto_rejected=$(curl -fsS -X POST "$backend_url/api/manifests/$overweight_id/transition" \
  -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' \
  -d "{\"status\":\"received\",\"expectedVersion\":$overweight_version,\"reason\":\"site weighing exceeds plan\",\"receivedWeightKg\":720,\"weightDiffReason\":\"桶底积液未沥净，复磅超出计划 5%，车辆暂扣待复核\"}")
printf '%s' "$auto_rejected" | jq -e '.data.status == "rejected" and .data.receivedWeightKg == 720 and .data.weightDiffKg == 39.5 and (.data.weightDiffReason | length > 0) and (.data.receivedAt | type == "string")' >/dev/null

rejected_check_payload=$(printf '%s' "$check_payload" | jq --arg code "CC-REJECTED-$stamp" --arg manifest "$overweight_code" '.code=$code | .manifestCode=$manifest')
rejected_check=$(curl -fsS -X POST "$backend_url/api/checks" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$rejected_check_payload")
rejected_check_id=$(printf '%s' "$rejected_check" | jq -er '.data.id')
rejected_pass=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$backend_url/api/checks/$rejected_check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'Content-Type: application/json' \
  -d '{"status":"pass","expectedVersion":1,"reason":"rejected manifest cannot pass"}')
[ "$rejected_pass" = "422" ]
rejected_fail=$(curl -fsS -X POST "$backend_url/api/checks/$rejected_check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'Content-Type: application/json' \
  -d '{"status":"fail","expectedVersion":1,"reason":"manifest rejected for overweight receipt"}')
printf '%s' "$rejected_fail" | jq -e '.data.status == "fail" and .data.version == 2' >/dev/null
escalated=$(curl -fsS -X POST "$backend_url/api/checks/$rejected_check_id/transition" \
  -H "Authorization: Bearer $reviewer_token" -H 'Content-Type: application/json' \
  -d '{"status":"escalated","expectedVersion":2,"reason":"escalate rejected manifest for senior review"}')
printf '%s' "$escalated" | jq -e '.data.status == "escalated"' >/dev/null

viewer_audit_status=$(curl -sS -o /dev/null -w '%{http_code}' "$backend_url/api/audits" -H "Authorization: Bearer $viewer_token")
[ "$viewer_audit_status" = "403" ]
audits=$(curl -fsS "$backend_url/api/audits?page=1&pageSize=100" -H "Authorization: Bearer $reviewer_token")
printf '%s' "$audits" | jq -e '([.data[].requestId]) as $ids | ($ids | index("validation-manifest-submit")) != null and ($ids | index("validation-reviewer-decision")) != null' >/dev/null
curl -fsS "$backend_url/api/audit-summary?windowHours=24" -H "Authorization: Bearer $reviewer_token" | jq -e '.data.total >= 5 and .data.transitions >= 2' >/dev/null

docker compose ps
[ "${KEEP_RUNNING:-0}" = "1" ] && echo "KEEP_RUNNING=1: containers left running for built-in Browser validation"
