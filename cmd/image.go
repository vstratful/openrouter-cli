package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/api"
)

var (
	imageModel       string
	imagePrompt      string
	imageFile        string
	imageBase64      bool
	imageAspectRatio string
	imageSize        string
)

const defaultImageModel = "google/gemini-2.5-flash-image"

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Generate an image using an image-capable model",
	Long: `Generate an image using the OpenRouter API with an image-capable model.

The model must support image output modality. Use 'openrouter models --image-only'
to see available image-capable models.

Options:
  --aspect-ratio    Aspect ratio for the generated image. Available values:
                    1:1 (1024x1024), 2:3 (832x1248), 3:2 (1248x832),
                    3:4 (864x1184), 4:3 (1184x864), 4:5 (896x1152),
                    5:4 (1152x896), 9:16 (768x1344), 16:9 (1344x768),
                    21:9 (1536x672). Default: 1:1

  --size            Image resolution. Available values:
                    1K (standard), 2K (higher), 4K (highest). Default: 1K

Examples:
  openrouter image -p "A sunset over mountains" -f output.png
  openrouter image -p "A sunset" --base64
  openrouter image -p "A portrait" --aspect-ratio 9:16 -f portrait.png
  openrouter image -m openai/gpt-5-image -p "A landscape" --size 2K -f hd.png`,
	RunE: runImage,
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&imageModel, "model", "m", "", "Model to use (default: "+defaultImageModel+")")
	imageCmd.Flags().StringVarP(&imagePrompt, "prompt", "p", "", "Image generation prompt (required)")
	imageCmd.Flags().StringVarP(&imageFile, "file", "f", "", "Output file path (e.g., output.png)")
	imageCmd.Flags().BoolVar(&imageBase64, "base64", false, "Output raw base64 instead of saving to file")
	imageCmd.Flags().StringVar(&imageAspectRatio, "aspect-ratio", "", "Aspect ratio (default: 1:1)")
	imageCmd.Flags().StringVar(&imageSize, "size", "", "Image resolution (default: 1K)")

	imageCmd.MarkFlagRequired("prompt")
}

func runImage(cmd *cobra.Command, args []string) error {
	// Validate output format
	if imageFile == "" && !imageBase64 {
		return fmt.Errorf("must specify either --file or --base64 for output")
	}
	if imageFile != "" && imageBase64 {
		return fmt.Errorf("--file and --base64 are mutually exclusive")
	}

	// Use default model if not specified
	if imageModel == "" {
		imageModel = defaultImageModel
	}

	apiKey, isFirstRun, err := getAPIKey()
	if err != nil {
		return err
	}
	if isFirstRun {
		fmt.Println("\nAPI key saved. Run the command again to generate an image.")
		return nil
	}

	client := api.DefaultClient(apiKey)
	imageClient := api.ImageClient(apiKey)

	// Fetch models and validate the selected model
	models, err := client.ListModels(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	// Find the selected model and validate it supports image output (single pass)
	var selectedModel *api.Model
	var imageModels []string
	var modelExistsButNotImage bool
	for i := range models {
		if models[i].IsImageModel() {
			imageModels = append(imageModels, models[i].ID)
			if models[i].ID == imageModel {
				selectedModel = &models[i]
			}
		} else if models[i].ID == imageModel {
			modelExistsButNotImage = true
		}
	}

	if selectedModel == nil {
		if modelExistsButNotImage {
			fmt.Fprintf(os.Stderr, "Error: model '%s' does not support image output.\n\n", imageModel)
		} else {
			fmt.Fprintf(os.Stderr, "Error: model '%s' not found.\n\n", imageModel)
		}
		fmt.Fprintf(os.Stderr, "Available image-capable models:\n")
		for _, id := range imageModels {
			fmt.Fprintf(os.Stderr, "  %s\n", id)
		}
		if modelExistsButNotImage {
			return fmt.Errorf("invalid model for image generation")
		}
		return fmt.Errorf("model not found")
	}

	// Build the request
	req := &api.ChatRequest{
		Model: imageModel,
		Messages: []api.Message{
			{Role: "user", Content: imagePrompt},
		},
		Modalities: []string{"image", "text"},
	}

	// Add image config if specified
	if imageAspectRatio != "" || imageSize != "" {
		req.ImageConfig = &api.ImageConfig{
			AspectRatio: imageAspectRatio,
			Size:        imageSize,
		}
	}

	// Make the request (use imageClient with longer timeout)
	resp, err := imageClient.Chat(context.Background(), req)
	if err != nil {
		return fmt.Errorf("image generation failed: %w", err)
	}

	// Extract image from response
	if len(resp.Choices) == 0 {
		return fmt.Errorf("no response from model")
	}

	choice := resp.Choices[0]
	if len(choice.Message.Images) == 0 {
		// Check if there's text content that might explain the issue
		if choice.Message.Content != "" {
			return fmt.Errorf("no image generated. Model response: %s", choice.Message.Content)
		}
		return fmt.Errorf("no image in response")
	}

	// Get the image data URL
	imageDataURL := choice.Message.Images[0].ImageURL.URL

	// Parse the data URL to extract base64 content
	base64Data, err := parseDataURL(imageDataURL)
	if err != nil {
		return err
	}

	if imageBase64 {
		// Output raw base64
		fmt.Println(base64Data)
		return nil
	}

	// Decode and save to file
	imageBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	if err := os.WriteFile(imageFile, imageBytes, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	fmt.Printf("Image saved to %s\n", imageFile)

	// Print any accompanying text
	if choice.Message.Content != "" {
		fmt.Printf("\nModel response: %s\n", choice.Message.Content)
	}

	return nil
}

// parseDataURL extracts the base64 data from a data URL.
// Expected format: data:<mediatype>;base64,<data>
func parseDataURL(dataURL string) (string, error) {
	const dataURLPrefix = "data:"
	if !strings.HasPrefix(dataURL, dataURLPrefix) {
		return "", fmt.Errorf("unexpected image URL format: must start with 'data:'")
	}

	const base64Marker = ";base64,"
	idx := strings.Index(dataURL, base64Marker)
	if idx == -1 {
		return "", fmt.Errorf("no base64 data found in image URL")
	}

	return dataURL[idx+len(base64Marker):], nil
}
