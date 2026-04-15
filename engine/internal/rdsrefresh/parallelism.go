/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// rdsInstanceClassPrefix is stripped to derive the instance size.
	rdsInstanceClassPrefix = "db."

	// minParallelJobs is the minimum parallelism level.
	minParallelJobs = 1

	// dumpParallelDivisor scales down vCPU count for dump parallelism.
	// pg_dump is I/O-bound against the remote RDS instance, so using half the vCPUs avoids over-subscribing.
	dumpParallelDivisor = 2
)

// instanceSizeVCPUs maps AWS instance size suffixes to their vCPU count.
// sizes micro and small only appear in t-family (burstable) RDS instances, both with 2 vCPUs.
// from large upward, the mapping is consistent across instance families (m5, m6g, r5, r6g, c5, etc.).
// "metal" is handled separately via metalFamilyVCPUs since its vCPU count depends on the family.
//
//nolint:mnd
var instanceSizeVCPUs = map[string]int{
	"micro":    2,
	"small":    2,
	"medium":   2,
	"large":    2,
	"xlarge":   4,
	"2xlarge":  8,
	"3xlarge":  12,
	"4xlarge":  16,
	"6xlarge":  24,
	"8xlarge":  32,
	"9xlarge":  36,
	"10xlarge": 40,
	"12xlarge": 48,
	"16xlarge": 64,
	"18xlarge": 72,
	"24xlarge": 96,
	"32xlarge": 128,
	"48xlarge": 192,
}

// metalFamilyVCPUs maps the base family (without storage suffix like "d") to metal vCPU counts.
// storage variants (m5d, r6gd, etc.) share the same vCPU count as their base family.
//
//nolint:mnd
var metalFamilyVCPUs = map[string]int{
	"m5": 96, "r5": 96, "c5": 96, "x1": 128,
	"m6g": 64, "r6g": 64, "c6g": 64, "x2g": 64,
	"m6i": 128, "r6i": 128, "c6i": 128,
	"m7g": 64, "r7g": 64, "c7g": 64,
	"m7i": 192, "r7i": 192, "c7i": 192,
}

// ParallelismConfig holds the computed parallelism levels for dump and restore.
type ParallelismConfig struct {
	DumpJobs    int
	RestoreJobs int
}

// ResolveParallelism determines the optimal parallelism levels for pg_dump and pg_restore.
// dump parallelism is half the vCPU count of the RDS clone instance (I/O-bound, conservative).
// restore parallelism is the full local CPU count (CPU-bound for index rebuilds).
// note: dump parallelism > 1 is incompatible with immediateRestore mode.
// when immediateRestore is enabled, the caller should ignore DumpJobs or cap it at 1.
func ResolveParallelism(cfg *Config) (*ParallelismConfig, error) {
	vcpus, err := resolveRDSInstanceVCPUs(cfg.RDSClone.InstanceClass)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RDS instance vCPUs: %w", err)
	}

	dumpJobs := vcpus / dumpParallelDivisor
	if dumpJobs < minParallelJobs {
		dumpJobs = minParallelJobs
	}

	restoreJobs := resolveLocalVCPUs()

	log.Msg("auto-parallelism: dump jobs =", dumpJobs, "(RDS clone vCPUs/2), restore jobs =", restoreJobs, "(local vCPUs)")

	return &ParallelismConfig{
		DumpJobs:    dumpJobs,
		RestoreJobs: restoreJobs,
	}, nil
}

// resolveRDSInstanceVCPUs estimates the vCPU count for the given RDS instance class
// by parsing the instance size suffix (e.g. "xlarge" from "db.m5.xlarge").
// the mapping covers standard AWS size naming used across RDS instance families.
// for "metal" instances, the family is used to look up family-specific vCPU counts.
// if the size is not recognized, it attempts to parse a numeric multiplier prefix
// (e.g. "2xlarge" → 8 vCPUs).
func resolveRDSInstanceVCPUs(instanceClass string) (int, error) {
	family, size, err := parseInstanceClass(instanceClass)
	if err != nil {
		return 0, err
	}

	if size == "metal" {
		return resolveMetalVCPUs(family, instanceClass)
	}

	if vcpus, ok := instanceSizeVCPUs[size]; ok {
		return vcpus, nil
	}

	// handle unlisted NUMxlarge sizes by parsing the multiplier
	vcpus, err := parseXlargeMultiplier(size)
	if err != nil {
		return 0, fmt.Errorf("unknown instance size %q in class %q", size, instanceClass)
	}

	return vcpus, nil
}

// parseInstanceClass extracts the family and size from an RDS instance class.
// for example, "db.m5.xlarge" → ("m5", "xlarge"), "db.r6gd.metal" → ("r6gd", "metal").
func parseInstanceClass(instanceClass string) (string, string, error) {
	if !strings.HasPrefix(instanceClass, rdsInstanceClassPrefix) {
		return "", "", fmt.Errorf("invalid RDS instance class %q: expected %q prefix", instanceClass, rdsInstanceClassPrefix)
	}

	withoutPrefix := strings.TrimPrefix(instanceClass, rdsInstanceClassPrefix)

	// format is "family.size", e.g. "m5.xlarge" or "r6g.2xlarge"
	parts := strings.SplitN(withoutPrefix, ".", 2)

	const expectedParts = 2
	if len(parts) != expectedParts || parts[1] == "" {
		return "", "", fmt.Errorf("invalid RDS instance class %q: expected format db.<family>.<size>", instanceClass)
	}

	return parts[0], parts[1], nil
}

// resolveMetalVCPUs looks up the vCPU count for a metal instance by family.
// storage variants (e.g. "m5d", "r6gd") are mapped to their base family ("m5", "r6g").
func resolveMetalVCPUs(family, instanceClass string) (int, error) {
	if vcpus, ok := metalFamilyVCPUs[family]; ok {
		return vcpus, nil
	}

	// strip trailing "d" storage suffix and retry (e.g. "m5d" → "m5", "r6gd" → "r6g")
	base := strings.TrimSuffix(family, "d")
	if base != family {
		if vcpus, ok := metalFamilyVCPUs[base]; ok {
			return vcpus, nil
		}
	}

	return 0, fmt.Errorf("unknown metal instance family %q in class %q", family, instanceClass)
}

// parseXlargeMultiplier handles NUMxlarge patterns not in the static map.
// for example, "5xlarge" → 5 * 4 = 20 vCPUs.
func parseXlargeMultiplier(size string) (int, error) {
	idx := strings.Index(size, "xlarge")
	if idx <= 0 {
		return 0, fmt.Errorf("not an xlarge variant: %q", size)
	}

	multiplier, err := strconv.Atoi(size[:idx])
	if err != nil {
		return 0, fmt.Errorf("invalid multiplier in %q: %w", size, err)
	}

	const vcpusPerXlarge = 4

	return multiplier * vcpusPerXlarge, nil
}

// resolveLocalVCPUs returns the number of logical CPUs available on the local machine.
// uses runtime.NumCPU() which reads from /proc/cpuinfo on Linux
// (the target platform for DBLab Engine).
func resolveLocalVCPUs() int {
	cpus := runtime.NumCPU()
	if cpus < minParallelJobs {
		return minParallelJobs
	}

	return cpus
}
