-- AI 產生：使用者改 email、依 tag 篩選、顯示點擊數。
UPDATE users SET email = $1 WHERE id = $2;

SELECT id, code, url FROM links WHERE tags LIKE '%go%';

UPDATE links SET click_count = click_count + 1 WHERE code = $1;
