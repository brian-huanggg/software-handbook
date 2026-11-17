# 051 Cookie Session

寫一個 HTTP 伺服器，登入成功後用 cookie 保存 session。

## 需求

- `GET /login`：產生不可預測的隨機 session ID，存進伺服器端的 session store，再用 `Set-Cookie` 回傳給瀏覽器。
- Cookie 必須帶 `Max-Age=3600`、`Path=/`、`HttpOnly`、`Secure`、`SameSite=Lax`。
- 回應 `200`，body 為 `{"message": "Login successful, cookie has been set."}`。
- 其他路由回應 `404 Not Found`。

## 練習重點

- 路由與 HTTP 標頭
- 用安全的亂數產生 token
- Cookie 安全屬性各自防範什麼攻擊
