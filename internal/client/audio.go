package client

import (
	"context"
	"net/http"
)

// AudioConfigForm mirrors AudioConfigUpdateForm in
// backend/open_webui/routers/audio.py, which requires both blocks.
type AudioConfigForm struct {
	TTS AudioTTSConfigForm `json:"tts"`
	STT AudioSTTConfigForm `json:"stt"`
}

// AudioTTSConfigForm mirrors TTSConfigForm in
// backend/open_webui/routers/audio.py.
type AudioTTSConfigForm struct {
	OpenAIAPIBaseURL        *string `json:"OPENAI_API_BASE_URL,omitempty"`
	OpenAIAPIKey            *string `json:"OPENAI_API_KEY,omitempty"`
	OpenAIParams            any     `json:"OPENAI_PARAMS,omitempty"`
	APIKey                  *string `json:"API_KEY,omitempty"`
	Engine                  *string `json:"ENGINE,omitempty"`
	Model                   *string `json:"MODEL,omitempty"`
	Voice                   *string `json:"VOICE,omitempty"`
	SplitOn                 *string `json:"SPLIT_ON,omitempty"`
	AzureSpeechRegion       *string `json:"AZURE_SPEECH_REGION,omitempty"`
	AzureSpeechBaseURL      *string `json:"AZURE_SPEECH_BASE_URL,omitempty"`
	AzureSpeechOutputFormat *string `json:"AZURE_SPEECH_OUTPUT_FORMAT,omitempty"`
	MistralAPIKey           *string `json:"MISTRAL_API_KEY,omitempty"`
	MistralAPIBaseURL       *string `json:"MISTRAL_API_BASE_URL,omitempty"`
}

// AudioSTTConfigForm mirrors STTConfigForm in
// backend/open_webui/routers/audio.py.
type AudioSTTConfigForm struct {
	OpenAIAPIBaseURL          *string   `json:"OPENAI_API_BASE_URL,omitempty"`
	OpenAIAPIKey              *string   `json:"OPENAI_API_KEY,omitempty"`
	OpenAIAPIRequestFormat    *string   `json:"OPENAI_API_REQUEST_FORMAT,omitempty"`
	Engine                    *string   `json:"ENGINE,omitempty"`
	Model                     *string   `json:"MODEL,omitempty"`
	SupportedContentTypes     *[]string `json:"SUPPORTED_CONTENT_TYPES,omitempty"`
	AllowedExtensions         *[]string `json:"ALLOWED_EXTENSIONS,omitempty"`
	WhisperModel              *string   `json:"WHISPER_MODEL,omitempty"`
	DeepgramAPIKey            *string   `json:"DEEPGRAM_API_KEY,omitempty"`
	AzureAPIKey               *string   `json:"AZURE_API_KEY,omitempty"`
	AzureRegion               *string   `json:"AZURE_REGION,omitempty"`
	AzureLocales              *string   `json:"AZURE_LOCALES,omitempty"`
	AzureBaseURL              *string   `json:"AZURE_BASE_URL,omitempty"`
	AzureMaxSpeakers          *string   `json:"AZURE_MAX_SPEAKERS,omitempty"`
	MistralAPIKey             *string   `json:"MISTRAL_API_KEY,omitempty"`
	MistralAPIBaseURL         *string   `json:"MISTRAL_API_BASE_URL,omitempty"`
	MistralUseChatCompletions *bool     `json:"MISTRAL_USE_CHAT_COMPLETIONS,omitempty"`
}

// GetAudioConfig retrieves the text to speech and speech to text config.
func (c *Client) GetAudioConfig(ctx context.Context) (*AudioConfigForm, error) {
	var resp AudioConfigForm
	if err := c.do(ctx, http.MethodGet, "audio/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetAudioConfig updates the text to speech and speech to text config. The
// route answers with the whole config, so the response is the new state.
func (c *Client) SetAudioConfig(ctx context.Context, form AudioConfigForm) (*AudioConfigForm, error) {
	var resp AudioConfigForm
	if err := c.do(ctx, http.MethodPost, "audio/config/update", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
