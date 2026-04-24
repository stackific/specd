package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// validCustomType matches alphanumeric values with optional spaces.
var validCustomType = regexp.MustCompile(`^[A-Za-z0-9 ]+$`)

// highlight renders text in bold with a color that adapts to the terminal
// background — visible on both dark and light themes.
var highlight = lipgloss.NewStyle().Bold(true).Foreground(
	lipgloss.AdaptiveColor{Light: "5", Dark: "6"}, // magenta on light, cyan on dark
)

// initCmd implements `specd init [project-path]`.
// It creates the specd project folder, writes the .specd.json marker,
// saves the username globally, and installs skills for selected AI providers.
var initCmd = &cobra.Command{
	Use:   "init [project-path]",
	Short: "Initialize specd in a project",
	Long:  "Creates a folder where specd stores its project-level files. Optionally specify a project path (defaults to current directory).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	// Register flags. All are optional — missing values trigger interactive prompts.
	initCmd.Flags().String("folder", "", "folder name for specd project files (default: \"specd\")")
	initCmd.Flags().String("username", "", "your username")
	initCmd.Flags().Bool("skip-skills", false, "skip skills installation")
	rootCmd.AddCommand(initCmd)
}

func runInit(c *cobra.Command, args []string) error {
	// Default to current directory if no project path is provided.
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Resolve to an absolute path so output messages are unambiguous.
	absProject, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolving project path: %w", err)
	}

	// Create the project directory if it doesn't exist yet.
	if err := os.MkdirAll(absProject, 0o755); err != nil { //nolint:gosec // project dir is part of a VCS repo, must be world-readable
		return fmt.Errorf("creating project directory: %w", err)
	}

	// Guard: refuse to re-initialize an already-initialized project.
	existing, err := LoadProjectConfig(absProject)
	if err != nil {
		return fmt.Errorf("checking existing config: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("specd is already initialized in %s\nRemove %s and %s/ to re-initialize",
			absProject,
			filepath.Join(absProject, ProjectMarker),
			filepath.Join(absProject, existing.Folder))
	}

	folder, username, err := resolveInputs(c)
	if err != nil {
		return err
	}

	specTypes, err := promptSpecTypes(c)
	if err != nil {
		return err
	}

	taskStages, err := promptTaskStages(c)
	if err != nil {
		return err
	}

	// Create the specd project folder inside the project directory.
	specdPath := filepath.Join(absProject, folder)
	if err := os.MkdirAll(specdPath, 0o755); err != nil { //nolint:gosec // specd folder is part of a VCS repo, must be world-readable
		return fmt.Errorf("creating folder: %w", err)
	}

	// Determine where to store the username.
	projectUsername, err := saveUsername(username)
	if err != nil {
		return err
	}

	// Write the .specd.json marker with all project settings.
	if err := SaveProjectConfig(absProject, &ProjectConfig{
		Folder:           folder,
		Username:         projectUsername,
		SpecTypes:        specTypes,
		TaskStages:       taskStages,
		TopSearchResults: TopSearchResults,
		SearchWeights: SearchWeights{
			Title:   BM25WeightTitle,
			Summary: BM25WeightSummary,
			Body:    BM25WeightBody,
		},
	}); err != nil {
		return err
	}

	// Initialize the cache database at the project root (.specd.cache).
	if err := InitDB(absProject, specTypes, taskStages); err != nil {
		return err
	}

	fmt.Printf("Initialized specd at %s/\n", specdPath)

	skipSkills, _ := c.Flags().GetBool("skip-skills")
	if !skipSkills {
		fmt.Println()
		if err := promptAndInstallSkills(); err != nil {
			return err
		}
	}

	printPostInitMessage()

	return nil
}

// resolveInputs gathers the folder name and username from flags, defaults,
// and interactive prompts. Returns validated, non-empty values.
func resolveInputs(c *cobra.Command) (folder, username string, err error) {
	folder, _ = c.Flags().GetString("folder")
	folderProvided := c.Flags().Changed("folder")
	if !folderProvided {
		folder = DefaultFolder
	}

	username, _ = c.Flags().GetString("username")
	usernameProvided := c.Flags().Changed("username")

	defaultSource := ""
	if !usernameProvided {
		username, defaultSource = resolveDefaultUsername()
	}

	if err := promptMissingInputs(folderProvided, usernameProvided, &folder, &username, defaultSource); err != nil {
		return "", "", err
	}

	if folder == "" {
		return "", "", fmt.Errorf("folder name cannot be empty")
	}
	if username == "" {
		return "", "", fmt.Errorf("username cannot be empty")
	}

	return folder, username, nil
}

// saveUsername decides where to store the username:
//   - No global username yet: save to both global and project config.
//   - Global username exists and user chose the same: no project override needed.
//   - Global username exists but user chose different: save only to project config.
//
// Returns the project-level username override (empty string means use global).
func saveUsername(username string) (string, error) {
	cfg, err := LoadGlobalConfig()
	if err != nil {
		return "", err
	}

	if cfg.Username == "" {
		// First time — set the global username.
		cfg.Username = username
		if err := SaveGlobalConfig(cfg); err != nil {
			return "", err
		}
		fmt.Printf("Username set globally: %s\n", username)
		return "", nil // no project override needed
	}

	if cfg.Username == username {
		// Same as global — no override needed.
		fmt.Printf("Username: %s (global)\n", username)
		return "", nil
	}

	// Different from global — store as project-level override only.
	fmt.Printf("Username: %s (project-level, global remains %q)\n", username, cfg.Username)
	return username, nil
}

// promptAndInstallSkills prompts for providers and install level, then installs.
func promptAndInstallSkills() error {
	selectedProviders, level, err := promptSkillsInstall()
	if err != nil {
		return err
	}
	return installSkills(selectedProviders, level)
}

// printPostInitMessage prints git instructions, warnings, and next-steps.
func printPostInitMessage() {
	fmt.Println()
	warn := lipgloss.NewStyle().Bold(true).Foreground(
		lipgloss.AdaptiveColor{Light: "1", Dark: "9"},
	)

	// Git instructions.
	fmt.Printf("Add %s to your .gitignore — it is a local cache and must not be committed.\n", highlight.Render(CacheDBFile))
	fmt.Printf("Commit and push %s and the %s/ folder — they are your project config and specs.\n", highlight.Render(ProjectMarker), highlight.Render(DefaultFolder))

	// Warning.
	fmt.Println()
	fmt.Printf("%s Do not manually edit spec types or task stages in %s — it may break your project.\n", warn.Render("Warning:"), ProjectMarker)

	// Next step.
	fmt.Printf("\nRun %s to start the Web UI.\n", highlight.Render("specd serve"))
}

// resolveDefaultUsername checks the global config and git for a username default.
// Returns the username and a description of where it came from.
func resolveDefaultUsername() (username, source string) {
	cfg, err := LoadGlobalConfig()
	if err == nil && cfg.Username != "" {
		return cfg.Username, "from global config"
	}

	out, err := exec.Command("git", "config", "user.name").Output()
	if err == nil {
		gitUser := strings.TrimSpace(string(out))
		if gitUser != "" {
			return gitUser, "detected from git"
		}
	}

	return "", ""
}

// promptMissingInputs shows an interactive form for folder and/or username
// when they weren't provided via flags.
func promptMissingInputs(folderProvided, usernameProvided bool, folder, username *string, defaultSource string) error {
	if folderProvided && usernameProvided {
		return nil
	}

	var fields []huh.Field

	if !folderProvided {
		fields = append(fields, huh.NewInput().
			Title("Folder name for specd project files").
			Description("specd will store its project-level files here").
			Value(folder))
	}

	if !usernameProvided {
		fields = append(fields, huh.NewInput().
			Title("Username").
			Description(defaultSource).
			Value(username))
	}

	if len(fields) > 0 {
		form := huh.NewForm(huh.NewGroup(fields...))
		if err := form.Run(); err != nil {
			return err
		}
	}

	return nil
}

// promptSpecTypes lets the user choose spec types interactively, or returns
// defaults when all flags are provided (non-interactive mode).
func promptSpecTypes(c *cobra.Command) ([]string, error) {
	// Non-interactive: if folder, username, and skip-skills are all provided,
	// use the defaults without prompting.
	if c.Flags().Changed("folder") && c.Flags().Changed("username") {
		return slugAll(DefaultSpecTypes), nil
	}

	// Build the multi-select options — all selected by default.
	specOptions := make([]huh.Option[string], len(DefaultSpecTypes))
	for i, st := range DefaultSpecTypes {
		specOptions[i] = huh.NewOption(st, st)
	}
	var selectedSpecs []string
	selectedSpecs = append(selectedSpecs, DefaultSpecTypes...)

	// Additional custom spec types entered as free text.
	var customSpecs string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Spec types").
				Description("Select at least one").
				Options(specOptions...).
				Value(&selectedSpecs),
			huh.NewInput().
				Title("Custom spec types (optional)").
				Description("Comma-separated, alphanumeric only").
				Value(&customSpecs),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Parse and validate custom spec types.
	if customSpecs != "" {
		for _, raw := range strings.Split(customSpecs, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if !validCustomType.MatchString(raw) {
				return nil, fmt.Errorf("invalid spec type %q: only alphanumeric values allowed", raw)
			}
			selectedSpecs = append(selectedSpecs, raw)
		}
	}

	if len(selectedSpecs) == 0 {
		return nil, fmt.Errorf("at least one spec type must be selected")
	}

	return slugAll(selectedSpecs), nil
}

// promptTaskStages lets the user choose task stages interactively, or returns
// defaults when all flags are provided (non-interactive mode).
func promptTaskStages(c *cobra.Command) ([]string, error) {
	// Non-interactive: use all stages.
	if c.Flags().Changed("folder") && c.Flags().Changed("username") {
		all := make([]string, 0, len(RequiredTaskStages)+len(OptionalTaskStages))
		all = append(all, RequiredTaskStages...)
		all = append(all, OptionalTaskStages...)
		return slugAll(all), nil
	}

	// Only the optional stages are toggleable — required ones are always included.
	optOptions := make([]huh.Option[string], len(OptionalTaskStages))
	for i, ts := range OptionalTaskStages {
		optOptions[i] = huh.NewOption(ts, ts)
	}
	// Default: all optional stages selected.
	var selectedOptional []string
	selectedOptional = append(selectedOptional, OptionalTaskStages...)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Task stages").
				Description("Backlog, Todo, In progress, Done are always included").
				Options(optOptions...).
				Value(&selectedOptional),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Combine required + selected optional stages.
	all := make([]string, 0, len(RequiredTaskStages)+len(selectedOptional))
	all = append(all, RequiredTaskStages...)
	all = append(all, selectedOptional...)
	return slugAll(all), nil
}

// slugAll converts a slice of display strings to slugs.
func slugAll(values []string) []string {
	slugs := make([]string, len(values))
	for i, v := range values {
		slugs[i] = ToSlug(v)
	}
	return slugs
}
