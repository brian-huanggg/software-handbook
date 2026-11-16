#pragma once

#include <chrono>
#include <cstdint>
#include <mutex>
#include <random>
#include <string>
#include <unordered_map>
#include <vector>

constexpr int SESSION_TTL = 3600;

struct Session {
    std::string status;
    std::chrono::system_clock::time_point created_at;
    int expires_in;  // 秒
};

inline std::mutex g_mu;
inline std::unordered_map<std::string, Session> g_sessions;

inline std::string base64url(const std::vector<uint8_t>& in) {
    static const char T[] =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
    std::string out;
    size_t i = 0;
    for (; i + 2 < in.size(); i += 3) {
        uint32_t n = (in[i] << 16) | (in[i + 1] << 8) | in[i + 2];
        out += T[(n >> 18) & 63];
        out += T[(n >> 12) & 63];
        out += T[(n >> 6) & 63];
        out += T[n & 63];
    }
    if (i + 1 == in.size()) {
        uint32_t n = in[i] << 16;
        out += T[(n >> 18) & 63];
        out += T[(n >> 12) & 63];
    } else if (i + 2 == in.size()) {
        uint32_t n = (in[i] << 16) | (in[i + 1] << 8);
        out += T[(n >> 18) & 63];
        out += T[(n >> 12) & 63];
        out += T[(n >> 6) & 63];
    }
    return out;  // 不補 '='
}

inline std::string create_session() {
    thread_local std::random_device rd;
    std::vector<uint8_t> b(32);
    for (auto& byte : b) byte = static_cast<uint8_t>(rd() & 0xFF);
    std::string id = base64url(b);

    std::lock_guard<std::mutex> lock(g_mu);
    g_sessions[id] = Session{"authenticated",
                             std::chrono::system_clock::now(), SESSION_TTL};
    return id;
}
