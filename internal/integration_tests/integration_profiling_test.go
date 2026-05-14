//go:build integration && profiling

package integration_tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fraenky8/tables-to-go/v2/internal/cmd"
)

const profilingTestDirectory = "profiling"

func TestIntegrationProfiling(t *testing.T) {
	t.Run("mysql 8", func(t *testing.T) {
		testSettings := newMySQLSettings("8", "mysql", profilingTestDirectory)
		args := []string{
			"tables-to-go",
			"-t", "mysql",
			"-u", "root",
			"-p", "mysecretpassword",
			"-d", "public",
			"-h", "localhost",
			"-port", "3306",
			"-of", filepath.Join("mysql", profilingTestDirectory, outputDirectoryName),
		}

		runProfilingScenario(t, testSettings, args)
	})

	t.Run("postgres 18", func(t *testing.T) {
		testSettings := newPostgresSettings("18", "postgres", profilingTestDirectory)
		args := []string{
			"tables-to-go",
			"-t", "pg",
			"-u", "postgres",
			"-p", "mysecretpassword",
			"-d", "postgres",
			"-s", "public",
			"-h", "localhost",
			"-port", "5432",
			"-sslmode", "disable",
			"-of", filepath.Join("postgres", profilingTestDirectory, outputDirectoryName),
		}

		runProfilingScenario(t, testSettings, args)
	})

	t.Run("sqlite 3", func(t *testing.T) {
		testSettings := newSQLiteSettings("sqlite3", profilingTestDirectory)
		args := []string{
			"tables-to-go",
			"-t", "sqlite3",
			"-d", filepath.Join("sqlite3", "database.db"),
			"-of", filepath.Join("sqlite3", profilingTestDirectory, outputDirectoryName),
		}

		runProfilingScenario(t, testSettings, args)
	})
}

func runProfilingScenario(t *testing.T, testSettings *testSettings, args []string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	parsedArgs, err := cmd.NewArgs(args, &stderr)
	if err != nil {
		t.Fatalf("could not parse args %q: %v", args, err)
	}
	testSettings.Settings = parsedArgs.Settings

	db := setupDatabase(t, testSettings)
	t.Cleanup(func() {
		_ = os.RemoveAll(testSettings.Settings.OutputFilePath)
	})

	loadTestData(t, db.SQLDriver(), testSettings)

	err = os.MkdirAll(testSettings.Settings.OutputFilePath, 0755)
	if err != nil {
		t.Fatalf("could not create output file path: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("could not close setup database connection before run: %v", err)
	}

	c := cmd.New(cmd.VersionInfo{}, db)

	// Warm-up run to reduce profiling noise from one-time setup costs.
	err = os.RemoveAll(testSettings.Settings.OutputFilePath)
	if err != nil {
		t.Fatalf("could not cleanup output file path before warm-up: %v", err)
	}
	err = os.MkdirAll(testSettings.Settings.OutputFilePath, 0755)
	if err != nil {
		t.Fatalf("could not recreate output file path before warm-up: %v", err)
	}

	err = c.Run(t.Context(), args, &stdout, &stderr)
	if err != nil {
		t.Fatalf("warm-up run failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	profilesPath := filepath.Join(testSettings.filepath, testSettings.testDirectory, "profiles")
	err = os.MkdirAll(profilesPath, 0755)
	if err != nil {
		t.Fatalf("could not create profiles path: %v", err)
	}

	err = os.RemoveAll(testSettings.Settings.OutputFilePath)
	if err != nil {
		t.Fatalf("could not cleanup output file path before measured run: %v", err)
	}
	err = os.MkdirAll(testSettings.Settings.OutputFilePath, 0755)
	if err != nil {
		t.Fatalf("could not recreate output file path before measured run: %v", err)
	}

	err = runWithProfiles(t, profilesPath, func() error {
		return c.Run(t.Context(), args, &stdout, &stderr)
	})
	if err != nil {
		t.Fatalf("measured run failed: %v", err)
	}

	assert.Regexp(t, "^$", stdout.String())
	assert.Regexp(t, `(?s).*running for.*done!.*`, stderr.String())

	verifyProfileFile(t, filepath.Join(profilesPath, "cpu.pprof"))
	verifyProfileFile(t, filepath.Join(profilesPath, "heap.pprof"))
	verifyProfileFile(t, filepath.Join(profilesPath, "allocs.pprof"))
}

func runWithProfiles(t *testing.T, profilesPath string, run func() error) error {
	t.Helper()

	if err := os.MkdirAll(profilesPath, 0755); err != nil {
		return fmt.Errorf("could not ensure profiles path: %w", err)
	}

	cpuPath := filepath.Join(profilesPath, "cpu.pprof")
	heapPath := filepath.Join(profilesPath, "heap.pprof")
	allocsPath := filepath.Join(profilesPath, "allocs.pprof")

	cpuFile, err := os.Create(cpuPath)
	if err != nil {
		return fmt.Errorf("could not create cpu profile %q: %w", cpuPath, err)
	}

	if err = pprof.StartCPUProfile(cpuFile); err != nil {
		_ = cpuFile.Close()
		return fmt.Errorf("could not start cpu profile: %w", err)
	}

	runErr := run()

	pprof.StopCPUProfile()
	if closeErr := cpuFile.Close(); closeErr != nil && runErr == nil {
		runErr = fmt.Errorf("could not close cpu profile file: %w", closeErr)
	}
	if runErr != nil {
		return runErr
	}

	runtime.GC()

	heapFile, err := os.Create(heapPath)
	if err != nil {
		return fmt.Errorf("could not create heap profile %q: %w", heapPath, err)
	}
	if err = pprof.WriteHeapProfile(heapFile); err != nil {
		_ = heapFile.Close()
		return fmt.Errorf("could not write heap profile: %w", err)
	}
	if err = heapFile.Close(); err != nil {
		return fmt.Errorf("could not close heap profile file: %w", err)
	}

	allocsFile, err := os.Create(allocsPath)
	if err != nil {
		return fmt.Errorf("could not create allocs profile %q: %w", allocsPath, err)
	}
	allocsProfile := pprof.Lookup("allocs")
	if allocsProfile == nil {
		_ = allocsFile.Close()
		return fmt.Errorf("could not look up allocs profile")
	}
	if err = allocsProfile.WriteTo(allocsFile, 0); err != nil {
		_ = allocsFile.Close()
		return fmt.Errorf("could not write allocs profile: %w", err)
	}
	if err = allocsFile.Close(); err != nil {
		return fmt.Errorf("could not close allocs profile file: %w", err)
	}

	return nil
}

func verifyProfileFile(t *testing.T, filePath string) {
	t.Helper()

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("profile file %q is missing: %v", filePath, err)
	}

	if info.Size() == 0 {
		t.Fatalf("profile file %q is empty", filePath)
	}
}
