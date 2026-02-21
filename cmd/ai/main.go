package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/queone/utl"
	"github.com/spf13/cobra"
)

const (
	program_name    = "ai"
	program_version = "1.0.1"
	ollamaHost      = "http://localhost:11434"
	defaultModel    = "mistral-nemo"
)

type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatResponse struct {
	Message       ChatMessage `json:"message"`
	Done          bool        `json:"done"`
	TotalDuration int64       `json:"total_duration"`
}

type WebSearchResult struct {
	Title   string
	URL     string
	Content string
}

type OllamaListResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func getAvailableModels() ([]ModelInfo, error) {
	resp, err := http.Get(ollamaHost + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	var listResp OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to parse models: %w", err)
	}

	return listResp.Models, nil
}

func findSmallestModel(models []ModelInfo) string {
	if len(models) == 0 {
		return defaultModel
	}
	smallest := models[0]
	for _, m := range models[1:] {
		if m.Size < smallest.Size {
			smallest = m
		}
	}
	return smallest.Name
}

func modelExists(models []ModelInfo, name string) bool {
	for _, m := range models {
		if m.Name == name {
			return true
		}
	}
	return false
}

func listModels(models []ModelInfo) {
	fmt.Printf("%-30s  %10s\n", "NAME", "SIZE")
	for _, m := range models {
		fmt.Printf("%-30s  %10s\n", m.Name, formatBytes(m.Size))
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func runAI(model string, query string, useWeb bool, debug bool, maxResults int, searchResults []WebSearchResult) error {
	userMessage := query
	if useWeb {
		// Use pre-fetched results or search if not provided
		var results []WebSearchResult
		var err error
		if searchResults != nil {
			results = searchResults
		} else {
			results, err = searchWeb(query, maxResults)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", utl.Red("[web search error: "+err.Error()+"]"))
			}
		}

		if debug && searchResults == nil {
			// Only show debug output on first search (when results weren't pre-fetched)
			fmt.Fprintf(os.Stderr, "\n%s\n", utl.Blu("=== WEB SEARCH DEBUG ==="))
			fmt.Fprintf(os.Stderr, "Found %d results\n", len(results))
			if len(results) > 0 {
				for i, r := range results {
					fmt.Fprintf(os.Stderr, "\n[%d] %s\n", i+1, r.Title)
					fmt.Fprintf(os.Stderr, "    URL: %s\n", r.URL)
					fmt.Fprintf(os.Stderr, "    Snippet (%d chars): %s\n", len(r.Content), r.Content)
				}
			}
			fmt.Fprintf(os.Stderr, "%s\n\n", utl.Blu("=== END DEBUG ==="))
		}

		if len(results) > 0 {
			context := formatSearchContext(results)
			userMessage = fmt.Sprintf("Use this web search context to answer the question:\n\n%s\n\nQuestion: %s", context, query)
			fmt.Fprintf(os.Stderr, "%s\n", utl.Gra("[web search context loaded]"))
		} else if !debug && err == nil {
			fmt.Fprintf(os.Stderr, "%s\n", utl.Gra("[no web search results]"))
		}
	}

	request := OllamaChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Stream: true,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		ollamaHost+"/api/chat",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w", ollamaHost, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(errBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	var totalDuration int64
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chatResp OllamaChatResponse
		if err := json.Unmarshal(line, &chatResp); err != nil {
			continue
		}

		if chatResp.Message.Content != "" {
			fmt.Print(chatResp.Message.Content)
		}

		if chatResp.Done && chatResp.TotalDuration > 0 {
			totalDuration = chatResp.TotalDuration
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	fmt.Println()
	if totalDuration > 0 {
		secs := float64(totalDuration) / 1e9
		fmt.Fprintf(os.Stderr, "%s\n", utl.Gre(fmt.Sprintf("total duration:       %.9fs", secs)))
	}

	return nil
}

func searchWeb(query string, maxResults int) ([]WebSearchResult, error) {
	param, err := NewSearchParam(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create search param: %w", err)
	}

	results, err := Search(param, maxResults)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	if results == nil {
		return []WebSearchResult{}, fmt.Errorf("search returned nil results")
	}

	if len(*results) == 0 {
		return []WebSearchResult{}, nil
	}

	// Convert []SearchResult to []WebSearchResult
	webResults := make([]WebSearchResult, 0, len(*results))
	for _, r := range *results {
		webResults = append(webResults, WebSearchResult{
			Title:   r.Title,
			URL:     r.Link,
			Content: r.Snippet,
		})
	}

	// Limit results if maxResults > 0
	if maxResults > 0 && len(webResults) > maxResults {
		webResults = webResults[:maxResults]
	}

	return webResults, nil
}

func formatSearchContext(results []WebSearchResult) string {
	var buf strings.Builder
	buf.WriteString("Web Search Results:\n")
	for i, r := range results {
		fmt.Fprintf(&buf, "\n[%d] %s\n", i+1, r.Title)
		fmt.Fprintf(&buf, "    URL: %s\n", r.URL)
		fmt.Fprintf(&buf, "    %s\n", r.Content)
	}
	return buf.String()
}

func printUsage() {
	n := utl.Whi2(program_name)
	v := program_version
	usage := fmt.Sprintf("%s v%s\n"+
		"Ollama CLI with optional web search\n"+
		"\n"+
		"%s\n"+
		"  %s [options] <query>\n"+
		"\n"+
		"%s\n"+
		"  -m, --model MODEL      Ollama model to use (default: %s)\n"+
		"  -w, --web              Enable web search before answering\n"+
		"  -a, --all-models       Run query across all available models\n"+
		"  -d, --debug            Show web search results before feeding to model\n"+
		"  -r, --results N        Max web search results to use (0 = all, default: 10)\n"+
		"  -v, --version          Show this help message and exit\n"+
		"\n"+
		"%s\n"+
		"  %s \"What is Go?\"\n"+
		"  %s -m qwen3:4b \"Who is the current US president?\"\n"+
		"  %s -w \"Latest developments in AI\"\n"+
		"  %s -a \"Compare model outputs\"\n"+
		"  %s -w -d -r 3 \"What happened today?\"\n",
		n, v, utl.Whi2("Usage"), n, utl.Whi2("Options"),
		defaultModel, utl.Whi2("Examples"), n, n, n, n, n)
	fmt.Print(usage)
}

func runCLI() {
	var model string
	var useWeb bool
	var allModels bool
	var debug bool
	var maxResults int
	var showVersion bool

	root := &cobra.Command{
		Use:   program_name,
		Short: "Ollama CLI with optional web search",
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				printUsage()
				return
			}

			if len(args) == 0 {
				printUsage()
				return
			}

			// Get available models
			models, err := getAvailableModels()
			if err != nil {
				log.Fatal(err)
			}

			if len(models) == 0 {
				log.Fatal("no models available in Ollama")
			}

			query := strings.Join(args, " ")

			// Do web search once if enabled
			var searchResults []WebSearchResult
			if useWeb {
				var err error
				searchResults, err = searchWeb(query, maxResults)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", utl.Red("[web search error: "+err.Error()+"]"))
				}

				if debug {
					fmt.Fprintf(os.Stderr, "\n%s\n", utl.Blu("=== WEB SEARCH DEBUG ==="))
					fmt.Fprintf(os.Stderr, "Found %d results\n", len(searchResults))
					if len(searchResults) > 0 {
						for i, r := range searchResults {
							fmt.Fprintf(os.Stderr, "\n[%d] %s\n", i+1, r.Title)
							fmt.Fprintf(os.Stderr, "    URL: %s\n", r.URL)
							fmt.Fprintf(os.Stderr, "    Snippet (%d chars): %s\n", len(r.Content), r.Content)
						}
					}
					fmt.Fprintf(os.Stderr, "%s\n\n", utl.Blu("=== END DEBUG ==="))
				}
			}

			// Run across all models if -a flag set
			if allModels {
				for _, m := range models {
					fmt.Fprintf(os.Stderr, "%s\n", utl.Gre("==== MODEL "+m.Name+" ===="))
					if err := runAI(m.Name, query, useWeb, false, maxResults, searchResults); err != nil {
						fmt.Fprintf(os.Stderr, "error with %s: %v\n", m.Name, err)
					}
					fmt.Println()
				}
				return
			}

			// Determine which model to use
			selectedModel := model
			if selectedModel == "" || selectedModel == defaultModel {
				// Use smallest model if none specified
				selectedModel = findSmallestModel(models)
			} else {
				// Check if specified model exists
				if !modelExists(models, selectedModel) {
					fmt.Printf("model '%s' not found. Available models:\n\n", selectedModel)
					listModels(models)
					os.Exit(1)
				}
			}

			if err := runAI(selectedModel, query, useWeb, debug, maxResults, nil); err != nil {
				log.Fatal(err)
			}
		},
	}

	root.Flags().StringVarP(&model, "model", "m", defaultModel, "Ollama model to use")
	root.Flags().BoolVarP(&useWeb, "web", "w", false, "Enable web search before answering")
	root.Flags().BoolVarP(&allModels, "all-models", "a", false, "Run query across all available models")
	root.Flags().BoolVarP(&debug, "debug", "d", false, "Show web search results before feeding to model")
	root.Flags().IntVarP(&maxResults, "results", "r", 10, "Max web search results to use (0 = all)")
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version and usage")

	// Disable default help
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true
	root.Flags().BoolP("help", "h", false, "")
	root.Flags().Lookup("help").Hidden = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	runCLI()
}
