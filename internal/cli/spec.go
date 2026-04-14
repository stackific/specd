package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	// new-spec
	var (
		specTitle   string
		specType    string
		specSummary string
		specBody    string
	)

	newSpecCmd := &cobra.Command{
		Use:   "new-spec",
		Short: "Create a new spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.NewSpec(workspace.NewSpecInput{
				Title:   specTitle,
				Type:    specType,
				Summary: specSummary,
				Body:    specBody,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Printf("Created %s at %s\n", result.ID, result.Path)
			}
			return nil
		},
	}

	newSpecCmd.Flags().StringVar(&specTitle, "title", "", "spec title (required)")
	newSpecCmd.Flags().StringVar(&specType, "type", "", "spec type: business, technical, non-technical (required)")
	newSpecCmd.Flags().StringVar(&specSummary, "summary", "", "one-line summary (required)")
	newSpecCmd.Flags().StringVar(&specBody, "body", "", "markdown body")
	newSpecCmd.MarkFlagRequired("title")
	newSpecCmd.MarkFlagRequired("type")
	newSpecCmd.MarkFlagRequired("summary")
	rootCmd.AddCommand(newSpecCmd)

	// read (handles both specs and tasks)
	var (
		withTasks     bool
		withLinks     bool
		withProgress  bool
		withDeps      bool
		withCriteria  bool
		withCitations bool
	)

	readCmd := &cobra.Command{
		Use:   "read <id>",
		Short: "Read a spec or task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]

			if isSpecID(id) {
				spec, err := w.ReadSpec(id)
				if err != nil {
					return err
				}

				response := map[string]any{"spec": spec}

				if withTasks {
					tasks, err := w.ListTasks(workspace.ListTasksFilter{SpecID: id})
					if err != nil {
						return err
					}
					response["tasks"] = tasks
				}

				if jsonOutput {
					printJSON(response)
				} else {
					printSpec(spec)
				}
			} else if isTaskID(id) {
				task, err := w.ReadTask(id)
				if err != nil {
					return err
				}

				response := map[string]any{"task": task}

				if withCriteria {
					criteria, err := w.ListCriteria(task.ID)
					if err != nil {
						return err
					}
					response["criteria"] = criteria
				}

				if jsonOutput {
					printJSON(response)
				} else {
					printTask(task)
				}
			} else {
				return fmt.Errorf("invalid ID format: %s (expected SPEC-N or TASK-N)", id)
			}
			return nil
		},
	}

	readCmd.Flags().BoolVar(&withTasks, "with-tasks", false, "include tasks (spec only)")
	readCmd.Flags().BoolVar(&withLinks, "with-links", false, "include linked items")
	readCmd.Flags().BoolVar(&withProgress, "with-progress", false, "include progress")
	readCmd.Flags().BoolVar(&withDeps, "with-deps", false, "include dependencies (task only)")
	readCmd.Flags().BoolVar(&withCriteria, "with-criteria", false, "include criteria (task only)")
	readCmd.Flags().BoolVar(&withCitations, "with-citations", false, "include citations")
	rootCmd.AddCommand(readCmd)

	// list
	var (
		listType      string
		listStatus    string
		listSpecID    string
		listLinkedTo  string
		listDependsOn string
		listCreatedBy string
		listLimit     int
	)

	listCmd := &cobra.Command{
		Use:   "list <specs|tasks>",
		Short: "List specs or tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			switch args[0] {
			case "specs":
				specs, err := w.ListSpecs(workspace.ListSpecsFilter{
					Type:     listType,
					LinkedTo: listLinkedTo,
					Limit:    listLimit,
				})
				if err != nil {
					return err
				}
				if jsonOutput {
					printJSON(specs)
				} else {
					for _, s := range specs {
						fmt.Printf("%-10s [%s] %s\n", s.ID, s.Type, s.Title)
					}
					if len(specs) == 0 {
						fmt.Println("No specs found.")
					}
				}

			case "tasks":
				tasks, err := w.ListTasks(workspace.ListTasksFilter{
					SpecID:    listSpecID,
					Status:    listStatus,
					LinkedTo:  listLinkedTo,
					DependsOn: listDependsOn,
					CreatedBy: listCreatedBy,
					Limit:     listLimit,
				})
				if err != nil {
					return err
				}
				if jsonOutput {
					printJSON(tasks)
				} else {
					for _, t := range tasks {
						fmt.Printf("%-10s [%-20s] %s\n", t.ID, t.Status, t.Title)
					}
					if len(tasks) == 0 {
						fmt.Println("No tasks found.")
					}
				}

			default:
				return fmt.Errorf("unknown list kind: %s (use 'specs' or 'tasks')", args[0])
			}
			return nil
		},
	}

	listCmd.Flags().StringVar(&listType, "type", "", "filter specs by type")
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter tasks by status")
	listCmd.Flags().StringVar(&listSpecID, "spec-id", "", "filter tasks by parent spec")
	listCmd.Flags().StringVar(&listLinkedTo, "linked-to", "", "filter by linked item")
	listCmd.Flags().StringVar(&listDependsOn, "depends-on", "", "filter tasks by dependency")
	listCmd.Flags().StringVar(&listCreatedBy, "created-by", "", "filter tasks by creator")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "max results")
	rootCmd.AddCommand(listCmd)
}

// isSpecID returns true if the ID looks like "SPEC-N".
func isSpecID(id string) bool {
	return len(id) > 5 && id[:5] == "SPEC-"
}

// isTaskID returns true if the ID looks like "TASK-N".
func isTaskID(id string) bool {
	return len(id) > 5 && id[:5] == "TASK-"
}

// printSpec writes a human-readable spec summary to stdout.
func printSpec(s *workspace.Spec) {
	fmt.Printf("%s: %s\n", s.ID, s.Title)
	fmt.Printf("Type: %s\n", s.Type)
	fmt.Printf("Summary: %s\n", s.Summary)
	fmt.Printf("Path: %s\n", s.Path)
	if s.Body != "" {
		fmt.Printf("\n%s\n", s.Body)
	}
}

// printTask writes a human-readable task summary to stdout.
func printTask(t *workspace.Task) {
	fmt.Printf("%s: %s\n", t.ID, t.Title)
	fmt.Printf("Status: %s\n", t.Status)
	fmt.Printf("Spec: %s\n", t.SpecID)
	fmt.Printf("Summary: %s\n", t.Summary)
	fmt.Printf("Path: %s\n", t.Path)
	if t.Body != "" {
		fmt.Printf("\n%s\n", t.Body)
	}
}
