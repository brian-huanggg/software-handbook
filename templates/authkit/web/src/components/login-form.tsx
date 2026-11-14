import { useState } from "react"
import { cn } from "cn"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

export type Kind = "login" | "register"

interface LoginFormProps extends Omit<React.ComponentProps<"div">, "onSubmit"> {
  onSubmit: (kind: Kind, email: string, password: string) => Promise<boolean>
  error?: string | null
  busy?: boolean
}

// Based on the shadcn "login-03" block, with the social buttons removed and
// a login / register toggle added.
export function LoginForm({
  className,
  onSubmit,
  error,
  busy,
  ...props
}: LoginFormProps) {
  const [kind, setKind] = useState<Kind>("login")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const isLogin = kind === "login"

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-xl">
            {isLogin ? "Welcome back" : "Create an account"}
          </CardTitle>
          <CardDescription>
            {isLogin
              ? "Login with your email and password"
              : "Use a password of 8 to 128 characters"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={async (e) => {
              e.preventDefault()
              const ok = await onSubmit(kind, email, password)
              if (ok && !isLogin) setKind("login") // registered: now sign in
              if (ok) setPassword("")
            }}
          >
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="email">Email</FieldLabel>
                <Input
                  id="email"
                  type="email"
                  autoComplete="username"
                  placeholder="m@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">Password</FieldLabel>
                <Input
                  id="password"
                  type="password"
                  autoComplete={isLogin ? "current-password" : "new-password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </Field>
              {error && (
                <p role="alert" className="text-sm text-destructive">
                  {error}
                </p>
              )}
              <Field>
                <Button type="submit" disabled={busy}>
                  {isLogin ? "Login" : "Sign up"}
                </Button>
                <FieldDescription className="text-center">
                  {isLogin ? "Don't have an account? " : "Already registered? "}
                  <button
                    type="button"
                    className="underline underline-offset-4"
                    onClick={() => setKind(isLogin ? "register" : "login")}
                  >
                    {isLogin ? "Sign up" : "Login"}
                  </button>
                </FieldDescription>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
