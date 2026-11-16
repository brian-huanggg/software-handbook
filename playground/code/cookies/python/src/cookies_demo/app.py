from fastapi import FastAPI, Response

from .store import SESSION_TTL, create_session

# Entry
app = FastAPI()

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
    return {"message": "Login successful, cookie has been set"}