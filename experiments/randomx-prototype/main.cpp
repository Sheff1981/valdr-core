#include <array>
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

std::vector<unsigned char> decodeHex(const std::string& value) {
    if ((value.size() % 2) != 0) return {};
    std::vector<unsigned char> out;
    out.reserve(value.size() / 2);
    for (std::size_t i = 0; i < value.size(); i += 2) {
        unsigned int byte = 0;
        std::istringstream in(value.substr(i, 2));
        in >> std::hex >> byte;
        if (in.fail()) return {};
        out.push_back(static_cast<unsigned char>(byte));
    }
    return out;
}

void appendBE32(std::vector<unsigned char>& out, std::uint32_t value) {
    for (int i = 3; i >= 0; --i) {
        out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
    }
}

void appendBE64(std::vector<unsigned char>& out, std::uint64_t value) {
    for (int i = 7; i >= 0; --i) {
        out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
    }
}

void appendStringBE64(std::vector<unsigned char>& out, const std::string& value) {
    appendBE64(out, static_cast<std::uint64_t>(value.size()));
    out.insert(out.end(), value.begin(), value.end());
}

void appendLE32(std::vector<unsigned char>& out, std::uint32_t value) {
    for (int i = 0; i < 4; ++i) {
        out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
    }
}

void appendLE64(std::vector<unsigned char>& out, std::uint64_t value) {
    for (int i = 0; i < 8; ++i) {
        out.push_back(static_cast<unsigned char>((value >> (8 * i)) & 0xff));
    }
}

void appendText(std::vector<unsigned char>& out, const std::string& text) {
    appendLE32(out, static_cast<std::uint32_t>(text.size()));
    out.insert(out.end(), text.begin(), text.end());
}

void appendPattern(std::vector<unsigned char>& out, unsigned char start, std::size_t count) {
    for (std::size_t i = 0; i < count; ++i) {
        out.push_back(static_cast<unsigned char>(start + i));
    }
}

// Historical prototype vector retained for regression continuity.
std::vector<unsigned char> sampleMiningBlob() {
    std::vector<unsigned char> out;
    appendText(out, "VALDR/RandomX/Mainnet/v1");
    appendText(out, "valdr-mainnet-1");
    appendLE32(out, 3);
    appendLE64(out, 262800);
    appendPattern(out, 0x00, 32);
    appendPattern(out, 0x20, 32);
    appendLE64(out, 1800000000ULL);
    appendPattern(out, 0x40, 32);
    appendLE64(out, 0x0102030405060708ULL);
    return out;
}

// Cross-language reproduction of the Go powblob candidate:
//
//   pow_blob = "VALDR/RANDOMX/POW/V1\x00" || HeaderBytesChecked()
//
// HeaderBytesChecked() currently uses big-endian integers, length-prefixed
// previous/Merkle hex strings, 32 raw target bytes, nonce, ChainID and ExtraData.
std::vector<unsigned char> mainnetCandidateMiningBlob() {
    std::vector<unsigned char> out;

    const std::string domain("VALDR/RANDOMX/POW/V1", 20);
    out.insert(out.end(), domain.begin(), domain.end());
    out.push_back(0x00);

    appendBE32(out, 2);
    appendBE64(out, 262800);

    appendStringBE64(
        out,
        "0000000000000000000000000000000000000000000000000000000000000000"
    );
    appendStringBE64(
        out,
        "2020202020202020202020202020202020202020202020202020202020202020"
    );

    appendBE64(out, 1800000000ULL);

    for (int i = 0; i < 32; ++i) {
        out.push_back(0x40);
    }

    appendBE64(out, 0x0102030405060708ULL);
    appendStringBE64(out, "valdr-mainnet-1");
    appendStringBE64(out, "");

    return out;
}

bool hashWithKey(const void* key,
                 std::size_t keySize,
                 const std::vector<unsigned char>& input,
                 std::array<unsigned char, RANDOMX_HASH_SIZE>& output) {
    randomx_flags flags = randomx_get_flags();
    flags = static_cast<randomx_flags>(flags | RANDOMX_FLAG_SECURE);

    randomx_cache* cache = randomx_alloc_cache(flags);
    if (cache == nullptr) return false;

    randomx_init_cache(cache, key, keySize);

    randomx_vm* vm = randomx_create_vm(flags, cache, nullptr);
    if (vm == nullptr) {
        randomx_release_cache(cache);
        flags = RANDOMX_FLAG_DEFAULT;
        cache = randomx_alloc_cache(flags);
        if (cache == nullptr) return false;
        randomx_init_cache(cache, key, keySize);
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

bool hashWithStringKey(const std::string& key,
                       const std::vector<unsigned char>& input,
                       std::array<unsigned char, RANDOMX_HASH_SIZE>& output) {
    return hashWithKey(key.data(), key.size(), input, output);
}

} // namespace

int main() {
    // Official RandomX v1 compatibility vector.
    const std::string officialKey = "test key 000";
    const std::string officialInputText = "This is a test";
    std::vector<unsigned char> officialInput(officialInputText.begin(), officialInputText.end());

    std::array<unsigned char, RANDOMX_HASH_SIZE> officialHash{};
    if (!hashWithStringKey(officialKey, officialInput, officialHash)) {
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

    // Historical VALDR prototype vector.
    const std::string valdrSeed = "VALDR/RandomX/Mainnet/v1/epoch-0";
    const auto blob = sampleMiningBlob();
    std::array<unsigned char, RANDOMX_HASH_SIZE> valdrHash{};

    if (!hashWithStringKey(valdrSeed, blob, valdrHash)) {
        std::cerr << "VALDR historical vector initialization failed\n";
        return 4;
    }

    std::cout << "valdr_blob_bytes=" << blob.size() << "\n";
    std::cout << "valdr_blob_hex=" << hex(blob.data(), blob.size()) << "\n";
    std::cout << "valdr_randomx_hash=" << hex(valdrHash.data(), valdrHash.size()) << "\n";

    // Candidate Mainnet mining-blob vector tied to experiments/mainnet-powblob.
    const std::string epoch0SeedHex =
        "0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9";
    const auto epoch0Seed = decodeHex(epoch0SeedHex);
    if (epoch0Seed.size() != 32) {
        std::cerr << "invalid epoch-0 seed vector\n";
        return 5;
    }

    const auto mainnetBlob = mainnetCandidateMiningBlob();
    if (mainnetBlob.size() != 256) {
        std::cerr << "candidate mining blob length mismatch: " << mainnetBlob.size() << "\n";
        return 6;
    }

    std::array<unsigned char, RANDOMX_HASH_SIZE> mainnetPowHash{};
    if (!hashWithKey(
            epoch0Seed.data(),
            epoch0Seed.size(),
            mainnetBlob,
            mainnetPowHash)) {
        std::cerr << "Mainnet candidate RandomX initialization failed\n";
        return 7;
    }

    std::cout << "valdr_mainnet_powblob_bytes=" << mainnetBlob.size() << "\n";
    std::cout << "valdr_mainnet_powblob_hex=" << hex(mainnetBlob.data(), mainnetBlob.size()) << "\n";
    std::cout << "valdr_mainnet_epoch0_seed=" << epoch0SeedHex << "\n";
    std::cout << "valdr_mainnet_randomx_hash="
              << hex(mainnetPowHash.data(), mainnetPowHash.size()) << "\n";
    std::cout << "status=PASS\n";
    return 0;
}
