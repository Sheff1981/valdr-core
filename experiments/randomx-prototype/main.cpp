#include <array>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

#include "randomx.h"

namespace {

std::string hex(const unsigned char* data, std::size_t size) {
    std::ostringstream out;
    out << std::hex << std::setfill('0');
    for (std::size_t i = 0; i < size; ++i) {
        out << std::setw(2) << static_cast<unsigned int>(data[i]);
    }
    return out.str();
}

void appendLE32(std::vector<unsigned char>& out, std::uint32_t value) {
    for (int i = 0; i < 4; ++i) out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
}

void appendLE64(std::vector<unsigned char>& out, std::uint64_t value) {
    for (int i = 0; i < 8; ++i) out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
}

void appendText(std::vector<unsigned char>& out, const std::string& text) {
    appendLE32(out, static_cast<std::uint32_t>(text.size()));
    out.insert(out.end(), text.begin(), text.end());
}

void appendPattern(std::vector<unsigned char>& out, unsigned char start, std::size_t count) {
    for (std::size_t i = 0; i < count; ++i) out.push_back(static_cast<unsigned char>(start + i));
}

std::vector<unsigned char> sampleMiningBlob() {
    std::vector<unsigned char> out;
    appendText(out, "VALDR/RandomX/Mainnet/v1");
    appendText(out, "valdr-mainnet-1");
    appendLE32(out, 3);                    // prototype block version
    appendLE64(out, 262800);               // sample height
    appendPattern(out, 0x00, 32);          // previous block hash sample
    appendPattern(out, 0x20, 32);          // merkle root sample
    appendLE64(out, 1800000000ULL);        // sample timestamp
    appendPattern(out, 0x40, 32);          // sample exact target bytes
    appendLE64(out, 0x0102030405060708ULL); // sample nonce
    return out;
}

bool hashWithKey(const std::string& key,
                 const std::vector<unsigned char>& input,
                 std::array<unsigned char, RANDOMX_HASH_SIZE>& output) {
    randomx_flags flags = randomx_get_flags();
    flags = static_cast<randomx_flags>(flags | RANDOMX_FLAG_SECURE);

    randomx_cache* cache = randomx_alloc_cache(flags);
    if (cache == nullptr) return false;

    randomx_init_cache(cache, key.data(), key.size());

    randomx_vm* vm = randomx_create_vm(flags, cache, nullptr);
    if (vm == nullptr) {
        // JIT/secure can be unavailable on restricted runners. Retry portable mode.
        randomx_release_cache(cache);
        flags = RANDOMX_FLAG_DEFAULT;
        cache = randomx_alloc_cache(flags);
        if (cache == nullptr) return false;
        randomx_init_cache(cache, key.data(), key.size());
        vm = randomx_create_vm(flags, cache, nullptr);
        if (vm == nullptr) {
            randomx_release_cache(cache);
            return false;
        }
    }

    randomx_calculate_hash(vm, input.data(), input.size(), output.data());
    randomx_destroy_vm(vm);
    randomx_release_cache(cache);
    return true;
}

} // namespace

int main() {
    // Official RandomX v1 compatibility vector.
    const std::string officialKey = "test key 000";
    const std::string officialInputText = "This is a test";
    std::vector<unsigned char> officialInput(officialInputText.begin(), officialInputText.end());

    std::array<unsigned char, RANDOMX_HASH_SIZE> officialHash{};
    if (!hashWithKey(officialKey, officialInput, officialHash)) {
        std::cerr << "RandomX initialization failed\n";
        return 2;
    }

    const std::string expectedOfficial =
        "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f";
    const std::string actualOfficial = hex(officialHash.data(), officialHash.size());

    std::cout << "randomx_vector=" << actualOfficial << "\n";
    if (actualOfficial != expectedOfficial) {
        std::cerr << "official RandomX v1 vector mismatch\n";
        return 3;
    }

    const std::string valdrSeed = "VALDR/RandomX/Mainnet/v1/epoch-0";
    const auto blob = sampleMiningBlob();
    std::array<unsigned char, RANDOMX_HASH_SIZE> valdrHash{};

    if (!hashWithKey(valdrSeed, blob, valdrHash)) {
        std::cerr << "VALDR vector initialization failed\n";
        return 4;
    }

    std::cout << "valdr_blob_bytes=" << blob.size() << "\n";
    std::cout << "valdr_blob_hex=" << hex(blob.data(), blob.size()) << "\n";
    std::cout << "valdr_randomx_hash=" << hex(valdrHash.data(), valdrHash.size()) << "\n";
    std::cout << "status=PASS\n";
    return 0;
}
