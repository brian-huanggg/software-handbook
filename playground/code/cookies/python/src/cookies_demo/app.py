from fastapi import FastAPI, Request, Response
from fastapi.responses import PlainTextResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

from .store import SESSION_TTL, create_session

# Entry
app = FastAPI()


@app.exception_handler(StarletteHTTPException)
def not_found(_: Request, __: StarletteHTTPException):
    # 405 等未定義的 method/路由一律回 404
    return PlainTextResponse("Not Found", status_code=404)


@app.post("/login")
def login(response: Response):
    session_id = create_session()
    response.set_cookie(
        key="session_id",
        value=session_id,
        max_age=SESSION_TTL,
        path="/",
        httponly=True,
        secure=True,
        samesite="lax",
    )
    return {"message": "Login successful, cookie has been set."}
