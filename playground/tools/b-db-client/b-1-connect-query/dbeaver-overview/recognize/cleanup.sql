-- AI 產生：清掉測試短網址的點擊與連結（貼進正式環境的 SQL 編輯器，用執行腳本跑）
DELETE FROM clicks
WHERE link_id IN (SELECT id FROM links WHERE code LIKE '%test%');

DELETE FROM links
WHERE code LIKE '%test%';
