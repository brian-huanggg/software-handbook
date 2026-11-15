#!/usr/bin/env bash
# 驗證練習結果：在 clicks.link_id 建索引之前與之後，各看一次計畫節點。
set -euo pipefail

q() { docker compose exec -T postgres psql -U app -d shortener -At -c "$1"; }

plan() {
  q "EXPLAIN SELECT count(*) FROM clicks WHERE link_id = 42" | grep -Eo 'Seq Scan|Index Only Scan|Index Scan|Bitmap Heap Scan' | head -1
}

echo "rows: $(q 'SELECT count(*) FROM clicks')"
before=$(plan)
echo "before index: $before"
if [ "$before" != "Seq Scan" ]; then
  echo "FAIL: expected Seq Scan before creating the index" >&2
  exit 1
fi

q "CREATE INDEX IF NOT EXISTS clicks_link_id_idx ON clicks (link_id)" >/dev/null
q "ANALYZE clicks" >/dev/null
after=$(plan)
echo "after index: $after"
if [ "$after" = "Seq Scan" ]; then
  echo "FAIL: planner still uses Seq Scan" >&2
  exit 1
fi
q "DROP INDEX clicks_link_id_idx" >/dev/null
echo "PASS"
