#include <httplib.h>

#include <iostream>
#include <string>

#include "store.hpp"

int main() {
    httplib::Server svr;

    svr.Post("/login", [](const httplib::Request&, httplib::Response& res) {
        std::string id = create_session();

        std::string cookie = "session_id=" + id +
                             "; Max-Age=" + std::to_string(SESSION_TTL) +
                             "; Path=/; HttpOnly; Secure; SameSite=Lax";

        res.set_header("Set-Cookie", cookie);
        res.set_content(
            R"({"message":"Login successful, cookie has been set."})",
            "application/json");
    });

    std::cout << "App is listening on http://localhost:3000\n";
    svr.listen("0.0.0.0", 3000);
}
