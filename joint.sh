#!/bin/sh
set -e
ARCIS=${ARCIS:-http://127.0.0.1:9090}
DESK=${DESK:-http://127.0.0.1:8790}
curl -fsS "$ARCIS/health" >/dev/null
curl -fsS "$DESK/home" | grep -q student_arcis
curl -fsS -X POST "$DESK/api/agent/enroll" -H 'Content-Type: application/json' -d '{"name":"study-pc","platform":"windows","token":"desk"}' >/dev/null
curl -fsS -X POST "$DESK/interop/hours" -H 'Content-Type: application/json' -d '{"text":"Algebra 2 hours this week: 4. Students do not get Arcis."}' | grep -q '"ok":true'
echo JOINT-TEST PASSED
