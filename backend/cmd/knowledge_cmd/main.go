package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"memoryflow/internal/ai/agent/knowledge_pipeline"
	"memoryflow/internal/ai/workflow/memory_analyze"
	"memoryflow/internal/bootstrap"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/service"
)

type knowledgeOutput struct {
	MemoryID        uint           `json:"memory_id"`
	Summary         string         `json:"summary,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Mood            string         `json:"mood,omitempty"`
	ImportanceScore float64        `json:"importance_score,omitempty"`
	Project         *model.Project `json:"project,omitempty"`
	Indexed         bool           `json:"indexed"`
	Warnings        []string       `json:"warnings,omitempty"`
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "knowledge_cmd failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	text := flag.String("text", "", "text memory content")
	filePath := flag.String("file", "", "read text memory content from file")
	imagePath := flag.String("image", "", "local image path")
	source := flag.String("source", "manual", "source: manual, file, image, github, web")
	projectName := flag.String("project", "", "optional project name")
	projectID := flag.Uint("project-id", 0, "optional project id; higher priority than -project")
	tags := flag.String("tags", "", "comma-separated tags")
	analyze := flag.Bool("analyze", true, "run LLM/image analysis")
	index := flag.Bool("index", false, "reindex memories into vector store")
	jsonOutput := flag.Bool("json", false, "print JSON output")
	flag.Parse()

	content, warnings, err := loadContent(*text, *filePath)
	if err != nil {
		return err
	}
	image := strings.TrimSpace(*imagePath)
	if content == "" && image == "" {
		flag.Usage()
		return errors.New("one of -text, -file or -image is required")
	}
	if err := validateSource(*source); err != nil {
		return err
	}
	if image != "" {
		warnings = append(warnings, "image pipeline is not fully implemented; saved image path only")
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap app failed: %w", err)
	}
	defer app.Close(ctx)

	project, err := resolveProject(ctx, app.ProjectService, *projectID, *projectName)
	if err != nil {
		return err
	}
	if project != nil {
		warnings = append(warnings, "project is resolved for CLI output; memory-project association is not persisted in the current schema")
	}

	item, err := createMemory(ctx, app.MemoryService, content, image, *source)
	if err != nil {
		return fmt.Errorf("create memory failed: %w", err)
	}

	output := knowledgeOutput{MemoryID: item.ID, Project: project, Warnings: warnings}
	if len(parseTags(*tags)) > 0 {
		output.Tags = parseTags(*tags)
	}

	if *analyze {
		result, err := app.MemoryAnalyzeWorkflow.Invoke(ctx, memory_analyze.AnalyzeInput{
			MemoryID: item.ID, Type: item.Type, ContentText: item.ContentText, ImageURL: item.ImageURL, Location: item.Location, OccurredAt: item.OccurredAt,
		})
		if err != nil {
			return fmt.Errorf("analyze memory failed: %w", err)
		}
		output.Summary = result.Summary
		output.Tags = mergeTags(result.Tags, output.Tags)
		output.Mood = result.Mood
		output.ImportanceScore = result.ImportanceScore
		if err := updateAnalysis(ctx, app.MemoryService, item.ID, output); err != nil {
			return fmt.Errorf("save analysis failed: %w", err)
		}
	} else if len(output.Tags) > 0 {
		if err := updateAnalysis(ctx, app.MemoryService, item.ID, output); err != nil {
			return fmt.Errorf("save tags failed: %w", err)
		}
	}

	if *index {
		indexOutput, err := app.KnowledgePipeline.ReindexAll(ctx, knowledge_pipeline.ReindexInput{BatchSize: 100})
		if err != nil {
			output.Warnings = append(output.Warnings, "index failed: "+err.Error())
		} else if indexOutput.Failed > 0 {
			output.Warnings = append(output.Warnings, fmt.Sprintf("index completed with failures: total=%d succeeded=%d failed=%d", indexOutput.Total, indexOutput.Succeeded, indexOutput.Failed))
			output.Indexed = indexOutput.Succeeded > 0
		} else {
			output.Indexed = true
		}
	}

	if *jsonOutput {
		return printJSON(output)
	}
	printHuman(output)
	return nil
}

func loadContent(text string, filePath string) (string, []string, error) {
	content := strings.TrimSpace(text)
	if file := strings.TrimSpace(filePath); file != "" {
		bytes, err := os.ReadFile(file)
		if err != nil {
			return "", nil, fmt.Errorf("read file %s failed: %w", file, err)
		}
		if content != "" {
			return "", nil, errors.New("-text and -file cannot be used together")
		}
		content = strings.TrimSpace(string(bytes))
		return content, nil, nil
	}
	return content, nil, nil
}

func validateSource(source string) error {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "manual", "file", "image", "github", "web":
		return nil
	default:
		return fmt.Errorf("invalid -source %q, expected manual/file/image/github/web", source)
	}
}

func resolveProject(ctx context.Context, projects *service.ProjectService, id uint, name string) (*model.Project, error) {
	if id > 0 {
		project, err := projects.GetProjectByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("project-id %d not found: %w", id, err)
		}
		return project, nil
	}
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	project, err := projects.FindProjectByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("project %q not found: %w", name, err)
	}
	return project, nil
}

func createMemory(ctx context.Context, memories *service.MemoryService, content string, image string, source string) (*model.MemoryItem, error) {
	location := strings.TrimSpace(source)
	if strings.TrimSpace(image) != "" {
		return memories.CreateImageMemory(ctx, &service.CreateImageMemoryRequest{
			ContentText: content,
			ImageURL:    image,
			Location:    location,
		})
	}
	return memories.CreateTextMemory(ctx, &service.CreateTextMemoryRequest{
		ContentText: content,
		Location:    location,
	})
}

func updateAnalysis(ctx context.Context, memories *service.MemoryService, id uint, output knowledgeOutput) error {
	tagsJSON, err := json.Marshal(output.Tags)
	if err != nil {
		return err
	}
	return memories.UpdateAnalysis(ctx, id, output.Summary, string(tagsJSON), output.Mood, output.ImportanceScore)
}

func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if tag := strings.TrimSpace(part); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func mergeTags(primary []string, extra []string) []string {
	seen := make(map[string]bool)
	merged := make([]string, 0, len(primary)+len(extra))
	for _, tag := range append(primary, extra...) {
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[tag] {
			seen[tag] = true
			merged = append(merged, tag)
		}
	}
	return merged
}

func printHuman(output knowledgeOutput) {
	fmt.Printf("memory_id: %d\n", output.MemoryID)
	if output.Summary != "" {
		fmt.Printf("summary: %s\n", output.Summary)
	}
	if len(output.Tags) > 0 {
		fmt.Printf("tags: %s\n", strings.Join(output.Tags, ", "))
	}
	if output.Mood != "" {
		fmt.Printf("mood: %s\n", output.Mood)
	}
	if output.Project != nil {
		fmt.Printf("project: %s\n", output.Project.Name)
	}
	fmt.Printf("indexed: %v\n", output.Indexed)
	for _, warning := range output.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
	}
}

func printJSON(v any) error {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output failed: %w", err)
	}
	fmt.Println(string(bytes))
	return nil
}
