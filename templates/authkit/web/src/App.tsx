import { useCallback, useEffect, useState } from "react"

import { LoginForm, type Kind } from "@/components/login-form"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import * as api from "@/lib/api"

const MODES: { id: api.Mode; label: string; hint: string }[] = [
  {
    id: "session",
    label: "Session",
    hint: "Server-side session. Browser keeps an HttpOnly cookie.",
  },
  {
    id: "token",
    label: "Token",
    hint: "Access JWT lives in memory. Refresh token is an HttpOnly cookie.",
  },
]

export default function App() {
  const [mode, setMode] = useState<api.Mode>("session")
  const [user, setUser] = useState<api.Me | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [log, setLog] = useState<string[]>([])

  const note = useCallback(
    (line: string) =>
      setLog((l) => [`${new Date().toLocaleTimeString()}  ${line}`, ...l].slice(0, 30)),
    [],
  )

  // Wrap an API call: show busy state, record the outcome, surface errors.
  const run = useCallback(
    async (label: string, fn: () => Promise<unknown>): Promise<boolean> => {
      setBusy(true)
      setError(null)
      try {
        await fn()
        note(`${label} -> ok`)
        return true
      } catch (e) {
        const err = e as api.ApiError
        note(`${label} -> ${err.status ?? "network error"} ${err.message}`)
        setError(err.message)
        return false
      } finally {
        setBusy(false)
      }
    },
    [note],
  )

  // On load and when switching mode: restore an existing login, if any.
  // In token mode that means a silent refresh using the refresh cookie.
  useEffect(() => {
    let cancelled = false
    api
      .me(mode)
      .then((u) => {
        if (cancelled) return
        setUser(u)
        note(`restored ${mode} login`)
      })
      .catch(() => {
        /* not logged in: show the form */
      })
    return () => {
      cancelled = true
    }
  }, [mode, note])

  async function handleSubmit(
    kind: Kind,
    email: string,
    password: string,
  ): Promise<boolean> {
    if (kind === "register") {
      return run("POST /register", () => api.register(email, password))
    }
    return run(`POST /${mode}/login`, async () => {
      await api.login(mode, email, password)
      setUser(await api.me(mode))
    })
  }

  return (
    <div className="flex min-h-svh flex-col items-center gap-6 bg-muted p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <div className="grid grid-cols-2 gap-2 rounded-lg bg-background p-1">
          {MODES.map((m) => (
            <Button
              key={m.id}
              variant={mode === m.id ? "default" : "ghost"}
              onClick={() => {
                setMode(m.id)
                setUser(null)
                setError(null)
              }}
            >
              {m.label}
            </Button>
          ))}
        </div>
        <p className="text-center text-sm text-muted-foreground">
          {MODES.find((m) => m.id === mode)?.hint}
        </p>

        {user ? (
          <Card>
            <CardHeader className="text-center">
              <CardTitle className="text-xl">Signed in</CardTitle>
              <CardDescription>{mode} mode</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs">
                {JSON.stringify(user, null, 2)}
              </pre>
              {mode === "token" && (
                <p className="text-xs text-muted-foreground">
                  access token in memory:{" "}
                  <code>{api.getAccessToken()?.slice(0, 16) ?? "none"}…</code>
                </p>
              )}
              <Button
                variant="outline"
                disabled={busy}
                onClick={() =>
                  run(`GET /${mode}/me`, async () => setUser(await api.me(mode)))
                }
              >
                Call /me
              </Button>
              {mode === "token" && (
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() =>
                    run("POST /token/refresh", api.refreshAccessToken)
                  }
                >
                  Refresh access token
                </Button>
              )}
              <Button
                disabled={busy}
                onClick={async () => {
                  await run(`POST /${mode}/logout`, () => api.logout(mode))
                  setUser(null)
                }}
              >
                Logout
              </Button>
              {error && (
                <p role="alert" className="text-sm text-destructive">
                  {error}
                </p>
              )}
            </CardContent>
          </Card>
        ) : (
          <LoginForm onSubmit={handleSubmit} error={error} busy={busy} />
        )}

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Request log</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="max-h-48 overflow-auto text-xs whitespace-pre-wrap">
              {log.length ? log.join("\n") : "nothing yet"}
            </pre>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
