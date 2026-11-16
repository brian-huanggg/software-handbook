import express from "express"
import { create_session, SESSION_TTL } from "./store.js";

const app = express();

app.post("/login", (_, res) => {
    const sessionId = create_session();
    res.cookie("session_id", sessionId, {
        maxAge: SESSION_TTL * 1000,  // Express 的 maxAge 單位是毫秒，
        path: "/",
        httpOnly: true,
        secure: true,
        sameSite: "lax"
    });

    res.json({"message": "Login successful, cookie has been set."});
});

app.listen(3000, () => console.log("App is listening in http://localhost:3000"));