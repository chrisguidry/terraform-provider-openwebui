package client

import (
	"context"
	"net/http"
)

// ImagesConfigForm mirrors ImagesConfig in
// backend/open_webui/routers/images.py. No field of that class carries a default,
// so every key rides on every request. An omitted key is a 422, not a key that
// keeps its stored value.
type ImagesConfigForm struct {
	EnableImageGeneration          *bool   `json:"ENABLE_IMAGE_GENERATION"`
	EnableImagePromptGeneration    *bool   `json:"ENABLE_IMAGE_PROMPT_GENERATION"`
	ImageGenerationEngine          *string `json:"IMAGE_GENERATION_ENGINE"`
	ImageGenerationModel           *string `json:"IMAGE_GENERATION_MODEL"`
	ImageSize                      *string `json:"IMAGE_SIZE"`
	ImageSteps                     *int64  `json:"IMAGE_STEPS"`
	ImagesOpenAIAPIBaseURL         *string `json:"IMAGES_OPENAI_API_BASE_URL"`
	ImagesOpenAIAPIKey             *string `json:"IMAGES_OPENAI_API_KEY"`
	ImagesOpenAIAPIVersion         *string `json:"IMAGES_OPENAI_API_VERSION"`
	ImagesOpenAIAPIParams          any     `json:"IMAGES_OPENAI_API_PARAMS"`
	Automatic1111BaseURL           *string `json:"AUTOMATIC1111_BASE_URL"`
	Automatic1111APIAuth           *string `json:"AUTOMATIC1111_API_AUTH"`
	Automatic1111Params            any     `json:"AUTOMATIC1111_PARAMS"`
	ComfyUIBaseURL                 *string `json:"COMFYUI_BASE_URL"`
	ComfyUIAPIKey                  *string `json:"COMFYUI_API_KEY"`
	ComfyUIWorkflow                *string `json:"COMFYUI_WORKFLOW"`
	ComfyUIWorkflowNodes           any     `json:"COMFYUI_WORKFLOW_NODES"`
	ImagesGeminiAPIBaseURL         *string `json:"IMAGES_GEMINI_API_BASE_URL"`
	ImagesGeminiAPIKey             *string `json:"IMAGES_GEMINI_API_KEY"`
	ImagesGeminiEndpointMethod     *string `json:"IMAGES_GEMINI_ENDPOINT_METHOD"`
	EnableImageEdit                *bool   `json:"ENABLE_IMAGE_EDIT"`
	ImageEditEngine                *string `json:"IMAGE_EDIT_ENGINE"`
	ImageEditModel                 *string `json:"IMAGE_EDIT_MODEL"`
	ImageEditSize                  *string `json:"IMAGE_EDIT_SIZE"`
	ImagesEditOpenAIAPIBaseURL     *string `json:"IMAGES_EDIT_OPENAI_API_BASE_URL"`
	ImagesEditOpenAIAPIKey         *string `json:"IMAGES_EDIT_OPENAI_API_KEY"`
	ImagesEditOpenAIAPIVersion     *string `json:"IMAGES_EDIT_OPENAI_API_VERSION"`
	ImagesEditGeminiAPIBaseURL     *string `json:"IMAGES_EDIT_GEMINI_API_BASE_URL"`
	ImagesEditGeminiAPIKey         *string `json:"IMAGES_EDIT_GEMINI_API_KEY"`
	ImagesEditComfyUIBaseURL       *string `json:"IMAGES_EDIT_COMFYUI_BASE_URL"`
	ImagesEditComfyUIAPIKey        *string `json:"IMAGES_EDIT_COMFYUI_API_KEY"`
	ImagesEditComfyUIWorkflow      *string `json:"IMAGES_EDIT_COMFYUI_WORKFLOW"`
	ImagesEditComfyUIWorkflowNodes any     `json:"IMAGES_EDIT_COMFYUI_WORKFLOW_NODES"`
}

// GetImagesConfig retrieves the image generation and image editing config.
func (c *Client) GetImagesConfig(ctx context.Context) (*ImagesConfigForm, error) {
	var resp ImagesConfigForm
	if err := c.do(ctx, http.MethodGet, "images/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetImagesConfig updates the image generation and image editing config.
func (c *Client) SetImagesConfig(ctx context.Context, form ImagesConfigForm) (*ImagesConfigForm, error) {
	var resp ImagesConfigForm
	if err := c.do(ctx, http.MethodPost, "images/config/update", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
