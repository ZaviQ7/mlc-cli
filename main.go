package main

import (
	"errors"
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

func main() {
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
