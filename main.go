package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
)

const (
	Green   = "\033[32m"
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Success = "[" + Green + "✓" + Reset + "] "
	Error   = "[" + Red + "✗" + Reset + "] "
)

func cliError(msg string, err error) {
	fmt.Println(Error + msg + err.Error())
	os.Exit(1)
}

func printUsage() {
	fmt.Println(`MLC-LLM CLI - Interactive and Non-Interactive Modes

Usage:
  mlc-cli                      Launch interactive mode
  mlc-cli <command> [flags]    Run a command non-interactively

Commands:
  build       Build TVM/MLC from source and install wheels
  run         Chat with a model
  compile     Pre-compile a model library (.so) to skip JIT at runtime
  quantize    Convert raw model weights to MLC format with quantization

Examples:
  # Full build + install on mac with Metal
  mlc-cli build --os mac --action full --tvm-source bundled --metal y --build-wheels y

  # Install pre-built wheels only
  mlc-cli build --os mac --action install-wheels

  # Run a model
  mlc-cli run --os mac --model-name Llama-3-8B --device metal --profile default

  # Compile a model library
  mlc-cli compile --os mac --model models/Llama-3-8B --quant q4f16_1 --device metal --output dist/libs/llama.so

  # Quantize a model
  mlc-cli quantize --os mac --model models/Llama-3-8B --quant q4f16_1 --output dist/Llama-q4f16_1-MLC --template llama-3 --device metal

Run 'mlc-cli <command> --help' for more information on a command.`)
}

func main() {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		runNonInteractive(os.Args[1], os.Args[2:])
		return
	}

	fmt.Println("Welcome to MLC-LLM CLI!")

	prompt := promptui.Select{
		Label: "Options",
		Items: []string{
			"Build (build TVM/MLC from source and install wheels)",
			"Run (chat with a model)",
			"Compile Model (build the .so file to skip JIT compilation at runtime)",
			"Quantize Model (convert raw model weights to MLC format with quantization)",
			"Deploy",
		},
	}
	_, selection, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Println("\nExiting...")
			os.Exit(0)
		}
		cliError("Error getting selection: ", err)
	}

	if strings.HasPrefix(selection, "Build") {
		buildPrompt := promptui.Select{
			Label: "Select build action",
			Items: []string{
				"Full Build + Install",
				"Build Only (no install)",
				"Install Wheels Only",
			},
		}
		_, buildAction, err := buildPrompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				fmt.Println("\nExiting...")
				os.Exit(0)
			}
			cliError("Error getting selection: ", err)
		}

		platform := CreatePlatform()

		switch buildAction {
		case "Full Build + Install":
			platform.ConfigureGitHubRepo()
			platform.ConfigureRepoAction()
			platform.ConfigureBuildOptions()
			platform.ConfigureWheelBuildOption()
			promptInstall(platform, "cuda")
			promptBuild(platform, "tvm")
			promptBuild(platform, "mlc")
			promptInstall(platform, "tvm")
			promptInstall(platform, "mlc")

		case "Build Only (no install)":
			platform.ConfigureGitHubRepo()
			platform.ConfigureRepoAction()
			platform.ConfigureBuildOptions()
			platform.ConfigureWheelBuildOption()
			promptInstall(platform, "cuda")
			promptBuild(platform, "tvm")
			promptBuild(platform, "mlc")

		case "Install Wheels Only":
			fmt.Println("\nInstalling pre-built wheels into the CLI environment...")
			platform.install("tvm")
			platform.install("mlc")
			fmt.Println("\n" + Success + "Wheels installed successfully.")
		}

	} else if strings.HasPrefix(selection, "Run") {
		platform := CreatePlatform()
		platform.ConfigureModel()
		platform.run()

	} else if strings.HasPrefix(selection, "Compile Model") {
		platform := CreatePlatform()
		promptCompileModel(platform)

	} else if strings.HasPrefix(selection, "Quantize Model") {
		platform := CreatePlatform()
		promptQuantizeModel(platform)

	} else if strings.HasPrefix(selection, "Deploy") {
	}
}

func runNonInteractive(command string, args []string) {
	switch command {
	case "build":
		runBuildCmd(args)
	case "run":
		runRunCmd(args)
	case "compile":
		runCompileCmd(args)
	case "quantize":
		runQuantizeCmd(args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runBuildCmd(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	osFlag := fs.String("os", detectOS(), "Operating system (mac or linux)")
	action := fs.String("action", "full", "Build action: full, build-only, install-wheels")
	tvmBuildEnv := fs.String("tvm-build-env", "tvm-build-venv", "Conda env for TVM build")
	mlcBuildEnv := fs.String("mlc-build-env", "mlc-build-venv", "Conda env for MLC build")
	cliEnv := fs.String("cli-env", "mlc-cli-venv", "Conda env for CLI/install")
	gitRepo := fs.String("git-repo", "https://github.com/mlc-ai/mlc-llm", "GitHub repo URL")
	tvmSource := fs.String("tvm-source", "bundled", "TVM source: bundled, relax, or custom")
	buildWheels := fs.String("build-wheels", "y", "Build Python wheels (y/n)")
	forceClone := fs.String("force-clone", "n", "Force re-clone repos (y/n)")
	cuda := fs.String("cuda", "n", "Enable CUDA (y/n)")
	cudaArch := fs.String("cuda-arch", "86", "CUDA compute capability")
	cutlass := fs.String("cutlass", "n", "Enable Cutlass (y/n)")
	cublas := fs.String("cublas", "n", "Enable cuBLAS (y/n)")
	flashInfer := fs.String("flash-infer", "n", "Enable FlashInfer (y/n)")
	rocm := fs.String("rocm", "n", "Enable ROCm (y/n)")
	vulkan := fs.String("vulkan", "n", "Enable Vulkan (y/n)")
	metal := fs.String("metal", "n", "Enable Metal (y/n)")
	openCL := fs.String("opencl", "n", "Enable OpenCL (y/n)")
	err := fs.Parse(args)
	if err != nil {
		printUsage()
	}

	p := &Platform{
		OperatingSystem: *osFlag,
		TVMBuildEnv:     *tvmBuildEnv,
		MLCBuildEnv:     *mlcBuildEnv,
		CliEnv:          *cliEnv,
		GitHubRepo:      *gitRepo,
		TVMSource:       *tvmSource,
		BuildWheels:     *buildWheels,
		ForceClone:      *forceClone,
		CUDA:            *cuda,
		CUDAArch:        *cudaArch,
		Cutlass:         *cutlass,
		CuBLAS:          *cublas,
		FlashInfer:      *flashInfer,
		ROCM:            *rocm,
		Vulkan:          *vulkan,
		Metal:           *metal,
		OpenCL:          *openCL,
	}

	switch *action {
	case "full":
		fmt.Println("🚀 Full Build + Install (non-interactive)")
		p.build("tvm")
		p.build("mlc")
		p.install("tvm")
		p.install("mlc")
		fmt.Println("\n" + Success + "Full build + install complete.")
	case "build-only":
		fmt.Println("🔨 Build Only (non-interactive)")
		p.build("tvm")
		p.build("mlc")
		fmt.Println("\n" + Success + "Build complete.")
	case "install-wheels":
		fmt.Println("📦 Install Wheels Only (non-interactive)")
		p.install("tvm")
		p.install("mlc")
		fmt.Println("\n" + Success + "Wheels installed.")
	default:
		fmt.Printf("Unknown build action: %s (use: full, build-only, install-wheels)\n", *action)
		os.Exit(1)
	}
}

func runRunCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	osFlag := fs.String("os", detectOS(), "Operating system (mac or linux)")
	cliEnv := fs.String("cli-env", "mlc-cli-venv", "Conda env for CLI")
	modelURL := fs.String("model-url", "", "Model Git URL (to clone)")
	modelName := fs.String("model-name", "", "Model directory name in models/")
	device := fs.String("device", "", "Device: metal, cuda, vulkan, etc.")
	profile := fs.String("profile", "default", "Compute profile: really-low, low, default, high")
	modelLib := fs.String("model-lib", "", "Path to pre-compiled model library (.so)")
	err := fs.Parse(args)
	if err != nil {
		printUsage()
	}

	if *device == "" {
		if *osFlag == "mac" {
			*device = "metal"
		} else {
			*device = "cuda"
		}
	}

	var overrides string
	switch *profile {
	case "really-low":
		overrides = "context_window_size=10240;prefill_chunk_size=512"
	case "low":
		overrides = "context_window_size=20480;prefill_chunk_size=1024"
	case "high":
		overrides = "context_window_size=81920;prefill_chunk_size=4096"
	default:
		overrides = ""
	}

	fmt.Printf("🚀 Running model (non-interactive) on %s...\n", *device)
	cmd := exec.Command("bash", "scripts/"+*osFlag+"_run_model.sh", *cliEnv, *modelURL, *modelName, *device, overrides, *modelLib)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cliError("Run failed: ", err)
	}
}

func runCompileCmd(args []string) {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)
	osFlag := fs.String("os", detectOS(), "Operating system (mac or linux)")
	cliEnv := fs.String("cli-env", "mlc-cli-venv", "Conda env for CLI")
	model := fs.String("model", "", "Model path (required)")
	quant := fs.String("quant", "q4f16_1", "Quantization: q4f16_1, q4f16_ft, q4f32_1, q3f16_1, q8f16_1, q0f16, q0f32")
	device := fs.String("device", "", "Device: metal, cuda, vulkan, etc.")
	output := fs.String("output", "", "Output path for compiled .so library")
	err := fs.Parse(args)
	if err != nil {
		printUsage()
	}

	if *model == "" {
		fmt.Println("Error: --model is required")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *device == "" {
		if *osFlag == "mac" {
			*device = "metal"
		} else {
			*device = "cuda"
		}
	}
	if *output == "" {
		*output = "dist/libs/" + baseName(*model) + "-" + *quant + "-" + *device + ".so"
	}

	fmt.Printf("🔧 Compiling model [%s] quant=[%s] device=[%s] (non-interactive)...\n", *model, *quant, *device)
	cmd := exec.Command("bash", "scripts/"+*osFlag+"_compile_model.sh", *cliEnv, *model, *quant, *device, *output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cliError("Compilation failed: ", err)
	}
	fmt.Println("\n" + Success + "Model compiled! Library saved to " + *output)
}

func runQuantizeCmd(args []string) {
	fs := flag.NewFlagSet("quantize", flag.ExitOnError)
	osFlag := fs.String("os", detectOS(), "Operating system (mac or linux)")
	cliEnv := fs.String("cli-env", "mlc-cli-venv", "Conda env for CLI")
	model := fs.String("model", "", "Model path (required)")
	quant := fs.String("quant", "q4f16_1", "Quantization: q4f16_1, q4f16_ft, q4f32_1, q3f16_1, q8f16_1, q0f16, q0f32")
	output := fs.String("output", "", "Output directory for quantized model")
	template := fs.String("template", "llama-3", "Conversation template: llama-3, chatml, mistral_default, phi-2, gemma, qwen2")
	device := fs.String("device", "", "Device for quantization: metal, cuda, etc.")
	err := fs.Parse(args)
	if err != nil {
		printUsage()
	}

	if *model == "" {
		fmt.Println("Error: --model is required")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *device == "" {
		if *osFlag == "mac" {
			*device = "metal"
		} else {
			*device = "cuda"
		}
	}
	if *output == "" {
		*output = "dist/" + baseName(*model) + "-" + *quant + "-MLC"
	}

	fmt.Printf("🚀 Quantizing [%s] with [%s] on [%s] (non-interactive)...\n", *model, *quant, *device)

	cmd := exec.Command("conda", "run", "--no-capture-output", "-n", *cliEnv,
		"python", "-m", "mlc_llm", "convert_weight",
		*model,
		"--quantization", *quant,
		"--device", *device,
		"-o", *output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cliError("Quantization failed: ", err)
	}

	fmt.Println("\n📄 Generating config...")
	cmdConfig := exec.Command("conda", "run", "--no-capture-output", "-n", *cliEnv,
		"python", "-m", "mlc_llm", "gen_config", *model,
		"--quantization", *quant,
		"--conv-template", *template,
		"-o", *output)
	cmdConfig.Stdout = os.Stdout
	cmdConfig.Stderr = os.Stderr
	if err := cmdConfig.Run(); err != nil {
		cliError("Config generation failed: ", err)
	}

	fmt.Println("\n" + Success + "Quantization complete! Model saved to " + *output)
}

func detectOS() string {
	if platform, _ := exec.Command("uname").Output(); strings.TrimSpace(string(platform)) == "Darwin" {
		return "mac"
	}
	return "linux"
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func promptQuantizeModel(platform *Platform) {
	// List models from models/ directory
	modelDirs := getLocalModelDirectories()
	var modelPath string

	if len(modelDirs) > 0 {
		items := append(modelDirs, "Enter path manually")
		modelSelectPrompt := promptui.Select{
			Label: "Select a model from models/ directory",
			Items: items,
		}
		_, modelSelection, err := modelSelectPrompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				fmt.Println("\nExiting...")
				os.Exit(0)
			}
			cliError("Selection failed", err)
		}
		if modelSelection == "Enter path manually" {
			manualPrompt := promptui.Prompt{
				Label: "Enter Hugging Face Model Path (or local path)",
			}
			modelPath, err = manualPrompt.Run()
			if err != nil {
				cliError("Input failed", err)
			}
		} else {
			modelPath = "models/" + modelSelection
		}
	} else {
		fmt.Println("No models found in models/ directory.")
		manualPrompt := promptui.Prompt{
			Label: "Enter Hugging Face Model Path (or local path)",
		}
		var err error
		modelPath, err = manualPrompt.Run()
		if err != nil {
			cliError("Input failed", err)
		}
	}

	// List all supported quantization options
	quantOptions := []string{
		"q4f16_1   (4-bit group quantization, float16)",
		"q4f16_ft  (4-bit FasterTransformer, float16)",
		"q4f32_1   (4-bit group quantization, float32)",
		"q3f16_1   (3-bit group quantization, float16)",
		"q8f16_1   (8-bit group quantization, float16)",
		"q0f16     (No quantization, float16)",
		"q0f32     (No quantization, float32)",
	}
	promptQuant := promptui.Select{
		Label: "Select Quantization",
		Items: quantOptions,
	}
	_, quantResult, err := promptQuant.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Println("\nExiting...")
			os.Exit(0)
		}
		cliError("Selection failed", err)
	}

	// Extract quantization code (everything before the first space)
	quantCode := quantResult
	for i, ch := range quantResult {
		if ch == ' ' {
			quantCode = quantResult[:i]
			break
		}
	}

	// Default output directory based on model name and quantization
	modelName := modelPath
	for i := len(modelPath) - 1; i >= 0; i-- {
		if modelPath[i] == '/' {
			modelName = modelPath[i+1:]
			break
		}
	}
	defaultOutput := "dist/" + modelName + "-" + quantCode + "-MLC"

	promptOut := promptui.Prompt{
		Label:   "Enter Output Directory",
		Default: defaultOutput,
	}
	outputDir, err := promptOut.Run()
	if err != nil {
		cliError("Input failed", err)
	}

	promptTemplate := promptui.Select{
		Label: "Select Conversation Template",
		Items: []string{"llama-3", "chatml", "mistral_default", "phi-2", "gemma", "qwen2"},
	}
	_, convTemplate, err := promptTemplate.Run()
	if err != nil {
		cliError("Selection failed", err)
	}

	platform.configureDevice()

	fmt.Printf("\n🚀 Starting Quantization [%s] using env [%s] on device [%s]...\n", quantCode, platform.CliEnv, platform.Device)

	cmd := exec.Command("conda", "run", "--no-capture-output", "-n", platform.CliEnv,
		"python", "-m", "mlc_llm", "convert_weight",
		modelPath,
		"--quantization", quantCode,
		"--device", platform.Device,
		"-o", outputDir)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cliError("Quantization failed: ", err)
	}

	fmt.Println("\n📄 Generating config...")
	cmdConfig := exec.Command("conda", "run", "--no-capture-output", "-n", platform.CliEnv,
		"python", "-m", "mlc_llm", "gen_config", modelPath,
		"--quantization", quantCode,
		"--conv-template", convTemplate,
		"-o", outputDir)

	cmdConfig.Stdout = os.Stdout
	cmdConfig.Stderr = os.Stderr

	if err := cmdConfig.Run(); err != nil {
		cliError("Config generation failed: ", err)
	}

	fmt.Println("\n" + Success + "Quantization Complete! Model saved to " + outputDir)
}

func promptCompileModel(platform *Platform) {
	// List models from models/ directory
	modelDirs := getLocalModelDirectories()
	var modelPath string

	if len(modelDirs) > 0 {
		items := append(modelDirs, "Enter path manually")
		modelSelectPrompt := promptui.Select{
			Label: "Select a model from models/ directory",
			Items: items,
		}
		_, modelSelection, err := modelSelectPrompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				fmt.Println("\nExiting...")
				os.Exit(0)
			}
			cliError("Selection failed", err)
		}
		if modelSelection == "Enter path manually" {
			manualPrompt := promptui.Prompt{
				Label: "Enter model path",
			}
			modelPath, err = manualPrompt.Run()
			if err != nil {
				cliError("Input failed", err)
			}
		} else {
			modelPath = "models/" + modelSelection
		}
	} else {
		fmt.Println("No models found in models/ directory.")
		manualPrompt := promptui.Prompt{
			Label: "Enter model path",
		}
		var err error
		modelPath, err = manualPrompt.Run()
		if err != nil {
			cliError("Input failed", err)
		}
	}

	// List all supported quantization options
	quantOptions := []string{
		"q4f16_1   (4-bit group quantization, float16)",
		"q4f16_ft  (4-bit FasterTransformer, float16)",
		"q4f32_1   (4-bit group quantization, float32)",
		"q3f16_1   (3-bit group quantization, float16)",
		"q8f16_1   (8-bit group quantization, float16)",
		"q0f16     (No quantization, float16)",
		"q0f32     (No quantization, float32)",
	}
	quantPrompt := promptui.Select{
		Label: "Select Quantization",
		Items: quantOptions,
	}
	_, quantResult, err := quantPrompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Println("\nExiting...")
			os.Exit(0)
		}
		cliError("Selection failed", err)
	}

	// Extract quantization code (everything before the first space)
	quantCode := quantResult
	for i, ch := range quantResult {
		if ch == ' ' {
			quantCode = quantResult[:i]
			break
		}
	}

	platform.configureDevice()

	// Default output uses model name, quant, and device
	modelName := modelPath
	for i := len(modelPath) - 1; i >= 0; i-- {
		if modelPath[i] == '/' {
			modelName = modelPath[i+1:]
			break
		}
	}
	defaultOutput := "dist/libs/" + modelName + "-" + quantCode + "-" + platform.Device + ".so"

	outputPrompt := promptui.Prompt{
		Label:   "Enter output path for compiled model library",
		Default: defaultOutput,
	}
	outputPath, err := outputPrompt.Run()
	if err != nil {
		cliError("Input failed", err)
	}

	fmt.Printf("\n🔧 Compiling model [%s] with quantization [%s] for device [%s]...\n", modelPath, quantCode, platform.Device)

	cmd := exec.Command("bash", "scripts/"+platform.OperatingSystem+"_compile_model.sh",
		platform.CliEnv, modelPath, quantCode, platform.Device, outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cliError("Compilation failed: ", err)
	}

	fmt.Println("\n" + Success + "Model compiled successfully! Library saved to " + outputPath)
}

func promptInstall(platform *Platform, pkg string) {
	prompt := promptui.Select{
		Label: "Install " + pkg + "?",
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Println("\nExiting...")
			os.Exit(0)
		}
		cliError("Error getting selection: ", err)
	}
	if result == "Yes" {
		platform.install(pkg)
	}
}

func promptBuild(platform *Platform, pkg string) {
	prompt := promptui.Select{
		Label: "Build " + pkg + " from source?",
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Println("\nExiting...")
			os.Exit(0)
		}
		cliError("Error getting selection: ", err)
	}
	if result == "Yes" {
		platform.build(pkg)
	}
}
