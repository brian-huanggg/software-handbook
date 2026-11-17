import secrets
import time

SESSION_TTL = 3600
# Store
_sessions: dict[str, dict] = {}

def create_session() -> str:
    session_id = secrets.token_urlsafe(32) # 產生 32 個密碼學安全的隨機位元組，再編碼成 URL 安全的字串
    _sessions[session_id] = {
        "status": "authenticated",
        "created_at": time.time(), # time.time() 回傳目前時間，格式是 Unix 時間戳記：從 1970-01-01 UTC 起算的秒數，型別是 float
        "expires_in": SESSION_TTL,
    }
    return session_id
