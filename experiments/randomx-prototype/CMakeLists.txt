cmake_minimum_required(VERSION 3.16)
project(valdr_randomx_prototype LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(NOT DEFINED RANDOMX_DIR)
  set(RANDOMX_DIR "${CMAKE_CURRENT_SOURCE_DIR}/.deps/RandomX")
endif()

if(NOT EXISTS "${RANDOMX_DIR}/CMakeLists.txt")
  message(FATAL_ERROR "RandomX source not found at ${RANDOMX_DIR}. Run build.sh/build.ps1 first.")
endif()

set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
add_subdirectory("${RANDOMX_DIR}" "${CMAKE_BINARY_DIR}/randomx")

add_executable(valdr-randomx-vector main.cpp)
target_include_directories(valdr-randomx-vector PRIVATE "${RANDOMX_DIR}/src")
target_link_libraries(valdr-randomx-vector PRIVATE randomx)
