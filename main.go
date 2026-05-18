package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
)

const version = "1.3.0"

var (
	red     = color.New(color.FgRed).Add(color.Underline)
	green   = color.New(color.FgGreen).Add(color.Bold)
	magenta = color.New(color.FgMagenta).Add(color.Bold)
	orange  = color.New(color.FgHiWhite, color.BgBlack).Add(color.Bold)
	blue    = color.New(color.FgBlue)
	cyan    = color.New(color.FgCyan)
	yellow  = color.New(color.FgYellow).Add(color.Italic)
)

type Config struct {
	Std        string `json:"std"`
	Timeout    int    `json:"timeout"`
	Template   string `json:"template"`
	Iterations int    `json:"iterations"`
}

func defaultConfig() Config {
	return Config{
		Std:        "c++20",
		Timeout:    60,
		Template:   "default",
		Iterations: 1000,
	}
}

func configPath() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".raga.json")
}

func loadConfig() Config {
	cfg := defaultConfig()
	path := configPath()
	if path == "" {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg Config) {
	path := configPath()
	if path == "" {
		return
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0644)
}

func isOnlyWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func fileExists(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

func spinner(done chan struct{}) {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Print("\r")
			return
		default:
			fmt.Printf("\r%s Compiling...", chars[i%len(chars)])
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}

func listTemplates() {
	dirs := []string{"."}
	if execPath, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(execPath))
	}
	seen := map[string]string{}
	for _, d := range dirs {
		tmplDir := filepath.Join(d, "templates")
		entries, err := os.ReadDir(tmplDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".cpp") {
				label := strings.TrimSuffix(name, ".cpp")
				if _, ok := seen[label]; !ok {
					info, _ := e.Info()
					size := ""
					if info != nil {
						size = fmt.Sprintf(" (%d bytes)", info.Size())
					}
					seen[label] = size
				}
			}
		}
	}
	if len(seen) == 0 {
		yellow.Println("No templates found. Create a templates/ directory with .cpp files.")
		return
	}
	fmt.Println("Available templates:")
	for name, size := range seen {
		fmt.Printf("  %s%s\n", name, size)
	}
}

func resolveTemplate(name string) (string, error) {
	locations := []string{"."}
	if execPath, err := os.Executable(); err == nil {
		locations = append(locations, filepath.Dir(execPath))
	}
	for _, loc := range locations {
		path := filepath.Join(loc, "templates", name+".cpp")
		if fileExists(path) {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}
	return "", fmt.Errorf("template %q not found", name)
}

func printUsage() {
	fmt.Println(`raga - C++ Problem Generator and Tester  v` + version + `

GENERATE:
  raga -new -name=<problem> [-template=<name>]

TEST:
  raga -test -name=<problem> [-std=<standard>] [-timeout=<sec>] [-watch] [-cases]

STRESS TEST:
  raga -stress -brute=<file> -optimized=<file> -gen=<file> [-iterations=<N>]

OTHER:
  raga -list
  raga -clean
  raga -config
  raga -version
  raga -help

FLAGS:
  -new               Generate problem files from a template
  -test              Compile and test a C++ solution against I/O files
  -stress            Stress-test an optimized solution against a brute force
  -watch             Auto re-run tests when files change (with -test)
  -cases             Run all numbered test cases (problem-1.in, problem-2.in...)
  -list              List available templates
  -clean             Remove build artifacts (.temp/, binaries)
  -config            Show current configuration
  -name=<string>     Problem name (used for .cpp, -in.txt, -out.txt)
  -template=<name>   Template to use (default: "default")
  -brute=<file>      Brute-force solution file for stress testing (no extension)
  -optimized=<file>  Optimized solution file for stress testing (no extension)
  -gen=<file>        Test case generator file for stress testing (no extension)
  -iterations=<N>    Number of stress test iterations (default: 1000)
  -std=<standard>    C++ standard: c++17, c++20, c++23 (default: "c++20")
  -timeout=<sec>     Max execution seconds per run (default: 60)
  -version           Show version and exit
  -help              Show this help message`)
}

func compileCpp(source, output, std string) error {
	cmd := exec.Command("g++", "-D", "YASH_DEBUG=fsaf", source, "-o", output, "-std="+std)
	return cmd.Run()
}

func runBinaryWithTimeout(binary string, input []byte, timeout time.Duration) (string, string, time.Duration, error) {
	cmd := exec.Command(binary)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", "", 0, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	start := time.Now()
	select {
	case <-time.After(timeout):
		cmd.Process.Kill()
		return "", "", time.Since(start), fmt.Errorf("timed out after %v", timeout)
	case err := <-done:
		return stdout.String(), stderr.String(), time.Since(start), err
	}
}

func evaluateCode(name, std string, timeoutSec int) {
	if err := os.MkdirAll(".temp", 0755); err != nil {
		red.Println("Error creating .temp directory:", err)
		os.Exit(1)
	}

	outputBinary := filepath.Join(".temp", name)
	blue.Println("Compiling", name+".cpp", "with", std, "...")

	done := make(chan struct{})
	go spinner(done)
	startCompile := time.Now()
	err := compileCpp(name+".cpp", outputBinary, std)
	close(done)
	compileDuration := time.Since(startCompile)

	if err != nil {
		fmt.Println()
		red.Println("Compilation Error:")
		cmd := exec.Command("g++", "-D", "YASH_DEBUG=fsaf", name+".cpp", "-o", outputBinary, "-std="+std)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Run()
		fmt.Println(stderr.String())
		return
	}
	blue.Println("Compiled in", compileDuration)

	inputData, err := os.ReadFile(name + "-in.txt")
	if err != nil {
		red.Println("Error reading", name+"-in.txt:", err)
		return
	}
	expectedOutput, err := os.ReadFile(name + "-out.txt")
	if err != nil {
		red.Println("Error reading", name+"-out.txt:", err)
		return
	}

	if isOnlyWhitespace(string(inputData)) {
		yellow.Println("Input file", name+"-in.txt", "is empty. Fill it with test cases and re-run.")
		return
	}
	if isOnlyWhitespace(string(expectedOutput)) {
		yellow.Println("Output file", name+"-out.txt", "is empty. Fill it with expected output and re-run.")
		return
	}

	timeout := time.Duration(timeoutSec) * time.Second
	stdout, stderr, runtime, err := runBinaryWithTimeout(outputBinary, inputData, timeout)

	if err != nil {
		if strings.Contains(err.Error(), "timed out") {
			red.Println("Timed out after", timeoutSec, "seconds.")
		} else {
			red.Println("Runtime Error:", stderr)
		}
	}

	actualOutput := strings.Split(stdout, "\n")
	expectedLines := strings.Split(string(expectedOutput), "\n")

	mismatch := false
	for i := 0; i < len(expectedLines); i++ {
		if i >= len(actualOutput) || strings.TrimSpace(actualOutput[i]) != strings.TrimSpace(expectedLines[i]) {
			if !mismatch {
				red.Println("\nMismatches:")
			}
			mismatch = true
			if i < len(actualOutput) {
				yellow.Printf("  Line %d:\n", i+1)
				orange.Printf("    Expected: %s\n    Actual:   %s\n", expectedLines[i], actualOutput[i])
			} else {
				yellow.Printf("  Line %d:\n", i+1)
				orange.Printf("    Expected: %s\n    Actual:   <no output>\n", expectedLines[i])
			}
		}
	}

	if !mismatch {
		green.Println("Success: Output matches expected result!")
	} else {
		red.Println("\nYour output:")
		for _, line := range actualOutput {
			fmt.Println(" ", line)
		}
	}
	blue.Println("Runtime:", runtime)

	os.Remove(outputBinary)
}

func evaluateCodeCases(name, std string, timeoutSec int) {
	pattern := name + "-*.in"
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		yellow.Println("No test case files found matching", pattern)
		return
	}

	passed, failed := 0, 0
	for _, inFile := range matches {
		caseName := strings.TrimSuffix(inFile, ".in")
		caseName = strings.TrimPrefix(caseName, name+"-")
		outFile := name + "-" + caseName + ".out"

		if !fileExists(outFile) {
			continue
		}

		cyan.Println("\n--- Test case:", caseName, "---")
		os.WriteFile(name+"-in.txt", mustReadFile(inFile), 0644)
		os.WriteFile(name+"-out.txt", mustReadFile(outFile), 0644)

		if err := os.MkdirAll(".temp", 0755); err != nil {
			red.Println("Error:", err)
			return
		}

		outputBinary := filepath.Join(".temp", name)
		if err := compileCpp(name+".cpp", outputBinary, std); err != nil {
			red.Println("Compilation failed for", name+".cpp")
			return
		}

		timeout := time.Duration(timeoutSec) * time.Second
		inputData := mustReadFile(inFile)
		stdout, _, runtime, err := runBinaryWithTimeout(outputBinary, inputData, timeout)

		expectedData := mustReadFile(outFile)
		actual := strings.TrimSpace(stdout)
		expected := strings.TrimSpace(string(expectedData))

		if err != nil || actual != expected {
			red.Println("FAILED (runtime:", runtime, ")")
			if err != nil {
				red.Println(" Error:", err)
			}
			if actual != expected {
				red.Println(" Expected:", expected)
				red.Println(" Actual:  ", actual)
			}
			failed++
		} else {
			green.Println("PASSED (runtime:", runtime, ")")
			passed++
		}

		os.Remove(outputBinary)
	}

	os.Remove(name + "-in.txt")
	os.Remove(name + "-out.txt")

	fmt.Println()
	cyan.Println("Results:", passed, "passed,", failed, "failed")
}

func stressTest(brute, optimized, gen, std string, iterations, timeoutSec int) {
	if err := os.MkdirAll(".temp", 0755); err != nil {
		red.Println("Error creating .temp directory:", err)
		os.Exit(1)
	}

	magenta.Println("=== Stress Test ===")
	blue.Println("Brute:    ", brute+".cpp")
	blue.Println("Optimized:", optimized+".cpp")
	blue.Println("Generator:", gen+".cpp")
	blue.Println("Iterations:", iterations)
	fmt.Println()

	bruteBin := filepath.Join(".temp", "brute_"+brute)
	optBin := filepath.Join(".temp", "opt_"+optimized)
	genBin := filepath.Join(".temp", "gen_"+gen)

	blue.Print("Compiling brute force...")
	if err := compileCpp(brute+".cpp", bruteBin, std); err != nil {
		red.Println(" FAILED")
		return
	}
	green.Println(" OK")

	blue.Print("Compiling optimized...")
	if err := compileCpp(optimized+".cpp", optBin, std); err != nil {
		red.Println(" FAILED")
		return
	}
	green.Println(" OK")

	blue.Print("Compiling generator...")
	if err := compileCpp(gen+".cpp", genBin, std); err != nil {
		red.Println(" FAILED")
		return
	}
	green.Println(" OK")

	timeout := time.Duration(timeoutSec) * time.Second
	failed := 0

	for i := 0; i < iterations; i++ {
		fmt.Printf("\rIteration %d/%d", i+1, iterations)

		genOut, _, _, err := runBinaryWithTimeout(genBin, nil, timeout)
		if err != nil {
			fmt.Printf("\rIteration %d/%d - generator failed: %v\n", i+1, iterations, err)
			failed++
			continue
		}
		input := []byte(genOut)

		bruteOut, _, _, err := runBinaryWithTimeout(bruteBin, input, timeout)
		if err != nil {
			fmt.Printf("\rIteration %d/%d - brute force failed: %v\n", i+1, iterations, err)
			failed++
			continue
		}

		optOut, _, _, err := runBinaryWithTimeout(optBin, input, timeout)
		if err != nil {
			fmt.Printf("\rIteration %d/%d - optimized failed: %v\n", i+1, iterations, err)
			failed++
			continue
		}

		if strings.TrimSpace(bruteOut) != strings.TrimSpace(optOut) {
			failed++
			fmt.Printf("\r\033[K")
			red.Printf("\nMismatch found on iteration %d!\n", i+1)
			yellow.Println("Input:")
			for _, line := range strings.Split(string(input), "\n") {
				fmt.Println(" ", line)
			}
			orange.Println("Brute output:")
			for _, line := range strings.Split(strings.TrimSpace(bruteOut), "\n") {
				fmt.Println(" ", line)
			}
			orange.Println("Optimized output:")
			for _, line := range strings.Split(strings.TrimSpace(optOut), "\n") {
				fmt.Println(" ", line)
			}

			os.MkdirAll(".stress-failures", 0755)
			os.WriteFile(fmt.Sprintf(".stress-failures/input-%d.txt", i+1), input, 0644)
			os.WriteFile(fmt.Sprintf(".stress-failures/brute-%d.txt", i+1), []byte(bruteOut), 0644)
			os.WriteFile(fmt.Sprintf(".stress-failures/optimized-%d.txt", i+1), []byte(optOut), 0644)
			yellow.Println("Saved to .stress-failures/")
			break
		}
	}

	fmt.Print("\r\033[K")
	total := iterations
	if failed > 0 && failed < iterations {
		total = failed
	}
	passed := total - failed
	fmt.Println()
	cyan.Println("Stress test complete.")
	green.Println(" Passed:", passed)
	if failed > 0 {
		red.Println(" Failed:", failed)
	}

	os.Remove(bruteBin)
	os.Remove(optBin)
	os.Remove(genBin)
}

func genCode(problemName *string, cppTemplate *string) {
	blue.Println("Generating files...")
	if *problemName == "" {
		yellow.Println("You must provide a problem name")
		return
	}

	paths := []string{
		*problemName + ".cpp",
		*problemName + "-out.txt",
		*problemName + "-in.txt",
	}

	files := make([]*os.File, 3)
	var err error
	for i, p := range paths {
		files[i], err = os.Create(p)
		if err != nil {
			red.Println("Failed to create", p+":", err)
			for j := 0; j < i; j++ {
				files[j].Close()
				os.Remove(paths[j])
			}
			return
		}
	}

	_, err = files[0].WriteString(*cppTemplate)
	if err != nil {
		red.Println("Failed to write template:", err)
	}

	for _, f := range files {
		f.Close()
	}

	green.Println("Done. Created:")
	for _, p := range paths {
		green.Println("  " + p)
	}
}

func cleanArtifacts() {
	removed := false
	dirs := []string{".temp", ".stress-failures"}
	for _, d := range dirs {
		if fileExists(d) {
			os.RemoveAll(d)
			blue.Println("Removed", d+"/")
			removed = true
		}
	}
	patterns := []string{"raga", "raga.exe", "mochi", "mochi.exe"}
	for _, p := range patterns {
		if fileExists(p) {
			os.Remove(p)
			blue.Println("Removed", p)
			removed = true
		}
	}
	if !removed {
		yellow.Println("Nothing to clean.")
	} else {
		green.Println("Clean complete.")
	}
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return data
}

func watchLoop(name, std string, timeoutSec int) {
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	files := []fileInfo{
		{path: name + ".cpp"},
		{path: name + "-in.txt"},
		{path: name + "-out.txt"},
	}

	for i := range files {
		info, err := os.Stat(files[i].path)
		if err != nil {
			red.Println("Cannot watch", files[i].path, "-", err)
			return
		}
		files[i].modTime = info.ModTime()
	}

	cyan.Println("Watching for changes. Press Ctrl+C to stop.")
	evaluateCode(name, std, timeoutSec)

	poll := time.NewTicker(500 * time.Millisecond)
	defer poll.Stop()

	for range poll.C {
		changed := false
		for i, f := range files {
			info, err := os.Stat(f.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(f.modTime) {
				files[i].modTime = info.ModTime()
				changed = true
			}
		}
		if changed {
			cyan.Println("\n--- Change detected, re-running ---")
			evaluateCode(name, std, timeoutSec)
			cyan.Println("Watching for changes. Press Ctrl+C to stop.")
		}
	}
}

func main() {
	cfg := loadConfig()

	problemName := flag.String("name", "", "problem name")
	isGenStage := flag.Bool("new", false, "generate problem files from a template")
	isTester := flag.Bool("test", false, "compile and test C++ code against I/O files")
	isWatch := flag.Bool("watch", false, "auto re-run tests when files change")
	isCases := flag.Bool("cases", false, "run all numbered test cases")
	isStress := flag.Bool("stress", false, "stress-test optimized solution against brute force")
	listTemplatesFlag := flag.Bool("list", false, "list available templates")
	cleanFlag := flag.Bool("clean", false, "remove build artifacts")
	showConfig := flag.Bool("config", false, "show current configuration")
	templateName := flag.String("template", cfg.Template, "template name to use")
	std := flag.String("std", cfg.Std, "C++ standard")
	timeoutSec := flag.Int("timeout", cfg.Timeout, "max execution time in seconds")
	bruteFile := flag.String("brute", "", "brute-force solution file (no extension)")
	optimizedFile := flag.String("optimized", "", "optimized solution file (no extension)")
	genFile := flag.String("gen", "", "test case generator file (no extension)")
	iterations := flag.Int("iterations", cfg.Iterations, "number of stress test iterations")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Usage = printUsage
	flag.Parse()

	if *showVersion {
		fmt.Println("raga version", version)
		return
	}

	if *listTemplatesFlag {
		listTemplates()
		return
	}

	if *cleanFlag {
		cleanArtifacts()
		return
	}

	if *showConfig {
		fmt.Println("Current configuration:")
		fmt.Printf("  std:        %s\n", cfg.Std)
		fmt.Printf("  timeout:    %d\n", cfg.Timeout)
		fmt.Printf("  template:   %s\n", cfg.Template)
		fmt.Printf("  iterations: %d\n", cfg.Iterations)
		fmt.Println("  config at:", configPath())
		return
	}

	if *isStress {
		if *bruteFile == "" || *optimizedFile == "" || *genFile == "" {
			red.Println("Stress testing requires -brute, -optimized, and -gen flags.")
			red.Println("  raga -stress -brute=Brute -optimized=Fast -gen=Generator")
			return
		}
		stressTest(*bruteFile, *optimizedFile, *genFile, *std, *iterations, *timeoutSec)
		return
	}

	if *isGenStage {
		templateSrc, err := resolveTemplate(*templateName)
		if err != nil {
			red.Println(err)
			listTemplates()
			return
		}
		if fileExists(*problemName+".cpp") || fileExists(*problemName+"-out.txt") || fileExists(*problemName+"-in.txt") {
			yellow.Println("Files already exist for", *problemName)
			return
		}
		genCode(problemName, &templateSrc)
		return
	}

	if *isTester {
		if *problemName == "" {
			yellow.Println("You must provide a problem name")
			return
		}
		if *isCases {
			evaluateCodeCases(*problemName, *std, *timeoutSec)
		} else if *isWatch {
			watchLoop(*problemName, *std, *timeoutSec)
		} else {
			blue.Println("Testing ...")
			evaluateCode(*problemName, *std, *timeoutSec)
			blue.Println("Test complete")
		}
		return
	}

	printUsage()
}
