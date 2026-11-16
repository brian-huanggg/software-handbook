# Cookies

## Pseudocode

```
// 模擬伺服器端的 Session 儲存空間（實務上常使用 Redis 或資料庫）
session_store = {}

FUNCTION handle_request(http_request):
    // 1. 解析 HTTP 請求的方法與路徑
    method = http_request.method
    path = http_request.path

    // 2. 路由判斷：處理 GET /login
    IF method == "GET" AND path == "/login":
        
        // 產生一組高強度、不可預測的隨機 Session ID
        session_id = generate_secure_random_token()
        
        // 在伺服器端記錄此 Session 的狀態與過期時間
        session_store[session_id] = {
            "status": "authenticated",
            "created_at": current_timestamp(),
            "expires_in": 3600 // 單位：秒（1 小時）
        }

        // 組合 Set-Cookie 標頭字串，並加上安全屬性
        cookie_header = "session_id=" + session_id + 
                        "; Max-Age=3600" + 
                        "; Path=/" + 
                        "; HttpOnly" + 
                        "; Secure" + 
                        "; SameSite=Lax"

        // 建立並回傳 HTTP 200 OK 回應
        http_response = {
            "status_code": 200,
            "headers": {
                "Content-Type": "application/json",
                "Set-Cookie": cookie_header
            },
            "body": '{"message": "Login successful, cookie has been set."}'
        }
        
        RETURN http_response

    ELSE:
        // 處理其他未定義的路由
        RETURN {
            "status_code": 404,
            "headers": { "Content-Type": "text/plain" },
            "body": "Not Found"
        }
```