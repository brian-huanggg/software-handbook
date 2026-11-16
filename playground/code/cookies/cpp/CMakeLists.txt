cmake_minimum_required(VERSION 3.20)
project(cookies_demo CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(FetchContent)
FetchContent_Declare(
  httplib
  GIT_REPOSITORY https://github.com/yhirose/cpp-httplib.git
  GIT_TAG v0.18.3
)
FetchContent_MakeAvailable(httplib)

add_executable(cookies_demo src/main.cpp)
target_link_libraries(cookies_demo PRIVATE httplib::httplib)
